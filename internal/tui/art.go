package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/timfewi/grepquest/internal/challenge"
)

func renderRoomScene(c challenge.Challenge, index, total int) string {
	actName := "SIGNAL CELLAR"
	route := "@---+---+---[Q]"
	mechanism := "literal signal lock"
	if c.Act == 2 {
		actName = "REGEX FOUNDRY"
		route = "@=={ }==[ ]==[Q]"
		mechanism = "pattern forge online"
	}

	files := make([]string, 0, len(c.Fixtures))
	for name := range c.Fixtures {
		files = append(files, name)
	}
	sort.Strings(files)
	target := strings.Join(files, ", ")
	if target == "" {
		target = "no fixture"
	}

	progressWidth := 12
	cleared := index * progressWidth / total
	if cleared > progressWidth {
		cleared = progressWidth
	}
	progress := strings.Repeat("#", cleared) + strings.Repeat(".", progressWidth-cleared)
	progressLine := fmt.Sprintf("run [%s] %2d/%-2d", progress, index, total)

	return fmt.Sprintf(
		"+-- %s / ROOM %02d --+\n| %-27s |\n| target: %-19.19s |\n| %-27s |\n| %-27s |\n+-----------------------------+",
		actName,
		index+1,
		route,
		target,
		mechanism,
		progressLine,
	)
}
