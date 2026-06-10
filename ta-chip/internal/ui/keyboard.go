package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// keyboard layout rows — key display name → bubbletea key string
// We track by display label; bubbletea key names are mapped where they differ.
var keyRows = [][]string{
	{"esc", "f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12"},
	{"`", "1", "2", "3", "4", "5", "6", "7", "8", "9", "0", "-", "=", "bksp"},
	{"tab", "q", "w", "e", "r", "t", "y", "u", "i", "o", "p", "[", "]", "\\"},
	{"caps", "a", "s", "d", "f", "g", "h", "j", "k", "l", ";", "'", "enter"},
	{"shift", "z", "x", "c", "v", "b", "n", "m", ",", ".", "/", "rshift"},
	{"ctrl", "win", "alt", "space", "ralt", "fn", "rctrl"},
}

var mouseButtons = []string{"LMB", "MMB", "RMB"}

type KeyTestModel struct {
	pressed      map[string]bool
	mousePresses map[string]bool
}

func newKeyTestModel() KeyTestModel {
	return KeyTestModel{
		pressed:      make(map[string]bool),
		mousePresses: make(map[string]bool),
	}
}

// handleKeyPress maps tea.KeyMsg to our keyboard layout keys.
func (m *KeyTestModel) handleKeyPress(msg tea.KeyMsg) {
	k := strings.ToLower(string(msg.Runes))
	if k == "" {
		k = strings.ToLower(msg.Type.String())
	}
	// Normalise common aliases
	aliases := map[string]string{
		"backspace":  "bksp",
		"delete":     "del",
		"escape":     "esc",
		"return":     "enter",
		"left":       "←",
		"right":      "→",
		"up":         "↑",
		"down":       "↓",
		"pgup":       "pgup",
		"pgdown":     "pgdn",
		"home":       "home",
		"end":        "end",
		"insert":     "ins",
		"capslock":   "caps",
		"tab":        "tab",
		"space":      "space",
		" ":          "space",
		"shift":      "shift",
		"ctrl":       "ctrl",
		"alt":        "alt",
		"f1":  "f1", "f2": "f2", "f3": "f3", "f4": "f4",
		"f5":  "f5", "f6": "f6", "f7": "f7", "f8": "f8",
		"f9":  "f9", "f10": "f10", "f11": "f11", "f12": "f12",
	}
	if mapped, ok := aliases[k]; ok {
		k = mapped
	}
	m.pressed[k] = true
}

func (m *KeyTestModel) handleMousePress(btn string) {
	m.mousePresses[btn] = true
}

func (m *KeyTestModel) pressedCount() int {
	return len(m.pressed)
}

func (m *KeyTestModel) renderKeyboard() string {
	var sb strings.Builder
	for _, row := range keyRows {
		var cells []string
		for _, key := range row {
			label := strings.ToUpper(key)
			if len(label) > 4 {
				label = label[:4]
			}
			padding := 4 - len(label)
			cell := fmt.Sprintf(" %s%s ", label, strings.Repeat(" ", padding))
			if m.pressed[key] {
				cells = append(cells, styleKeyPressed.Render(cell))
			} else {
				cells = append(cells, styleKeyUnpressed.Render(cell))
			}
		}
		sb.WriteString(strings.Join(cells, " "))
		sb.WriteString("\n")
	}

	// Mouse row
	sb.WriteString("\n  Mouse:  ")
	for _, btn := range mouseButtons {
		if m.mousePresses[btn] {
			sb.WriteString(styleKeyPressed.Render(" "+btn+" ") + "  ")
		} else {
			sb.WriteString(styleKeyUnpressed.Render(" "+btn+" ") + "  ")
		}
	}
	sb.WriteString("\n")

	return sb.String()
}

func renderKeyboardTestScreen(kt KeyTestModel) string {
	header := styleHeader.Render("Keyboard & Mouse Test")
	hint := styleDim.Render("Press every key you want to test. Click each mouse button. Press Enter when done.")
	keyboard := kt.renderKeyboard()
	footer := fmt.Sprintf("\n  %s  %s",
		styleDim.Render(fmt.Sprintf("Keys registered: %d", kt.pressedCount())),
		styleDim.Render("Enter → Done"),
	)
	return fmt.Sprintf("%s\n%s\n\n%s%s", header, hint, keyboard, footer)
}
