package prompt

import (
	"fmt"
	"sort"

	"github.com/fatih/color"
	"github.com/isacikgoz/gitin/term"
)

func itemText(item interface{}, matches []int, selected bool) []term.Cell {
	var line []term.Cell
	text := fmt.Sprint(item)
	if selected {
		line = append(line, term.Cprint("> ", color.FgCyan)...)
	} else {
		line = append(line, term.Cprint("  ", color.FgWhite)...)
	}
	if len(matches) == 0 {
		return append(line, term.Cprint(text)...)
	}
	highlighted := make([]term.Cell, 0)
	for _, r := range text {
		highlighted = append(highlighted, term.Cell{
			Ch: r,
		})
	}
	for _, m := range matches {
		if m > len(highlighted)-1 {
			continue
		}
		highlighted[m] = term.Cell{
			Ch:   highlighted[m].Ch,
			Attr: append(highlighted[m].Attr, color.Underline),
		}
	}
	line = append(line, highlighted...)
	return line
}

// returns multiline so the return value will be a 2-d slice
func genHelp(pairs map[string]string) [][]term.Cell {
	var grid [][]term.Cell
	// sort keys alphabetically
	keys := make([]string, 0, len(pairs))
	for key := range pairs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		grid = append(grid, append(term.Cprint(fmt.Sprintf("%s: ", key), color.Faint),
			term.Cprint(fmt.Sprintf("%s", pairs[key]), color.FgYellow)...))
	}
	grid = append(grid, term.Cprint("", 0))
	grid = append(grid, term.Cprint("press any key to return.", color.Faint))
	return grid
}

func renderSearch(placeholder string, inputMode bool, input string) []term.Cell {
	var cells []term.Cell
	if inputMode {
		cells = term.Cprint("Search ", color.Faint)
		cells = append(cells, term.Cprint(placeholder+" ", color.Faint)...)
		cells = append(cells, term.Cprint(input, color.FgWhite)...)
		cells = append(cells, term.Cprint("█", color.Faint, color.BlinkRapid)...)
		return cells
	}
	cells = term.Cprint(placeholder, color.Faint)
	if len(input) > 0 {
		cells = append(cells, term.Cprint(" /"+input, color.FgWhite)...)
	}

	return cells
}
