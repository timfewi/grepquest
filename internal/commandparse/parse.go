package commandparse

import (
	"errors"
	"fmt"
	"strings"
)

// Split accepts shell-like quoting but intentionally implements no expansion,
// substitution, redirection, pipelines, or command separators.
func Split(input string) ([]string, error) {
	var out []string
	var word strings.Builder
	quote := rune(0)
	escaped, active := false, false
	flush := func() {
		if active {
			out = append(out, word.String())
			word.Reset()
			active = false
		}
	}
	for _, r := range input {
		if escaped {
			// Within double quotes, POSIX shells preserve a backslash unless it
			// precedes one of the handful of special characters. There is no
			// expansion here, but retaining this rule makes regexes such as
			// "\\.conf$" behave as players expect.
			if quote == '"' && r != '"' && r != '\\' && r != '$' && r != '`' && r != '\n' {
				word.WriteRune('\\')
			}
			if r == '\n' {
				escaped = false
				continue
			}
			word.WriteRune(r)
			active = true
			escaped = false
			continue
		}
		if quote == '\'' {
			if r == '\'' {
				quote = 0
			} else {
				word.WriteRune(r)
			}
			active = true
			continue
		}
		if quote == '"' {
			switch r {
			case '"':
				quote = 0
			case '\\':
				escaped = true
			default:
				word.WriteRune(r)
			}
			active = true
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
			active = true
		case '\\':
			escaped = true
			active = true
		case ' ', '\t', '\n':
			flush()
		case ';', '|', '&', '>', '<', '`', '$', '(', ')':
			return nil, fmt.Errorf("shell syntax %q is not allowed", r)
		default:
			word.WriteRune(r)
			active = true
		}
	}
	if escaped || quote != 0 {
		return nil, errors.New("unfinished escape or quote")
	}
	flush()
	return out, nil
}
