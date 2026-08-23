package game

import "context"

type ContentRequest struct {
	Item                                                Item
	ChallengeID, ChallengeTitle, Skill, ChallengeSHA256 string
}
type GeneratedContent struct{ Name, Flavor, Sprite, Explanation, Source string }
type ContentGenerator interface {
	Generate(context.Context, ContentRequest) (GeneratedContent, error)
}
