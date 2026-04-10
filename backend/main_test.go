package main

import (
	"math/rand"
	"strings"
	"testing"
)

func TestGenerateRoundProducesStableShape(t *testing.T) {
	server := &apiServer{rng: rand.New(rand.NewSource(7))}

	round, err := server.generateRound(roundRequest{
		Players: []string{"Avery", "Sam", "Riley"},
		Chores:  []string{"Laundry", "Dishes", "Trash"},
	})
	if err != nil {
		t.Fatalf("generateRound returned error: %v", err)
	}

	if len(round.SeedOrder) != 3 {
		t.Fatalf("expected 3 seeded players, got %d", len(round.SeedOrder))
	}
	if len(round.Matchups) != 1 {
		t.Fatalf("expected 1 matchup for 3 players, got %d", len(round.Matchups))
	}
	if round.Challenge.ID == "" {
		t.Fatal("expected a selected challenge")
	}
}

func TestGenerateRoundHonorsEnabledChallenges(t *testing.T) {
	server := &apiServer{rng: rand.New(rand.NewSource(7))}

	round, err := server.generateRound(roundRequest{
		Players:             []string{"Avery", "Sam", "Riley", "Jordan"},
		Chores:              []string{"Laundry", "Dishes", "Trash", "Bathroom"},
		EnabledChallengeIDs: []string{"mime-battle"},
	})
	if err != nil {
		t.Fatalf("generateRound returned error: %v", err)
	}

	if round.Challenge.ID != "mime-battle" {
		t.Fatalf("expected mime-battle, got %s", round.Challenge.ID)
	}
}

func TestGenerateRoundRejectsDisabledDeck(t *testing.T) {
	server := &apiServer{rng: rand.New(rand.NewSource(7))}

	_, err := server.generateRound(roundRequest{
		Players:             []string{"Avery", "Sam"},
		Chores:              []string{"Laundry"},
		EnabledChallengeIDs: []string{"missing-id"},
	})
	if err == nil {
		t.Fatal("expected generateRound to fail when all challenges are disabled")
	}
}

func TestBuildAssignmentsExemptsWinnerFromChores(t *testing.T) {
	server := &apiServer{rng: rand.New(rand.NewSource(11))}
	result, err := server.buildAssignments(assignmentRequest{
		Players:   []string{"Avery", "Sam", "Riley", "Jordan"},
		Chores:    []string{"Pick music", "Wipe counters", "Trash", "Bathroom"},
		SeedOrder: []string{"Avery", "Sam", "Riley", "Jordan"},
		Results: []result{
			{
				MatchupID: "match-1",
				Votes: []vote{
					{Voter: "Avery", Choice: "Sam"},
					{Voter: "Sam", Choice: "Sam"},
					{Voter: "Riley", Choice: "Sam"},
					{Voter: "Jordan", Choice: "Sam"},
				},
			},
			{
				MatchupID: "match-2",
				Votes: []vote{
					{Voter: "Avery", Choice: "Jordan"},
					{Voter: "Sam", Choice: "Jordan"},
					{Voter: "Riley", Choice: "Jordan"},
					{Voter: "Jordan", Choice: "Jordan"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("buildAssignments returned error: %v", err)
	}

	if result.Rankings[0].Player != "Sam" && result.Rankings[0].Player != "Jordan" {
		t.Fatalf("expected one of the tied winners to finish first, got %#v", result.Rankings[0])
	}
	if len(result.Assignments) != 5 {
		t.Fatalf("expected winner plus four fully assigned chores, got %d", len(result.Assignments))
	}
	if result.Assignments[0].Rank != 1 || result.Assignments[0].Chore != "Relax" {
		t.Fatalf("expected first place to be shown as Relax, got %#v", result.Assignments[0])
	}
	if result.Assignments[0].Player == result.Rankings[0].Player {
		if result.Assignments[0].Chore != "Relax" {
			t.Fatalf("expected winner %s to receive Relax, got assignments %#v", result.Rankings[0].Player, result.Assignments)
		}
	} else {
		t.Fatalf("expected winner %s to appear first in assignments, got %#v", result.Rankings[0].Player, result.Assignments)
	}
	if result.Assignments[1].Rank != 2 || result.Assignments[1].Chore != "Pick music" {
		t.Fatalf("expected second place to receive the first real chore, got %#v", result.Assignments[1])
	}
	if result.Assignments[3].Rank != 4 || result.Assignments[3].Chore != "Trash" {
		t.Fatalf("expected last place to receive the third chore, got %#v", result.Assignments[3])
	}
	if result.Assignments[4].Rank != 4 || result.Assignments[4].Chore != "Bathroom" {
		t.Fatalf("expected extra chore to roll back to last place first, got %#v", result.Assignments[4])
	}
	if result.Rankings[0].Score != 4 || result.Rankings[1].Score != 4 {
		t.Fatalf("expected top players to have 4 votes each, got %#v", result.Rankings[:2])
	}
}

func TestBuildAssignmentsDistributesExtraChoresFromLastToSecond(t *testing.T) {
	server := &apiServer{rng: rand.New(rand.NewSource(4))}
	result, err := server.buildAssignments(assignmentRequest{
		Players:   []string{"Avery", "Sam", "Riley", "Jordan"},
		Chores:    []string{"Laundry", "Dishes", "Trash", "Bathroom", "Sweep", "Mop"},
		SeedOrder: []string{"Avery", "Sam", "Riley", "Jordan"},
		Results: []result{
			{
				MatchupID: "match-1",
				Votes: []vote{
					{Voter: "Avery", Choice: "Sam"},
					{Voter: "Sam", Choice: "Sam"},
					{Voter: "Riley", Choice: "Sam"},
					{Voter: "Jordan", Choice: "Sam"},
				},
			},
			{
				MatchupID: "match-2",
				Votes: []vote{
					{Voter: "Avery", Choice: "Riley"},
					{Voter: "Sam", Choice: "Riley"},
					{Voter: "Riley", Choice: "Riley"},
					{Voter: "Jordan", Choice: "Jordan"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("buildAssignments returned error: %v", err)
	}

	if len(result.Assignments) != 7 {
		t.Fatalf("expected winner plus six fully assigned chores, got %d", len(result.Assignments))
	}

	expected := []assignment{
		{Player: "Sam", Chore: "Relax", Rank: 1},
		{Player: "Riley", Chore: "Laundry", Rank: 2},
		{Player: "Jordan", Chore: "Dishes", Rank: 3},
		{Player: "Avery", Chore: "Trash", Rank: 4},
		{Player: "Avery", Chore: "Bathroom", Rank: 4},
		{Player: "Jordan", Chore: "Sweep", Rank: 3},
		{Player: "Riley", Chore: "Mop", Rank: 2},
	}

	for index, assignment := range expected {
		if result.Assignments[index] != assignment {
			t.Fatalf("unexpected assignment at %d: got %#v want %#v", index, result.Assignments[index], assignment)
		}
	}

	if len(result.UnusedChores) != 0 {
		t.Fatalf("expected no unused chores, got %#v", result.UnusedChores)
	}
}

func TestGenerateRoundAddsMimeScenarioWhenSelected(t *testing.T) {
	server := &apiServer{rng: rand.New(rand.NewSource(0))}

	for range 200 {
		round, err := server.generateRound(roundRequest{
			Players: []string{"Avery", "Sam", "Riley", "Jordan"},
			Chores:  []string{"Laundry", "Dishes", "Trash", "Bathroom"},
		})
		if err != nil {
			t.Fatalf("generateRound returned error: %v", err)
		}

		if round.Challenge.ID != "mime-battle" {
			continue
		}

		if round.Challenge.Prompt == "" {
			t.Fatal("expected mime battle to include a scenario")
		}
		if round.Challenge.PromptLabel != "Mime scenario" {
			t.Fatalf("expected mime prompt label, got %q", round.Challenge.PromptLabel)
		}

		foundInstruction := false
		for _, instruction := range round.Instructions {
			if strings.Contains(instruction, round.Challenge.Prompt) {
				foundInstruction = true
				break
			}
		}
		if !foundInstruction {
			t.Fatal("expected mime scenario to be surfaced in instructions")
		}

		return
	}

	t.Fatal("mime battle was not selected during test run")
}

func TestBuildAssignmentsDoesNotAwardByePoints(t *testing.T) {
	server := &apiServer{rng: rand.New(rand.NewSource(3))}
	result, err := server.buildAssignments(assignmentRequest{
		Players:   []string{"Avery", "Sam", "Riley"},
		Chores:    []string{"Laundry", "Dishes", "Trash"},
		SeedOrder: []string{"Avery", "Sam", "Riley"},
		Results: []result{
			{
				MatchupID: "match-1",
				Votes: []vote{
					{Voter: "Avery", Choice: "Sam"},
					{Voter: "Sam", Choice: "Sam"},
					{Voter: "Riley", Choice: "Sam"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("buildAssignments returned error: %v", err)
	}

	for _, ranking := range result.Rankings {
		if ranking.Player == "Sam" && ranking.Score != 3 {
			t.Fatalf("expected winner to have 3 votes, got %d", ranking.Score)
		}
		if ranking.Player != "Sam" && ranking.Score != 0 {
			t.Fatalf("expected non-winners to have 0 points, got %#v", ranking)
		}
	}
}

func TestBuildAssignmentsRejectsInvalidVoteChoice(t *testing.T) {
	server := &apiServer{rng: rand.New(rand.NewSource(2))}
	_, err := server.buildAssignments(assignmentRequest{
		Players:   []string{"Avery", "Sam", "Riley", "Jordan"},
		Chores:    []string{"Laundry", "Dishes", "Trash", "Bathroom"},
		SeedOrder: []string{"Avery", "Sam", "Riley", "Jordan"},
		Results: []result{{
			MatchupID: "match-1",
			Votes: []vote{{Voter: "Avery", Choice: "Jordan"}},
		}},
	})
	if err == nil {
		t.Fatal("expected invalid vote choice to fail")
	}
}

func TestBuildAssignmentsRandomizesTieOrder(t *testing.T) {
	server := &apiServer{rng: rand.New(rand.NewSource(5))}
	result, err := server.buildAssignments(assignmentRequest{
		Players:   []string{"Avery", "Sam", "Riley", "Jordan"},
		Chores:    []string{"Laundry", "Dishes", "Trash", "Bathroom"},
		SeedOrder: []string{"Avery", "Sam", "Riley", "Jordan"},
		Results:   []result{},
	})
	if err != nil {
		t.Fatalf("buildAssignments returned error: %v", err)
	}

	if result.Rankings[0].Player == "Avery" && result.Rankings[1].Player == "Sam" && result.Rankings[2].Player == "Riley" && result.Rankings[3].Player == "Jordan" {
		t.Fatal("expected tied players to be shuffled instead of staying in seed order")
	}
}

func TestGenerateRoundAddsTongueTwisterWhenSelected(t *testing.T) {
	server := &apiServer{rng: rand.New(rand.NewSource(1))}

	for range 200 {
		round, err := server.generateRound(roundRequest{
			Players: []string{"Avery", "Sam", "Riley", "Jordan"},
			Chores:  []string{"Laundry", "Dishes", "Trash", "Bathroom"},
		})
		if err != nil {
			t.Fatalf("generateRound returned error: %v", err)
		}

		if round.Challenge.ID != "one-breath" {
			continue
		}

		if round.Challenge.Prompt == "" {
			t.Fatal("expected one-breath challenge to include a tongue twister")
		}
		if round.Challenge.PromptLabel != "Tongue twister" {
			t.Fatalf("expected tongue twister label, got %q", round.Challenge.PromptLabel)
		}

		foundInstruction := false
		for _, instruction := range round.Instructions {
			if strings.Contains(instruction, round.Challenge.Prompt) {
				foundInstruction = true
				break
			}
		}
		if !foundInstruction {
			t.Fatal("expected tongue twister to be surfaced in instructions")
		}

		return
	}

	t.Fatal("one-breath challenge was not selected during test run")
}

func TestGenerateRoundAddsTriviaQuestionWhenSelected(t *testing.T) {
	server := &apiServer{rng: rand.New(rand.NewSource(2))}

	for range 200 {
		round, err := server.generateRound(roundRequest{
			Players: []string{"Avery", "Sam", "Riley", "Jordan"},
			Chores:  []string{"Laundry", "Dishes", "Trash", "Bathroom"},
		})
		if err != nil {
			t.Fatalf("generateRound returned error: %v", err)
		}

		if round.Challenge.ID != "trivia-flash" {
			continue
		}

		if round.Challenge.Prompt == "" {
			t.Fatal("expected trivia flash to include a trivia question")
		}
		if round.Challenge.PromptLabel != "Trivia question" {
			t.Fatalf("expected trivia question label, got %q", round.Challenge.PromptLabel)
		}

		foundInstruction := false
		for _, instruction := range round.Instructions {
			if strings.Contains(instruction, round.Challenge.Prompt) {
				foundInstruction = true
				break
			}
		}
		if !foundInstruction {
			t.Fatal("expected trivia question to be surfaced in instructions")
		}

		return
	}

	t.Fatal("trivia-flash challenge was not selected during test run")
}

func TestLoadPromptLinesSkipsCommentsAndBlankLines(t *testing.T) {
	prompts := loadPromptLines("# comment\n\nfirst\n second \n")
	if len(prompts) != 2 {
		t.Fatalf("expected 2 prompts, got %d", len(prompts))
	}
	if prompts[0] != "first" || prompts[1] != "second" {
		t.Fatalf("unexpected prompts: %#v", prompts)
	}
}