package command

import (
	"context"
	"math"
	"strconv"
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
				agreementQuestion, ok := evaluation.Questions["agreement"].(typesafe.ScoreQuestion)
				if !ok || len(agreementQuestion.Criteria) != len(opinionAgreementLevels) {
					t.Fatalf("agreement question = %+v", evaluation.Questions["agreement"])
				}
				if _, ok := evaluation.Questions["spiciness"]; ok {
					t.Fatal("opinion evaluation must not rate spiciness")
				}
				return &typesafe.Response{
					Model: "jev-test", RequestID: "opinion-123",
					Answers: map[string]typesafe.Answer{
						"agreement": typesafe.ScoreAnswer{Score: 6.3},
					},
				}, nil
			})
			reply, err := opinionCommand(client, testLogger()).Handle(context.Background(), Request{Input: tt.input, ParentContent: tt.parent})
			if err != nil {
				t.Fatal(err)
			}
			if reply != "i slightly disagree - 3.7/10" {
				t.Fatalf("reply = %q", reply)
			}
		})
	}
}

func TestOpinionAgreementScoreIsInverted(t *testing.T) {
	for _, tt := range []struct {
		name  string
		score float64
		want  string
	}{
		{name: "minimum", score: 0, want: "i strongly agree - 10.0/10"},
		{name: "fractional low score", score: 0.12, want: "i strongly agree - 9.9/10"},
		{name: "middle", score: 4, want: "i slightly agree - 6.0/10"},
		{name: "maximum", score: 9, want: "i strongly disagree - 1.0/10"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := clientFunc(func(context.Context, typesafe.Request) (*typesafe.Response, error) {
				return &typesafe.Response{Answers: map[string]typesafe.Answer{
					"agreement": typesafe.ScoreAnswer{Score: tt.score},
				}}, nil
			})
			reply, err := opinionCommand(client, testLogger()).Handle(context.Background(), Request{
				Input: "I don't think this is a good idea",
			})
			if err != nil || reply != tt.want {
				t.Fatalf("reply = %q, want %q, err=%v", reply, tt.want, err)
			}
		})
	}
}

func TestOpinionAlwaysAgreeOverride(t *testing.T) {
	clientCalls := 0
	client := clientFunc(func(context.Context, typesafe.Request) (*typesafe.Response, error) {
		clientCalls++
		return nil, nil
	})
	for range 100 {
		reply, err := opinionCommand(client, testLogger()).Handle(context.Background(), Request{
			AuthorID: "314389700179918850",
			Input:    "I completely disagree with this opinion",
		})
		if err != nil {
			t.Fatal(err)
		}
		separator := strings.LastIndex(reply, " - ")
		if separator < 0 {
			t.Fatalf("reply = %q, want an agreement label and score", reply)
		}
		label := reply[:separator]
		if label != "i agree" && label != "i strongly agree" {
			t.Fatalf("reply = %q, want an agreeing label", reply)
		}
		score, err := strconv.ParseFloat(strings.TrimSuffix(reply[separator+3:], "/10"), 64)
		if err != nil || score < 6.9 || score > 9.7 {
			t.Fatalf("reply = %q, score must be between 6.9 and 9.7 (err=%v)", reply, err)
		}
	}
	if clientCalls != 0 {
		t.Fatalf("TypeSafe called %d times, want 0", clientCalls)
	}
}

func TestOpinionOverrideLeavesOtherUsersModelEvaluated(t *testing.T) {
	clientCalled := false
	client := clientFunc(func(context.Context, typesafe.Request) (*typesafe.Response, error) {
		clientCalled = true
		return &typesafe.Response{Answers: map[string]typesafe.Answer{
			"agreement": typesafe.ScoreAnswer{Score: 0},
		}}, nil
	})
	reply, err := opinionCommand(client, testLogger()).Handle(context.Background(), Request{
		AuthorID: "314389700179918851",
		Input:    "I agree with this opinion",
	})
	if err != nil || reply != "i strongly agree - 10.0/10" || !clientCalled {
		t.Fatalf("reply = %q, err=%v, clientCalled=%v", reply, err, clientCalled)
	}
}

func TestHotTakeAnalysis(t *testing.T) {
	client := clientFunc(func(_ context.Context, evaluation typesafe.Request) (*typesafe.Response, error) {
		if evaluation.State != "I don't think Taylor Swift is that good" {
			t.Fatalf("state = %q", evaluation.State)
		}
		spiciness, ok := evaluation.Questions["spiciness"].(typesafe.ScoreQuestion)
		if !ok || len(spiciness.Criteria) != len(hotTakeHeatLevels) {
			t.Fatalf("spiciness question = %+v", evaluation.Questions["spiciness"])
		}
		if _, ok := evaluation.Questions["agreement"]; ok {
			t.Fatal("hot take evaluation must not rate agreement")
		}
		return &typesafe.Response{Answers: map[string]typesafe.Answer{
			"spiciness": typesafe.ScoreAnswer{Score: 7.3},
		}}, nil
	})
	reply, err := hotTakeCommand(client, testLogger()).Handle(context.Background(), Request{Input: "/hot take I don't think Taylor Swift is that good"})
	if err != nil || reply != "spicy take - 8.3/10" {
		t.Fatalf("reply = %q, err=%v", reply, err)
	}
}

func TestOpinionMissingState(t *testing.T) {
	called := false
	client := clientFunc(func(context.Context, typesafe.Request) (*typesafe.Response, error) {
		called = true
		return nil, nil
	})
	reply, err := opinionCommand(client, testLogger()).Handle(context.Background(), Request{Input: " \t", ParentContent: "\n"})
	if err != nil || called || reply != "give me an opinion to judge, or reply to one and tag me." {
		t.Fatalf("missing opinion = %q, err=%v, called=%v", reply, err, called)
	}
	reply, err = hotTakeCommand(client, testLogger()).Handle(context.Background(), Request{Input: "/hot take"})
	if err != nil || called || reply != "give me a take to rate, or reply to one and tag me." {
		t.Fatalf("missing hot take = %q, err=%v, called=%v", reply, err, called)
	}
}

func TestOpinionRejectsInvalidAnswers(t *testing.T) {
	for _, answer := range []typesafe.Answer{
		typesafe.ScoreAnswer{Score: math.NaN()},
		typesafe.ChoiceAnswer{Choice: "right"},
	} {
		t.Run(answer.Type(), func(t *testing.T) {
			client := clientFunc(func(context.Context, typesafe.Request) (*typesafe.Response, error) {
				return &typesafe.Response{Answers: map[string]typesafe.Answer{"agreement": answer}}, nil
			})
			reply, err := opinionCommand(client, testLogger()).Handle(context.Background(), Request{Input: "this is a take"})
			if err == nil || reply != "" || !strings.HasPrefix(err.Error(), "evaluate opinion: invalid agreement") {
				t.Fatalf("reply=%q err=%v", reply, err)
			}
		})
	}
}

func TestHotTakeRejectsInvalidAnswers(t *testing.T) {
	client := clientFunc(func(context.Context, typesafe.Request) (*typesafe.Response, error) {
		return &typesafe.Response{Answers: map[string]typesafe.Answer{"spiciness": typesafe.ScoreAnswer{Score: math.NaN()}}}, nil
	})
	reply, err := hotTakeCommand(client, testLogger()).Handle(context.Background(), Request{Input: "/hot take this is a take"})
	if err == nil || reply != "" || !strings.HasPrefix(err.Error(), "evaluate hot take: invalid spiciness") {
		t.Fatalf("reply=%q err=%v", reply, err)
	}
}

func TestHotTakeLabels(t *testing.T) {
	for _, tt := range []struct {
		heat float64
		want string
	}{
		{1, "mild take"}, {3, "warm take"}, {5, "hot take"}, {7, "spicy take"}, {9, "nuclear take"},
	} {
		if got := hotTakeLabel(tt.heat); got != tt.want {
			t.Errorf("hotTakeLabel(%v) = %q, want %q", tt.heat, got, tt.want)
		}
	}
}

func TestOpinionAgreementLabels(t *testing.T) {
	for _, tt := range []struct {
		agreement float64
		want      string
	}{
		{1, "strongly disagree"}, {3, "disagree"}, {4, "slightly disagree"}, {5, "undecided"},
		{6, "slightly agree"}, {7, "agree"}, {9, "strongly agree"},
	} {
		if got := opinionAgreementLabel(tt.agreement); got != tt.want {
			t.Errorf("opinionAgreementLabel(%v) = %q, want %q", tt.agreement, got, tt.want)
		}
	}
}

func TestFormattedOpinionAndHotTakeRepliesAreLowercase(t *testing.T) {
	for _, reply := range []string{formatOpinionReply(9.3), formatHotTakeReply(8.3)} {
		if reply != strings.ToLower(reply) {
			t.Errorf("reply = %q, must be lowercase", reply)
		}
	}
}
