package sui

import (
	"errors"
	"sort"
)

type Metric string

const (
	SeasonMastery Metric = "score"
	Accuracy      Metric = "accuracy"
	Recall        Metric = "recall"
	NoHintClears  Metric = "no-hint"
	Speed         Metric = "speed"
)

type RankEntry struct {
	Rank  int
	Value uint64
	Event ScoreEvent
}

func ParseMetric(value string) (Metric, error) {
	metric := Metric(value)
	switch metric {
	case SeasonMastery, Accuracy, Recall, NoHintClears, Speed:
		return metric, nil
	default:
		return "", errors.New("metric must be score, accuracy, recall, no-hint, or speed")
	}
}

// Rank filters one exact proof tier, retains each player's best event, then
// ranks locally. Speed is deliberately separate because it is hardware-bound.
func Rank(events []ScoreEvent, season string, tier ProofTier, metric Metric) []RankEntry {
	best := make(map[string]ScoreEvent)
	for _, event := range events {
		if event.SeasonID != season || event.ProofTier != tier || (metric == Speed && event.ElapsedMS == 0) {
			continue
		}
		previous, ok := best[event.Player]
		if !ok || better(event, previous, metric) {
			best[event.Player] = event
		}
	}
	entries := make([]RankEntry, 0, len(best))
	for _, event := range best {
		entries = append(entries, RankEntry{Value: metricValue(event, metric), Event: event})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Value != entries[j].Value {
			if metric == Speed {
				return entries[i].Value < entries[j].Value
			}
			return entries[i].Value > entries[j].Value
		}
		if entries[i].Event.Score != entries[j].Event.Score {
			return entries[i].Event.Score > entries[j].Event.Score
		}
		return entries[i].Event.Player < entries[j].Event.Player
	})
	for i := range entries {
		if i == 0 || entries[i].Value != entries[i-1].Value {
			entries[i].Rank = i + 1
		} else {
			entries[i].Rank = entries[i-1].Rank
		}
	}
	return entries
}

func better(candidate, current ScoreEvent, metric Metric) bool {
	candidateValue, currentValue := metricValue(candidate, metric), metricValue(current, metric)
	if candidateValue != currentValue {
		if metric == Speed {
			return candidateValue < currentValue
		}
		return candidateValue > currentValue
	}
	return candidate.Score > current.Score
}

func metricValue(event ScoreEvent, metric Metric) uint64 {
	switch metric {
	case Accuracy:
		return event.Accuracy
	case Recall:
		return event.Recall
	case NoHintClears:
		return event.NoHintClears
	case Speed:
		return event.ElapsedMS
	default:
		return event.Score
	}
}
