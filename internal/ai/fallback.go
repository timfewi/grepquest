package ai

import (
	"context"
	"github.com/timfewi/grepquest/internal/game"
)

// Fallback is deterministic and never needs network access or credentials.
type Fallback struct{}

func (Fallback) Generate(_ context.Context, r game.ContentRequest) (game.GeneratedContent, error) {
	return game.GeneratedContent{Name: r.Item.Name, Flavor: r.Item.Flavor, Sprite: r.Item.Sprite, Explanation: "Mastery grows through exact, reproducible grep practice.", Source: "offline"}, nil
}

type WithFallback struct {
	Primary  game.ContentGenerator
	Fallback game.ContentGenerator
}

func (g WithFallback) Generate(ctx context.Context, r game.ContentRequest) (game.GeneratedContent, error) {
	if g.Primary != nil {
		if content, err := g.Primary.Generate(ctx, r); err == nil {
			return content, nil
		}
	}
	if g.Fallback == nil {
		g.Fallback = Fallback{}
	}
	return g.Fallback.Generate(ctx, r)
}
