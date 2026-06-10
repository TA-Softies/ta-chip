package ui

import (
	"fmt"
	"strings"
)

type HardwareItem struct {
	Name     string
	Question string
}

type HardwareResult struct {
	Item   HardwareItem
	Status string // "V", "Y", "X", or "" (not yet set)
}

var hardwareItems = []HardwareItem{
	{"Display", "Is the display working correctly?"},
	{"Kensington Lock", "Is the Kensington lock present and secured?"},
	{"Conduiting", "Is cable management/conduiting tidy and secure?"},
	{"Tidiness", "Is the workstation tidy and in good order?"},
}

// renderHardwareScreen renders the current hardware check prompt.
func renderHardwareScreen(item HardwareItem, selected int) string {
	choices := []string{"V  Working/Present", "Y  Partial/Unsecured", "X  Faulty/Missing"}
	var rows []string
	for i, c := range choices {
		if i == selected {
			rows = append(rows, styleSelected.Render("● "+c))
		} else {
			rows = append(rows, styleUnselected.Render("○ "+c))
		}
	}

	content := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
		styleLabel.Render(item.Name),
		styleDim.Render(item.Question),
		strings.Join(rows, "\n"),
		styleDim.Render("↑↓ to select  •  V / Y / X keys  •  Enter to confirm"),
	)

	return styleBox.Render(content)
}
