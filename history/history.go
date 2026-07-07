package history

import "strings"

// Model stores prompt history and supports shell-like up/down navigation.
type Model struct {
	items    []string
	limit    int
	browsing bool
	index    int
	draft    string
}

func New(limit int) Model {
	if limit <= 0 {
		limit = 200
	}
	return Model{limit: limit}
}

func (m Model) Items() []string { return append([]string(nil), m.items...) }
func (m Model) Browsing() bool  { return m.browsing }

func (m Model) Record(input string) Model {
	input = strings.TrimSpace(input)
	if input == "" {
		return m.ResetNavigation()
	}
	if len(m.items) > 0 && m.items[len(m.items)-1] == input {
		return m.ResetNavigation()
	}
	m.items = append(m.items, input)
	if len(m.items) > m.limit {
		m.items = m.items[len(m.items)-m.limit:]
	}
	return m.ResetNavigation()
}

func (m Model) Prev(current string) (Model, string, bool) {
	if len(m.items) == 0 {
		return m, current, false
	}
	if !m.browsing {
		m.draft = current
		m.index = len(m.items) - 1
		m.browsing = true
	} else if m.index > 0 {
		m.index--
	}
	return m, m.items[m.index], true
}

func (m Model) Next(current string) (Model, string, bool) {
	if !m.browsing {
		return m, current, false
	}
	if m.index < len(m.items)-1 {
		m.index++
		return m, m.items[m.index], true
	}
	draft := m.draft
	m = m.ResetNavigation()
	return m, draft, true
}

func (m Model) ResetNavigation() Model {
	m.browsing = false
	m.index = 0
	m.draft = ""
	return m
}
