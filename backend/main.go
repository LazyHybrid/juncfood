package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type challenge struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	DurationSeconds int      `json:"durationSeconds"`
	Setup           []string `json:"setup"`
	WinCondition    string   `json:"winCondition"`
	PromptLabel     string   `json:"promptLabel,omitempty"`
	Prompt          string   `json:"prompt,omitempty"`
}

type matchup struct {
	ID        string `json:"id"`
	PlayerOne string `json:"playerOne"`
	PlayerTwo string `json:"playerTwo"`
}

type roundRequest struct {
	Players             []string `json:"players"`
	Chores              []string `json:"chores"`
	EnabledChallengeIDs []string `json:"enabledChallengeIds"`
}

type roundResponse struct {
	Challenge    challenge `json:"challenge"`
	SeedOrder    []string  `json:"seedOrder"`
	Instructions []string  `json:"instructions"`
}

type scoreEntry struct {
	Voter  string `json:"voter"`
	Player string `json:"player"`
	Score  int    `json:"score"`
}

type assignmentRequest struct {
	Players   []string `json:"players"`
	Chores    []string `json:"chores"`
	SeedOrder []string `json:"seedOrder"`
	Scores    []scoreEntry `json:"scores"`
}

type ranking struct {
	Player string `json:"player"`
	Score  int    `json:"score"`
	Seed   int    `json:"seed"`
}

type assignment struct {
	Player string `json:"player"`
	Chore  string `json:"chore"`
	Rank   int    `json:"rank"`
}

type assignmentResponse struct {
	Rankings          []ranking    `json:"rankings"`
	Assignments       []assignment `json:"assignments"`
	UnassignedPlayers []string     `json:"unassignedPlayers"`
	UnusedChores      []string     `json:"unusedChores"`
	Summary           string       `json:"summary"`
}

type apiServer struct {
	mu  sync.Mutex
	rng *rand.Rand
}

//go:embed data/mime_scenarios.txt
var mimeScenarioData string

//go:embed data/tongue_twisters.txt
var tongueTwisterData string

//go:embed data/trivia_questions.txt
var triviaQuestionData string

//go:embed web/* web/assets/*
var frontendBuild embed.FS

var challengeCatalog = []challenge{
	{
		ID:              "speed-stack",
		Title:           "Speed Stack Showdown",
		Description:     "Stack and unstack 10 cups or books faster than your opponent.",
		DurationSeconds: 45,
		Setup:           []string{"10 cups, books, or any stackable objects", "A flat surface"},
		WinCondition:    "Fastest clean stack cycle wins first pick.",
	},
	{
		ID:              "mime-battle",
		Title:           "Silent Mime Battle",
		Description:     "Each player acts out the same prompt without speaking. The room votes for the clearer performance.",
		DurationSeconds: 60,
		Setup:           []string{"One prompt from the host", "A judge or quick vote"},
		WinCondition:    "Funniest or clearest mime takes the win.",
	},
	{
		ID:              "paper-plane",
		Title:           "Paper Plane Precision",
		Description:     "Fold one plane and try to land it closest to a target.",
		DurationSeconds: 90,
		Setup:           []string{"Scrap paper", "Tape or coaster as a landing zone"},
		WinCondition:    "Closest landing wins the matchup.",
	},
	{
		ID:              "one-breath",
		Title:           "One-Breath Tongue Twister",
		Description:     "Recite a tongue twister in a single breath without stumbling.",
		DurationSeconds: 30,
		Setup:           []string{"Any tongue twister chosen by the host"},
		WinCondition:    "Cleanest single-breath run wins.",
	},
	{
		ID:              "coin-flick",
		Title:           "Coin Flick Accuracy",
		Description:     "Each player gets three flicks toward a cup or target marker.",
		DurationSeconds: 45,
		Setup:           []string{"Coins, buttons, or bottle caps", "A mug or tape target"},
		WinCondition:    "Most hits or closest average distance wins.",
	},
	{
		ID:              "trivia-flash",
		Title:           "Trivia Flash Round",
		Description:     "Ask five fast trivia questions and keep score.",
		DurationSeconds: 60,
		Setup:           []string{"Use the trivia prompt shown on the round card", "Players answer as quickly as they can"},
		WinCondition:    "Most correct answers wins.",
	},
	{
		ID:              "sock-slide",
		Title:           "Sock Slide Sprint",
		Description:     "Slide a sock across the floor toward a marked finish line.",
		DurationSeconds: 30,
		Setup:           []string{"One sock", "Tape or chalk target line"},
		WinCondition:    "Closest slide to the line without passing it wins.",
	},
	{
		ID:              "draw-fast",
		Title:           "Draw It Fast",
		Description:     "Sketch the prompt in 20 seconds and let the room vote.",
		DurationSeconds: 45,
		Setup:           []string{"Paper", "Pens or pencils", "One prompt from the host"},
		WinCondition:    "Best recognizable drawing wins.",
	},
}

var mimeBattleScenarios = loadPromptLines(mimeScenarioData)

var tongueTwisters = loadPromptLines(tongueTwisterData)

var triviaQuestions = loadPromptLines(triviaQuestionData)

func main() {
	seed := time.Now().UnixNano()
	server := &apiServer{rng: rand.New(rand.NewSource(seed))}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", server.handleHealth)
	mux.HandleFunc("/api/challenges", server.handleChallenges)
	mux.HandleFunc("/api/rounds", server.handleRounds)
	mux.HandleFunc("/api/assignments", server.handleAssignments)
	mux.Handle("/", server.handleFrontend())

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	host := strings.TrimSpace(os.Getenv("HOST"))
	if host == "" {
		host = "127.0.0.1"
	}

	address := net.JoinHostPort(host, port)
	httpServer := &http.Server{
		Addr:    address,
		Handler: withCORS(mux),
	}

	shutdownCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	go func() {
		<-shutdownCtx.Done()

		gracefulCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(gracefulCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server shutdown error: %v", err)
		}
	}()

	log.Printf("challenge server listening on http://%s", address)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *apiServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *apiServer) handleChallenges(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, challengeCatalog)
}

func (s *apiServer) handleRounds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req roundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	round, err := s.generateRound(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, round)
}

func (s *apiServer) handleAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req assignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	assignmentResult, err := s.buildAssignments(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, assignmentResult)
}

func (s *apiServer) handleFrontend() http.Handler {
	frontendFS, err := fs.Sub(frontendBuild, "web")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, http.StatusInternalServerError, "frontend bundle unavailable")
		})
	}

	fileServer := http.FileServer(http.FS(frontendFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, http.StatusNotFound, "not found")
			return
		}

		requestedPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if requestedPath == "" || requestedPath == "." {
			requestedPath = "index.html"
		}

		if _, err := fs.Stat(frontendFS, requestedPath); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		indexRequest := r.Clone(r.Context())
		indexRequest.URL.Path = "/index.html"
		fileServer.ServeHTTP(w, indexRequest)
	})
}

func (s *apiServer) generateRound(req roundRequest) (roundResponse, error) {
	players, err := normalizeUniqueValues(req.Players, "player")
	if err != nil {
		return roundResponse{}, err
	}
	if _, err := normalizeValues(req.Chores, "chore"); err != nil {
		return roundResponse{}, err
	}
	availableChallenges, err := selectChallenges(req.EnabledChallengeIDs)
	if err != nil {
		return roundResponse{}, err
	}

	seedOrder := append([]string(nil), players...)
	s.shuffle(len(seedOrder), func(i, j int) {
		seedOrder[i], seedOrder[j] = seedOrder[j], seedOrder[i]
	})

	chosenChallenge := availableChallenges[s.intn(len(availableChallenges))]
	applyChallengePrompt(s, &chosenChallenge)
	instructions := []string{
		"Enter chores in the order you want them assigned after first place gets to Relax.",
		"Run one challenge round with all players participating.",
		"Each player gives every other player a score from 1 to 5.",
		"Players are ranked by total score received in this round.",
		"First place is shown as Relax, then remaining chores are assigned from the top of your list downward.",
		"Any tie in total score is broken randomly before chores are assigned.",
	}
	if chosenChallenge.Prompt != "" {
		instructions = append(instructions, chosenChallenge.PromptLabel+": "+chosenChallenge.Prompt)
	}

	return roundResponse{
		Challenge:    chosenChallenge,
		SeedOrder:    seedOrder,
		Instructions: instructions,
	}, nil
}

func selectChallenges(enabledIDs []string) ([]challenge, error) {
	if len(enabledIDs) == 0 {
		return append([]challenge(nil), challengeCatalog...), nil
	}

	allowed := make(map[string]bool, len(enabledIDs))
	for _, id := range enabledIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		allowed[trimmed] = true
	}

	selected := make([]challenge, 0, len(allowed))
	for _, currentChallenge := range challengeCatalog {
		if allowed[currentChallenge.ID] {
			selected = append(selected, currentChallenge)
		}
	}

	if len(selected) == 0 {
		return nil, errors.New("enable at least one challenge")
	}

	return selected, nil
}

func applyChallengePrompt(server *apiServer, selected *challenge) {
	switch selected.ID {
	case "mime-battle":
		selected.PromptLabel = "Mime scenario"
		selected.Prompt = randomPrompt(server, mimeBattleScenarios)
	case "one-breath":
		selected.PromptLabel = "Tongue twister"
		selected.Prompt = randomPrompt(server, tongueTwisters)
	case "trivia-flash":
		selected.PromptLabel = "Trivia question"
		selected.Prompt = randomPrompt(server, triviaQuestions)
	}
}

func randomPrompt(server *apiServer, prompts []string) string {
	if len(prompts) == 0 {
		return ""
	}

	return prompts[server.intn(len(prompts))]
}

func loadPromptLines(contents string) []string {
	lines := strings.Split(contents, "\n")
	prompts := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		prompts = append(prompts, trimmed)
	}

	return prompts
}

func (s *apiServer) buildAssignments(req assignmentRequest) (assignmentResponse, error) {
	players, err := normalizeUniqueValues(req.Players, "player")
	if err != nil {
		return assignmentResponse{}, err
	}
	chores, err := normalizeValues(req.Chores, "chore")
	if err != nil {
		return assignmentResponse{}, err
	}
	seedOrder, err := normalizeUniqueValues(req.SeedOrder, "seed entry")
	if err != nil {
		return assignmentResponse{}, err
	}
	if len(seedOrder) != len(players) {
		return assignmentResponse{}, errors.New("seed order must contain every player exactly once")
	}

	seedIndex := make(map[string]int, len(seedOrder))
	for index, player := range seedOrder {
		seedIndex[player] = index
	}
	for _, player := range players {
		if _, ok := seedIndex[player]; !ok {
			return assignmentResponse{}, errors.New("seed order must contain every player exactly once")
		}
	}

	scoreTotals := make(map[string]int, len(players))
	seenScores := make(map[string]bool, len(req.Scores))
	for _, currentScore := range req.Scores {
		voter := strings.TrimSpace(currentScore.Voter)
		player := strings.TrimSpace(currentScore.Player)
		if voter == "" || player == "" || currentScore.Score == 0 {
			continue
		}
		if _, ok := seedIndex[voter]; !ok {
			return assignmentResponse{}, errors.New("scores reference an unknown voter")
		}
		if _, ok := seedIndex[player]; !ok {
			return assignmentResponse{}, errors.New("scores reference an unknown player")
		}
		if voter == player {
			return assignmentResponse{}, errors.New("players cannot score themselves")
		}
		if currentScore.Score < 1 || currentScore.Score > 5 {
			return assignmentResponse{}, errors.New("scores must be between 1 and 5")
		}

		seenKey := voter + "::" + player
		if seenScores[seenKey] {
			return assignmentResponse{}, errors.New("each player can only score each player once")
		}

		seenScores[seenKey] = true
		scoreTotals[player] += currentScore.Score
	}

	rankings := make([]ranking, 0, len(players))
	for _, player := range players {
		rankings = append(rankings, ranking{
			Player: player,
			Score:  scoreTotals[player],
			Seed:   seedIndex[player] + 1,
		})
	}

	sort.SliceStable(rankings, func(i, j int) bool {
		return rankings[i].Score > rankings[j].Score
	})

	for start := 0; start < len(rankings); {
		end := start + 1
		for end < len(rankings) && rankings[end].Score == rankings[start].Score {
			end++
		}
		if end-start > 1 {
			s.shuffle(end-start, func(i, j int) {
				rankings[start+i], rankings[start+j] = rankings[start+j], rankings[start+i]
			})
		}
		start = end
	}

	rankPositionByPlayer := make(map[string]int, len(rankings))
	for index, playerRanking := range rankings {
		rankPositionByPlayer[playerRanking.Player] = index + 1
	}

	assignableRankings := rankings
	assignments := make([]assignment, 0, min(len(rankings), len(chores)+1))
	if len(assignableRankings) > 0 {
		assignments = append(assignments, assignment{
			Player: assignableRankings[0].Player,
			Chore:  "Relax",
			Rank:   1,
		})
		assignableRankings = assignableRankings[1:]
	}

	choreRecipients := make([]ranking, 0, len(chores))
	for _, playerRanking := range assignableRankings {
		if len(choreRecipients) >= len(chores) {
			break
		}
		choreRecipients = append(choreRecipients, playerRanking)
	}
	for len(choreRecipients) < len(chores) && len(assignableRankings) > 0 {
		for index := len(assignableRankings) - 1; index >= 0 && len(choreRecipients) < len(chores); index-- {
			choreRecipients = append(choreRecipients, assignableRankings[index])
		}
	}

	for index, playerRanking := range choreRecipients {
		assignments = append(assignments, assignment{
			Player: playerRanking.Player,
			Chore:  chores[index],
			Rank:   rankPositionByPlayer[playerRanking.Player],
		})
	}

	unusedChores := []string{}
	if len(choreRecipients) == 0 && len(chores) > 0 {
		unusedChores = append(unusedChores, chores...)
	}

	unassignedPlayers := []string{}
	if len(assignableRankings) > len(chores) {
		for _, playerRanking := range assignableRankings[len(chores):] {
			unassignedPlayers = append(unassignedPlayers, playerRanking.Player)
		}
	}

	summary := "First place is shown as Relax. Remaining players are matched to earlier chores in the list you entered, and any extra chores continue from last place back up to second place. Ties in total score are resolved randomly."
	if len(unusedChores) > 0 {
		summary = summary + " Extra chores stay unassigned for a later round."
	}
	if len(unassignedPlayers) > 0 {
		summary = summary + " Some players were left without chores because you listed fewer chores than remaining players."
	}

	return assignmentResponse{
		Rankings:          rankings,
		Assignments:       assignments,
		UnassignedPlayers: unassignedPlayers,
		UnusedChores:      unusedChores,
		Summary:           summary,
	}, nil
}

func (s *apiServer) intn(max int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rng.Intn(max)
}

func (s *apiServer) shuffle(length int, swap func(i, j int)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rng.Shuffle(length, swap)
}

func buildMatchups(seedOrder []string) []matchup {
	matchups := make([]matchup, 0, len(seedOrder)/2)
	for i := 0; i+1 < len(seedOrder); i += 2 {
		matchups = append(matchups, matchup{
			ID:        "match-" + strconvItoa(len(matchups)+1),
			PlayerOne: seedOrder[i],
			PlayerTwo: seedOrder[i+1],
		})
	}

	return matchups
}

func normalizeUniqueValues(values []string, label string) ([]string, error) {
	normalized, err := normalizeValues(values, label)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(normalized))
	for _, value := range normalized {
		key := strings.ToLower(value)
		if seen[key] {
			return nil, errors.New(label + " values must be unique")
		}
		seen[key] = true
	}

	return normalized, nil
}

func normalizeValues(values []string, label string) ([]string, error) {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	if len(cleaned) < 2 && label == "player" {
		return nil, errors.New("at least two players are required")
	}
	if len(cleaned) == 0 && label == "chore" {
		return nil, errors.New("add at least one chore")
	}
	if len(cleaned) == 0 && label == "seed entry" {
		return nil, errors.New("seed order is required")
	}
	return cleaned, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func strconvItoa(value int) string {
	return strconv.Itoa(value)
}