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

func TestBuildAssignmentsRanksWinnersFirst(t *testing.T) {
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

	winners := map[string]bool{
		result.Assignments[0].Player: true,
		result.Assignments[1].Player: true,
	}
	if !winners["Sam"] || !winners["Jordan"] {
		t.Fatalf("expected Sam and Jordan to occupy the top two slots, got %#v", result.Assignments[:2])
	}
	if result.Assignments[3].Chore != "Bathroom" {
		t.Fatalf("expected final chore to remain aligned, got %s", result.Assignments[3].Chore)
	}
	if result.Rankings[0].Score != 4 || result.Rankings[1].Score != 4 {
		t.Fatalf("expected top players to have 4 votes each, got %#v", result.Rankings[:2])
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

func TestLoadPromptLinesSkipsCommentsAndBlankLines(t *testing.T) {
	prompts := loadPromptLines("# comment\n\nfirst\n second \n")
	if len(prompts) != 2 {
		t.Fatalf("expected 2 prompts, got %d", len(prompts))
	}
	if prompts[0] != "first" || prompts[1] != "second" {
		t.Fatalf("unexpected prompts: %#v", prompts)
	}
}