package learning

import (
	"github.com/timfewi/grepquest/internal/challenge"
	"github.com/timfewi/grepquest/internal/game"
	"sort"
	"time"
)

// Queue schedules due retrieval practice before unseen rooms. Completed rooms
// remain available as free play once no due or unseen work remains.
func Queue(catalog []challenge.Challenge, profile game.Profile, now time.Time) []challenge.Challenge {
	byID := map[string]challenge.Challenge{}
	for _, c := range catalog {
		byID[c.ID] = c
	}
	type dueItem struct {
		due time.Time
		c   challenge.Challenge
	}
	due := []dueItem{}
	for id, state := range profile.Reviews {
		if state.DueAt == "" {
			continue
		}
		when, err := time.Parse(time.RFC3339, state.DueAt)
		if err == nil && !now.UTC().Before(when) {
			if c, ok := byID[id]; ok {
				due = append(due, dueItem{when, c})
			}
		}
	}
	sort.Slice(due, func(i, j int) bool { return due[i].due.Before(due[j].due) })
	result := make([]challenge.Challenge, 0, len(catalog))
	for _, item := range due {
		result = append(result, item.c)
	}
	for _, c := range catalog {
		if !profile.Solved[c.ID] {
			result = append(result, c)
		}
	}
	if len(result) == 0 {
		return append([]challenge.Challenge(nil), catalog...)
	}
	return result
}
