package command

import (
	"context"
	"math"
	"strings"
	"testing"

	typesafe "github.com/haileyok/typesafe-client/go"
)

func TestOpinionAnalysis(t *testing.T) {
	for _, tt := range []struct {
		name, input, parent, wantState string
	}{
		{name: "inline", input: "I don't think Taylor Swift is that good", wantState: "I don't think Taylor Swift is that good"},
		{name: "reply", parent: "I don't think Taylor Swift is that good", wantState: "I don't think Taylor Swift is that good"},
		{name: "reply with prompt", input: "what do you think about this?", parent: "I don't think Taylor Swift is that good", wantState: "The user replied with: what do you think about this?\n\nThe post to analyze is: I don't think Taylor Swift is that good"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := clientFunc(func(_ context.Context, evaluation typesafe.Request) (*typesafe.Response, error) {
				if evaluation.State != tt.wantState {
					t.Fatalf("state = %q, want %q", evaluation.State, tt.wantState)
				}
				verdictQuestion, ok := evaluation.Questions["verdict"].(typesafe.ChoiceQuestion)
				if !ok || len(verdictQuestion.Criteria) != 3 {
					t.Fatalf("verdict question = %+v", evaluation.Questions["verdict"])
				}
				heatQuestion, ok := evaluation.Questions["spiciness"].(typesafe.ScoreQuestion)
				if !ok || len(heatQuestion.Criteria) != len(opinionHeatLevels) {
					t.Fatalf("spiciness question = %+v", evaluation.Questions["spiciness"])
				}
				return &typesafe.Response{
					Model: "jev-test", RequestID: "opinion-123",
					Answers: map[string]typesafe.Answer{
						"verdict":   typesafe.ChoiceAnswer{Choice: "right", Confidence: 0.8},
						"spiciness": typesafe.ScoreAnswer{Score: 6.3},
					},
				}, nil
			})
			reply, err := opinionCommand(client, testLogger()).Handle(context.Background(), Request{Input: tt.input, ParentContent: tt.parent})
			if err != nil {
				t.Fatal(err)
			}
			if reply != "Verdict: right.\n\nHeat: 7.3/10 (spicy)." {
				t.Fatalf("reply = %q", reply)
			}
		})
	}
}

func TestOpinionMissingState(t *testing.T) {
	called := false
	client := clientFunc(func(context.Context, typesafe.Request) (*typesafe.Response, error) {
		called = true
		return nil, nil
	})
	reply, err := opinionCommand(client, testLogger()).Handle(context.Background(), Request{Input: " \t", ParentContent: "\n"})
	if err != nil || called || !strings.Contains(reply, "reply to one") {
		t.Fatalf("missing opinion = %q, err=%v, called=%v", reply, err, called)
	}
}

func TestOpinionRejectsInvalidAnswers(t *testing.T) {
	for _, answer := range []typesafe.Answer{
		typesafe.ChoiceAnswer{Choice: "maybe"},
		typesafe.ChoiceAnswer{Choice: "right"},
	} {
		t.Run(answer.Type(), func(t *testing.T) {
			client := clientFunc(func(context.Context, typesafe.Request) (*typesafe.Response, error) {
				return &typesafe.Response{Answers: map[string]typesafe.Answer{
					"verdict":   answer,
					"spiciness": typesafe.ScoreAnswer{Score: math.NaN()},
				}}, nil
			})
			reply, err := opinionCommand(client, testLogger()).Handle(context.Background(), Request{Input: "this is a take"})
			if err == nil || reply != "" || !strings.HasPrefix(err.Error(), "evaluate opinion:") {
				t.Fatalf("reply=%q err=%v", reply, err)
			}
		})
	}
}

func TestOpinionHeatLabels(t *testing.T) {
	for _, tt := range []struct {
		heat float64
		want string
	}{
		{1, "mild"}, {3, "warm"}, {5, "hot"}, {7, "spicy"}, {9, "nuclear"},
	} {
		if got := opinionHeatLabel(tt.heat); got != tt.want {
			t.Errorf("opinionHeatLabel(%v) = %q, want %q", tt.heat, got, tt.want)
		}
	}
}
