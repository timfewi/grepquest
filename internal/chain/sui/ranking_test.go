package sui

import "testing"

func TestRankSeparatesProofTiersAndKeepsPlayerBest(t *testing.T) {
	t.Parallel()
	events := []ScoreEvent{
		{Player: "0xa", SeasonID: "s1", Score: 100, ProofTier: SelfAttested},
		{Player: "0xa", SeasonID: "s1", Score: 300, ProofTier: SelfAttested},
		{Player: "0xb", SeasonID: "s1", Score: 200, ProofTier: SelfAttested},
		{Player: "0xc", SeasonID: "s1", Score: 999, ProofTier: Verified},
		{Player: "0xd", SeasonID: "other", Score: 999, ProofTier: SelfAttested},
	}
	self := Rank(events, "s1", SelfAttested, SeasonMastery)
	if len(self) != 2 || self[0].Event.Player != "0xa" || self[0].Value != 300 || self[1].Event.Player != "0xb" {
		t.Fatalf("unexpected self ranking: %#v", self)
	}
	verified := Rank(events, "s1", Verified, SeasonMastery)
	if len(verified) != 1 || verified[0].Event.Player != "0xc" {
		t.Fatalf("unexpected verified ranking: %#v", verified)
	}
}

func TestRankKeepsSpeedSeparateAndAscending(t *testing.T) {
	t.Parallel()
	events := []ScoreEvent{
		{Player: "0xa", SeasonID: "s1", Score: 100, ProofTier: Verified, ElapsedMS: 900},
		{Player: "0xb", SeasonID: "s1", Score: 200, ProofTier: Verified, ElapsedMS: 500},
		{Player: "0xc", SeasonID: "s1", Score: 300, ProofTier: Verified},
	}
	ranking := Rank(events, "s1", Verified, Speed)
	if len(ranking) != 2 || ranking[0].Event.Player != "0xb" || ranking[0].Value != 500 {
		t.Fatalf("unexpected speed ranking: %#v", ranking)
	}
}
