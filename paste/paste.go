package paste

import (
	"fmt"
	"strings"

	"github.com/startvibecoding/agentui"
)

const (
	DefaultMaxInlineLines = 5
	DefaultMaxInlineChars = 500
)

// Manager stores large paste payloads behind short input markers.
type Manager struct {
	nextID         int
	items          map[int]string
	maxInlineLines int
	maxInlineChars int
}

func New(maxInlineLines, maxInlineChars int) *Manager {
	if maxInlineLines <= 0 {
		maxInlineLines = DefaultMaxInlineLines
	}
	if maxInlineChars <= 0 {
		maxInlineChars = DefaultMaxInlineChars
	}
	return &Manager{
		items:          make(map[int]string),
		maxInlineLines: maxInlineLines,
		maxInlineChars: maxInlineChars,
	}
}

func Default() *Manager {
	return New(DefaultMaxInlineLines, DefaultMaxInlineChars)
}

// Insert returns the string that should be inserted into the editor. Large
// pastes are replaced by markers and stored for later expansion.
func (m *Manager) Insert(text string) string {
	if m == nil {
		return Normalize(text)
	}
	text = Normalize(text)
	lines := strings.Split(text, "\n")
	if len(lines) <= m.maxInlineLines && len(text) <= m.maxInlineChars {
		return text
	}
	m.nextID++
	id := m.nextID
	m.items[id] = text
	if len(lines) > m.maxInlineLines {
		return fmt.Sprintf("[paste #%d +%d lines]", id, len(lines))
	}
	return fmt.Sprintf("[paste #%d %d chars]", id, len(text))
}

// Expand replaces known markers in text with original paste payloads. Expanded
// paste payloads are removed from the manager.
func (m *Manager) Expand(text string) string {
	if m == nil || len(m.items) == 0 {
		return text
	}
	result := text
	used := make(map[int]bool)
	for id, content := range m.items {
		lineMarker := fmt.Sprintf("[paste #%d +%d lines]", id, strings.Count(content, "\n")+1)
		charMarker := fmt.Sprintf("[paste #%d %d chars]", id, len(content))
		if strings.Contains(result, lineMarker) {
			result = strings.ReplaceAll(result, lineMarker, content)
			used[id] = true
			continue
		}
		if strings.Contains(result, charMarker) {
			result = strings.ReplaceAll(result, charMarker, content)
			used[id] = true
		}
	}
	for id := range used {
		delete(m.items, id)
	}
	return result
}

func (m *Manager) Count() int {
	if m == nil {
		return 0
	}
	return len(m.items)
}

func Normalize(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n")
}

// HasLineBreak reports whether key messages contain a logical newline.
func HasLineBreak(keys []agentui.KeyMsg) bool {
	for _, key := range keys {
		switch key.Type {
		case agentui.KeyEnter, agentui.KeyCtrlJ:
			if key.Type == agentui.KeyEnter && key.Alt {
				continue
			}
			return true
		case agentui.KeyRunes:
			if strings.ContainsAny(string(key.Runes), "\r\n") {
				return true
			}
		}
	}
	return false
}

// CoalesceSplit turns terminals' split paste key events back into one string.
// It returns false for ordinary typing and for events that include navigation
// or editing keys.
func CoalesceSplit(keys []agentui.KeyMsg) (string, bool) {
	var b strings.Builder
	enterCount := 0
	sawEnter := false
	textAfterEnter := false

	for _, key := range keys {
		switch key.Type {
		case agentui.KeyRunes:
			text := Normalize(string(key.Runes))
			if text == "" {
				continue
			}
			if sawEnter {
				textAfterEnter = true
			}
			enterCount += strings.Count(text, "\n")
			if strings.Contains(text, "\n") {
				sawEnter = true
				textAfterEnter = true
			}
			b.WriteString(text)
		case agentui.KeySpace:
			if sawEnter {
				textAfterEnter = true
			}
			b.WriteRune(' ')
		case agentui.KeyEnter, agentui.KeyCtrlJ:
			if key.Type == agentui.KeyEnter && key.Alt {
				return "", false
			}
			enterCount++
			sawEnter = true
			b.WriteRune('\n')
		default:
			return "", false
		}
	}
	if enterCount == 0 || !textAfterEnter {
		return "", false
	}
	return b.String(), true
}
