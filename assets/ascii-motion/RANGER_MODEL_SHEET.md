# Ranger ASCII model sheet

This sheet is the visual template for every GrepQuest player animation.

## Grid and silhouette

- Authored canvas: 60 columns by 60 rows in the Ascii Motion project.
- Runtime source: the exact colored character cells from
  `grepquest_ranger_char_bubble_tee_component.go.txt`.
- Natural terminal size: 60 columns by 30 rows after compensating for terminal
  cell proportions.
- Responsive views fit the complete export without cropping or adding glyphs.

## Motion language

- Idle: a slow signal-glow cycle over approximately seven seconds. The exported
  character cells stay anchored.
- Clear: briefly intensify the signal colors without replacing the source art.
- Loop: every sequence returns to its complete first frame.
- Reduced motion: use the first idle frame and never schedule animation ticks.

The approved visual source is `projects/grepquest_ranger_ascii.asciimtn`.
The reviewed Bubble Tea export is
`exports/grepquest_ranger_char_bubble_tee_component.go.txt`. Runtime rendering
uses only its generated character and color data; there is no separate line-art
overlay. `just ranger-build` regenerates the source cell table.

The visual source is `reference/grepquest-ranger-reference-transparent.png`. It
establishes the winged Q helmet, scarf, segmented shoulders, signal chest plate,
utility belt, split coat, knee plates, and heavy boots. Responsive rendering may
sample cells to fit, but must preserve the complete front-facing silhouette.
