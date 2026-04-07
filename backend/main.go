package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	Players []string `json:"players"`
	Chores  []string `json:"chores"`
}

type roundResponse struct {
	Challenge    challenge `json:"challenge"`
	Matchups     []matchup `json:"matchups"`
	SeedOrder    []string  `json:"seedOrder"`
	Instructions []string  `json:"instructions"`
}

type vote struct {
	Voter  string `json:"voter"`
	Choice string `json:"choice"`
}

type result struct {
	MatchupID string `json:"matchupId"`
	Votes     []vote `json:"votes"`
}

type assignmentRequest struct {
	Players   []string `json:"players"`
	Chores    []string `json:"chores"`
	SeedOrder []string `json:"seedOrder"`
	Results   []result `json:"results"`
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
		Setup:           []string{"Phone or host to read questions"},
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

func main() {
	seed := time.Now().UnixNano()
	server := &apiServer{rng: rand.New(rand.NewSource(seed))}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", server.handleHealth)
	mux.HandleFunc("/api/challenges", server.handleChallenges)
	mux.HandleFunc("/api/rounds", server.handleRounds)
	mux.HandleFunc("/api/assignments", server.handleAssignments)

	port := os.Getenv("PORT")
	if strings.TrimSpace(port) == "" {
		port = "8080"
	}

	log.Printf("challenge server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, withCORS(mux)); err != nil {
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

func (s *apiServer) generateRound(req roundRequest) (roundResponse, error) {
	players, err := normalizeUniqueValues(req.Players, "player")
	if err != nil {
		return roundResponse{}, err
	}
	if _, err := normalizeValues(req.Chores, "chore"); err != nil {
		return roundResponse{}, err
	}

	seedOrder := append([]string(nil), players...)
	s.shuffle(len(seedOrder), func(i, j int) {
		seedOrder[i], seedOrder[j] = seedOrder[j], seedOrder[i]
	})

	matchups := make([]matchup, 0, len(seedOrder)/2)
	for i := 0; i+1 < len(seedOrder); i += 2 {
		matchups = append(matchups, matchup{
			ID:        "match-" + strconvItoa(len(matchups)+1),
			PlayerOne: seedOrder[i],
			PlayerTwo: seedOrder[i+1],
		})
	}

	chosenChallenge := challengeCatalog[s.intn(len(challengeCatalog))]
	applyChallengePrompt(s, &chosenChallenge)
	instructions := []string{
		"Enter chores in the order you want them awarded. Top entry goes to the best performer.",
		"Run the selected challenge once for each matchup and record everyone's votes.",
		"Players are ranked only by total votes received in this round.",
		"Any tie in votes is broken randomly before chores are assigned.",
	}
	if chosenChallenge.Prompt != "" {
		instructions = append(instructions, chosenChallenge.PromptLabel+": "+chosenChallenge.Prompt)
	}
	if len(seedOrder)%2 == 1 {
		instructions = append(instructions, seedOrder[len(seedOrder)-1]+" sits out this round and gets no automatic points.")
	}

	return roundResponse{
		Challenge:    chosenChallenge,
		Matchups:     matchups,
		SeedOrder:    seedOrder,
		Instructions: instructions,
	}, nil
}

func applyChallengePrompt(server *apiServer, selected *challenge) {
	switch selected.ID {
	case "mime-battle":
		selected.PromptLabel = "Mime scenario"
		selected.Prompt = randomPrompt(server, mimeBattleScenarios)
	case "one-breath":
		selected.PromptLabel = "Tongue twister"
		selected.Prompt = randomPrompt(server, tongueTwisters)
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

	expectedMatchups := buildMatchups(seedOrder)
	expectedMatchupByID := make(map[string]matchup, len(expectedMatchups))
	for _, currentMatchup := range expectedMatchups {
		expectedMatchupByID[currentMatchup.ID] = currentMatchup
	}

	voteTotals := make(map[string]int, len(players))
	for _, entry := range req.Results {
		currentMatchup, ok := expectedMatchupByID[entry.MatchupID]
		if !ok {
			return assignmentResponse{}, errors.New("results reference an unknown matchup")
		}

		seenVoters := make(map[string]bool, len(entry.Votes))
		for _, currentVote := range entry.Votes {
			voter := strings.TrimSpace(currentVote.Voter)
			choice := strings.TrimSpace(currentVote.Choice)
			if voter == "" || choice == "" {
				continue
			}
			if _, ok := seedIndex[voter]; !ok {
				return assignmentResponse{}, errors.New("votes reference an unknown voter")
			}
			if seenVoters[voter] {
				return assignmentResponse{}, errors.New("each player can only vote once per matchup")
			}
			if choice != currentMatchup.PlayerOne && choice != currentMatchup.PlayerTwo {
				return assignmentResponse{}, errors.New("votes must target one of the matchup players")
			}

			seenVoters[voter] = true
			voteTotals[choice]++
		}
	}

	rankings := make([]ranking, 0, len(players))
	for _, player := range players {
		rankings = append(rankings, ranking{
			Player: player,
			Score:  voteTotals[player],
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

	assignments := make([]assignment, 0, min(len(rankings), len(chores)))
	for index, playerRanking := range rankings {
		if index >= len(chores) {
			break
		}
		assignments = append(assignments, assignment{
			Player: playerRanking.Player,
			Chore:  chores[index],
			Rank:   index + 1,
		})
	}

	unusedChores := []string{}
	if len(chores) > len(rankings) {
		unusedChores = append(unusedChores, chores[len(rankings):]...)
	}

	unassignedPlayers := []string{}
	if len(rankings) > len(chores) {
		for _, playerRanking := range rankings[len(chores):] {
			unassignedPlayers = append(unassignedPlayers, playerRanking.Player)
		}
	}

	summary := "Higher-ranked players are matched to earlier chores in the list you entered. Ties in votes are resolved randomly."
	if len(unusedChores) > 0 {
		summary = summary + " Extra chores stay unassigned for a later round."
	}
	if len(unassignedPlayers) > 0 {
		summary = summary + " Some players were left without chores because you listed fewer chores than players."
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