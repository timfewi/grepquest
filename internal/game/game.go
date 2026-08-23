package game

import (
	"context"
	"fmt"
	"time"

	"github.com/timfewi/grepquest/internal/challenge"
	"github.com/timfewi/grepquest/internal/judge"
)

type Profile struct {
	XP, Level int
	Solved    map[string]bool
	Mastery   map[string]int
	HintsUsed int
	Inventory map[string]bool
	Loadout   map[Slot]string
	Reviews   map[string]ReviewState
}

type ReviewState struct {
	Stage, Attempts, Successes int
	DueAt                      string
}

func NewProfile() Profile {
	return Profile{Level: 1, Solved: map[string]bool{}, Mastery: map[string]int{}, Inventory: map[string]bool{}, Loadout: map[Slot]string{}, Reviews: map[string]ReviewState{}}
}

type Result struct {
	judge.Verdict
	XPGained  int
	LeveledUp bool
	Drop      *Item
	Generated *GeneratedContent
	Review    bool
}

type Engine struct {
	Judge   judge.Judge
	Content ContentGenerator
	Now     func() time.Time
}

func (e Engine) Submit(ctx context.Context, p *Profile, c challenge.Challenge, command string, hintsUsed int) (Result, error) {
	if p == nil {
		return Result{}, fmt.Errorf("profile is required")
	}
	v, err := e.Judge.Evaluate(ctx, c, command)
	if err != nil {
		return Result{Verdict: v}, err
	}
	result := Result{Verdict: v}
	now := time.Now().UTC()
	if e.Now != nil {
		now = e.Now().UTC()
	}
	review := p.Reviews[c.ID]
	isDue := p.Solved[c.ID] && dueReview(review, now)
	if isDue {
		result.Review = true
		review.Attempts++
		if !v.Correct {
			if review.Stage > 0 {
				review.Stage--
			}
			review.DueAt = now.Add(24 * time.Hour).Format(time.RFC3339)
			p.Reviews[c.ID] = review
			return result, nil
		}
	}
	if !v.Correct {
		return result, nil
	}
	if !p.Solved[c.ID] {
		result.XPGained = 100 + c.Difficulty*20 - hintsUsed*10
		if result.XPGained < 20 {
			result.XPGained = 20
		}
		p.XP += result.XPGained
		p.Solved[c.ID] = true
		p.HintsUsed += hintsUsed
		for _, tag := range c.Tags {
			if p.Mastery[tag] < 100 {
				p.Mastery[tag] += 10 + c.Difficulty*2
				if p.Mastery[tag] > 100 {
					p.Mastery[tag] = 100
				}
			}
		}
		old := p.Level
		p.Level = 1 + p.XP/300
		result.LeveledUp = p.Level > old
		item := AwardFor(c.ID, len(p.Solved))
		p.Inventory[item.ID] = true
		p.Loadout[item.Slot] = item.ID
		result.Drop = &item
		p.Reviews[c.ID] = ReviewState{Stage: 0, DueAt: now.Add(24 * time.Hour).Format(time.RFC3339)}
		if e.Content != nil {
			hash, _ := c.Hash()
			generated, generateErr := e.Content.Generate(ctx, ContentRequest{Item: item, ChallengeID: c.ID, ChallengeTitle: c.Title, Skill: item.Skill, ChallengeSHA256: hash})
			if generateErr == nil {
				result.Generated = &generated
			}
		}
	} else if isDue {
		review.Successes++
		review.Stage++
		result.XPGained = 30
		p.XP += result.XPGained
		p.Mastery["recall"] = min(100, p.Mastery["recall"]+10)
		intervals := []time.Duration{24 * time.Hour, 3 * 24 * time.Hour, 7 * 24 * time.Hour, 21 * 24 * time.Hour}
		if review.Stage < len(intervals) {
			review.DueAt = now.Add(intervals[review.Stage]).Format(time.RFC3339)
		} else {
			review.DueAt = ""
		}
		p.Reviews[c.ID] = review
		p.Level = 1 + p.XP/300
	}
	return result, nil
}

func dueReview(review ReviewState, now time.Time) bool {
	if review.DueAt == "" {
		return false
	}
	due, err := time.Parse(time.RFC3339, review.DueAt)
	return err == nil && !now.Before(due)
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
