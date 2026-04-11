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
	if round.Challenge.ID == "" {
		t.Fatal("expected a selected challenge")
	}
	if len(round.Instructions) == 0 {
		t.Fatal("expected instructions for the scoring round")
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

func TestBuildAssignmentsShowsWinnerAsRelax(t *testing.T) {
	server := &apiServer{rng: rand.New(rand.NewSource(11))}
	result, err := server.buildAssignments(assignmentRequest{
		Players:   []string{"Avery", "Sam", "Riley", "Jordan"},
		Chores:    []string{"Pick music", "Wipe counters", "Trash", "Bathroom"},
		SeedOrder: []string{"Avery", "Sam", "Riley", "Jordan"},
		Scores: []scoreEntry{
			{Voter: "Avery", Player: "Sam", Score: 5},
			{Voter: "Avery", Player: "Riley", Score: 3},
			{Voter: "Avery", Player: "Jordan", Score: 1},
			{Voter: "Sam", Player: "Avery", Score: 4},
			{Voter: "Sam", Player: "Riley", Score: 2},
			{Voter: "Sam", Player: "Jordan", Score: 1},
			{Voter: "Riley", Player: "Sam", Score: 5},
			{Voter: "Riley", Player: "Avery", Score: 2},
			{Voter: "Riley", Player: "Jordan", Score: 1},
			{Voter: "Jordan", Player: "Sam", Score: 4},
			{Voter: "Jordan", Player: "Avery", Score: 3},
			{Voter: "Jordan", Player: "Riley", Score: 2},
		},
	})
	if err != nil {
		t.Fatalf("buildAssignments returned error: %v", err)
	}

	if result.Rankings[0].Player != "Sam" || result.Rankings[0].Score != 14 {
		t.Fatalf("expected Sam to win with 14 points, got %#v", result.Rankings[0])
	}
	if len(result.Assignments) != 5 {
		t.Fatalf("expected winner plus four assigned entries, got %d", len(result.Assignments))
	}
	if result.Assignments[0] != (assignment{Player: "Sam", Chore: "Relax", Rank: 1}) {
		t.Fatalf("expected Sam to receive Relax, got %#v", result.Assignments[0])
	}
	if result.Assignments[1] != (assignment{Player: "Avery", Chore: "Pick music", Rank: 2}) {
		t.Fatalf("expected second place to receive first chore, got %#v", result.Assignments[1])
	}
	if result.Assignments[4] != (assignment{Player: "Jordan", Chore: "Bathroom", Rank: 4}) {
		t.Fatalf("expected extra chore to roll back to last place first, got %#v", result.Assignments[4])
	}
}

func TestBuildAssignmentsDistributesExtraChoresFromLastToSecond(t *testing.T) {
	server := &apiServer{rng: rand.New(rand.NewSource(4))}
	result, err := server.buildAssignments(assignmentRequest{
		Players:   []string{"Avery", "Sam", "Riley", "Jordan"},
		Chores:    []string{"Laundry", "Dishes", "Trash", "Bathroom", "Sweep", "Mop"},
		SeedOrder: []string{"Avery", "Sam", "Riley", "Jordan"},
		Scores: []scoreEntry{
			{Voter: "Avery", Player: "Sam", Score: 5},
			{Voter: "Avery", Player: "Riley", Score: 3},
			{Voter: "Avery", Player: "Jordan", Score: 1},
			{Voter: "Sam", Player: "Avery", Score: 4},
			{Voter: "Sam", Player: "Riley", Score: 2},
			{Voter: "Sam", Player: "Jordan", Score: 1},
			{Voter: "Riley", Player: "Sam", Score: 5},
			{Voter: "Riley", Player: "Avery", Score: 2},
			{Voter: "Riley", Player: "Jordan", Score: 1},
			{Voter: "Jordan", Player: "Sam", Score: 4},
			{Voter: "Jordan", Player: "Avery", Score: 3},
			{Voter: "Jordan", Player: "Riley", Score: 2},
		},
	})
	if err != nil {
		t.Fatalf("buildAssignments returned error: %v", err)
	}

	expected := []assignment{
		{Player: "Sam", Chore: "Relax", Rank: 1},
		{Player: "Avery", Chore: "Laundry", Rank: 2},
		{Player: "Riley", Chore: "Dishes", Rank: 3},
		{Player: "Jordan", Chore: "Trash", Rank: 4},
		{Player: "Jordan", Chore: "Bathroom", Rank: 4},
		{Player: "Riley", Chore: "Sweep", Rank: 3},
		{Player: "Avery", Chore: "Mop", Rank: 2},
	}

	if len(result.Assignments) != len(expected) {
		t.Fatalf("expected %d assignments, got %d", len(expected), len(result.Assignments))
	}
	for index, expectedAssignment := range expected {
		if result.Assignments[index] != expectedAssignment {
			t.Fatalf("unexpected assignment at %d: got %#v want %#v", index, result.Assignments[index], expectedAssignment)
		}
	}
	if len(result.UnusedChores) != 0 {
		t.Fatalf("expected no unused chores, got %#v", result.UnusedChores)
	}
}

func TestBuildAssignmentsRejectsInvalidScoreValue(t *testing.T) {
	server := &apiServer{rng: rand.New(rand.NewSource(2))}
	_, err := server.buildAssignments(assignmentRequest{
		Players:   []string{"Avery", "Sam", "Riley", "Jordan"},
		Chores:    []string{"Laundry", "Dishes", "Trash", "Bathroom"},
		SeedOrder: []string{"Avery", "Sam", "Riley", "Jordan"},
		Scores:    []scoreEntry{{Voter: "Avery", Player: "Sam", Score: 6}},
	})
	if err == nil {
		t.Fatal("expected invalid score value to fail")
	}
}

func TestBuildAssignmentsRejectsPlayerScoringSelf(t *testing.T) {
	server := &apiServer{rng: rand.New(rand.NewSource(2))}
	_, err := server.buildAssignments(assignmentRequest{
		Players:   []string{"Avery", "Sam", "Riley", "Jordan"},
		Chores:    []string{"Laundry", "Dishes", "Trash", "Bathroom"},
		SeedOrder: []string{"Avery", "Sam", "Riley", "Jordan"},
		Scores:    []scoreEntry{{Voter: "Avery", Player: "Avery", Score: 5}},
	})
	if err == nil {
		t.Fatal("expected self scoring to fail")
	}
}

func TestBuildAssignmentsRandomizesTieOrder(t *testing.T) {
	server := &apiServer{rng: rand.New(rand.NewSource(5))}
	result, err := server.buildAssignments(assignmentRequest{
		Players:   []string{"Avery", "Sam", "Riley", "Jordan"},
		Chores:    []string{"Laundry", "Dishes", "Trash", "Bathroom"},
		SeedOrder: []string{"Avery", "Sam", "Riley", "Jordan"},
		Scores:    []scoreEntry{},
	})
	if err != nil {
		t.Fatalf("buildAssignments returned error: %v", err)
	}

	if result.Rankings[0].Player == "Avery" && result.Rankings[1].Player == "Sam" && result.Rankings[2].Player == "Riley" && result.Rankings[3].Player == "Jordan" {
		t.Fatal("expected tied players to be shuffled instead of staying in seed order")
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
