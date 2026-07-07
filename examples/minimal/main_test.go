package main

import (
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/startvibecoding/agentui"
	"github.com/startvibecoding/agentui/ansi"
)

func TestAssistantMarkdownEmphasisPreservesTextOrder(t *testing.T) {
	a := newApp()
	a.width = 24
	a.configureMarkdownRenderer()
	a.addAssistantMarkdown("用户查看 *AGENTS* 文件内容")

	got := ansi.Strip(a.renderAssistantMessage(len(a.messages) - 1))
	got = strings.Join(strings.Fields(got), "")
	want := "Assistant:用户查看AGENTS文件内容"
	if got != want {
		t.Fatalf("rendered text order = %q, want %q", got, want)
	}
}

func TestAssistantMarkdownInlineEmphasisStyleScope(t *testing.T) {
	a := newApp()
	a.width = 80
	a.configureMarkdownRenderer()
	a.addAssistantMarkdown("1111*22222*3333")

	cells := ansiStyleCells(a.renderAssistantMessage(len(a.messages) - 1))
	assertItalicSpan(t, cells, "1111", false)
	assertItalicSpan(t, cells, "22222", true)
	assertItalicSpan(t, cells, "3333", false)
}

func TestEchoMarkdownInlineEmphasisStyleScope(t *testing.T) {
	a := newApp()
	a.width = 80
	a.configureMarkdownRenderer()
	a.addAssistantMarkdown(echoMarkdown("1111*22222*3333"))

	cells := ansiStyleCells(a.renderAssistantMessage(len(a.messages) - 1))
	assertItalicSpan(t, cells, "1111", false)
	assertItalicSpan(t, cells, "22222", true)
	assertItalicSpan(t, cells, "3333", false)
}

func TestAssistantMarkdownWrapsWithinAppWidth(t *testing.T) {
	a := newApp()
	a.width = 28
	a.configureMarkdownRenderer()
	raw := "用户查看 *AGENTS* 文件内容，并打开 `internal/tui/renderutil/ansi_wrap.go` 继续检查折行。"
	a.addAssistantMarkdown(raw)

	rendered := a.renderAssistantMessage(len(a.messages) - 1)
	for _, line := range strings.Split(rendered, "\n") {
		if width := ansi.StringWidth(line); width > a.width {
			t.Fatalf("line width = %d, want <= %d: %q\nraw:\n%s", width, a.width, line, rendered)
		}
	}

	plain := strings.Join(strings.Fields(ansi.Strip(rendered)), "")
	for _, want := range []string{"用户查看AGENTS文件内容", "internal/tui/renderutil/ansi_wrap.go"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("rendered text missing %q in %q\nraw:\n%s", want, plain, rendered)
		}
	}
}

type styledCell struct {
	r      rune
	italic bool
}

func ansiStyleCells(s string) []styledCell {
	var cells []styledCell
	italic := false
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			if n, ok := parseSGR(s[i:], &italic); ok {
				i += n
				continue
			}
		}
		r, n := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && n == 1 {
			i++
			continue
		}
		cells = append(cells, styledCell{r: r, italic: italic})
		i += n
	}
	return cells
}

func parseSGR(s string, italic *bool) (int, bool) {
	if len(s) < 3 || s[0] != 0x1b || s[1] != '[' {
		return 0, false
	}
	end := strings.IndexByte(s, 'm')
	if end < 0 {
		return 0, false
	}
	body := s[2:end]
	if body == "" {
		*italic = false
		return end + 1, true
	}
	for _, part := range strings.Split(body, ";") {
		code, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		switch code {
		case 0:
			*italic = false
		case 3:
			*italic = true
		case 23:
			*italic = false
		}
	}
	return end + 1, true
}

func assertItalicSpan(t *testing.T, cells []styledCell, text string, want bool) {
	t.Helper()
	runes := []rune(text)
	start := indexStyledRunes(cells, runes)
	if start < 0 {
		t.Fatalf("visible text %q not found in %q", text, styledCellsString(cells))
	}
	for i := range runes {
		if got := cells[start+i].italic; got != want {
			t.Fatalf("italic(%q char %d) = %v, want %v in visible %q", text, i, got, want, styledCellsString(cells))
		}
	}
}

func indexStyledRunes(cells []styledCell, runes []rune) int {
	if len(runes) == 0 || len(runes) > len(cells) {
		return -1
	}
	for i := 0; i+len(runes) <= len(cells); i++ {
		ok := true
		for j := range runes {
			if cells[i+j].r != runes[j] {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

func styledCellsString(cells []styledCell) string {
	var b strings.Builder
	for _, cell := range cells {
		b.WriteRune(cell.r)
	}
	return b.String()
}

func TestMinimalViewportKeepsWrappedMarkdownText(t *testing.T) {
	a := newApp()
	a.width = 28
	a.height = 20
	a.view = a.view.SetSize(a.width, a.height)
	a.configureMarkdownRenderer()
	a.messages = nil
	a.assistantRaw = make(map[int]string)
	a.assistantRendered = make(map[int]string)
	a.assistantDirty = make(map[int]bool)
	a.addAssistantMarkdown("用户查看 *AGENTS* 文件内容，并打开 `internal/tui/renderutil/ansi_wrap.go` 继续检查折行。")
	a.rebuild()

	view := a.view.View()
	plain := strings.Join(strings.Fields(ansi.Strip(view)), "")
	for _, want := range []string{"用户查看AGENTS文件内容", "internal/tui/renderutil/ansi_wrap.go"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("viewport text missing %q in %q\nraw:\n%s", want, plain, view)
		}
	}
	for _, line := range strings.Split(view, "\n") {
		if width := ansi.StringWidth(line); width > a.width {
			t.Fatalf("viewport line width = %d, want <= %d: %q\nraw:\n%s", width, a.width, line, view)
		}
	}
}

func TestMinimalInputBoxIsVisible(t *testing.T) {
	a := newApp()
	a.width = 48
	a.height = 16
	a.rebuild()

	view := ansi.Strip(a.View())
	for _, want := range []string{"╭", "╰", "> Type here"} {
		if !strings.Contains(view, want) {
			t.Fatalf("minimal view missing input box marker %q:\n%s", want, view)
		}
	}
}

func TestMinimalPopupOpensAndFitsWidth(t *testing.T) {
	a := newApp()
	a.width = 42
	a.height = 18
	a.rebuild()

	model, _ := a.Update(agentui.KeyMsg{Type: agentui.KeyCtrlO})
	a = model.(*app)
	if !a.popup.Open {
		t.Fatal("popup should be open after Ctrl+O")
	}
	view := a.View()
	plain := ansi.Strip(view)
	for _, want := range []string{"Regression popup", "弹框回归内容", "Esc/q/Ctrl+O close"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("popup view missing %q:\n%s", want, plain)
		}
	}
	for _, line := range strings.Split(view, "\n") {
		if width := ansi.StringWidth(line); width > a.width {
			t.Fatalf("line width = %d, want <= %d: %q\nview:\n%s", width, a.width, line, view)
		}
	}
}

func TestMinimalPopupConsumesInputUntilClosed(t *testing.T) {
	a := newApp()
	a.rebuild()

	model, _ := a.Update(agentui.KeyMsg{Type: agentui.KeyCtrlO})
	a = model.(*app)
	model, _ = a.Update(agentui.KeyMsg{Type: agentui.KeyRunes, Runes: []rune("hidden")})
	a = model.(*app)
	if got := a.input.Value(); got != "" {
		t.Fatalf("input while popup open = %q, want empty", got)
	}

	model, _ = a.Update(agentui.KeyMsg{Type: agentui.KeyEsc})
	a = model.(*app)
	if a.popup.Open {
		t.Fatal("popup should close after Esc")
	}
	model, _ = a.Update(agentui.KeyMsg{Type: agentui.KeyRunes, Runes: []rune("visible")})
	a = model.(*app)
	if got := a.input.Value(); got != "visible" {
		t.Fatalf("input after popup close = %q, want visible", got)
	}
}

func TestMinimalSubmitShowsActiveTurnThenCommitPrintsAndClearsLiveView(t *testing.T) {
	a := newApp()
	a.width = 64
	a.height = 20
	a.rebuild()

	model, _ := a.Update(agentui.KeyMsg{Type: agentui.KeyRunes, Runes: []rune("scrollback check")})
	a = model.(*app)
	model, submitCmd := a.Update(agentui.KeyMsg{Type: agentui.KeyEnter})
	a = model.(*app)
	if submitCmd == nil {
		t.Fatal("enter should return submit command")
	}
	model, cmd := a.Update(submitCmd())
	a = model.(*app)
	if cmd != nil {
		t.Fatal("submit should keep the turn live instead of printing it immediately")
	}
	view := ansi.Strip(a.View())
	if !strings.Contains(view, "scrollback check") {
		t.Fatalf("submitted text should be visible in active live view:\n%s", view)
	}
	if strings.Contains(view, "Printed ") {
		t.Fatalf("managed live view should not contain a fake printed-status block:\n%s", view)
	}

	model, printCmd := a.Update(agentui.KeyMsg{Type: agentui.KeyCtrlP})
	a = model.(*app)
	if printCmd == nil {
		t.Fatal("Ctrl+P should return print command for active turn")
	}
	view = ansi.Strip(a.View())
	if strings.Contains(view, "scrollback check") {
		t.Fatalf("committed turn stayed in managed live view:\n%s", view)
	}
	if strings.Contains(view, "Printed ") {
		t.Fatalf("managed live view should not contain a fake printed-status block:\n%s", view)
	}
	if !strings.Contains(view, "terminal scrollback") {
		t.Fatalf("live view should explain scrollback printing:\n%s", view)
	}
}

func TestMinimalCommitCommandPrintsActiveTurn(t *testing.T) {
	a := newApp()
	a.width = 64
	a.height = 20
	a.rebuild()

	model, _ := a.Update(agentui.KeyMsg{Type: agentui.KeyRunes, Runes: []rune("command commit")})
	a = model.(*app)
	model, submitCmd := a.Update(agentui.KeyMsg{Type: agentui.KeyEnter})
	a = model.(*app)
	model, _ = a.Update(submitCmd())
	a = model.(*app)

	model, _ = a.Update(agentui.KeyMsg{Type: agentui.KeyRunes, Runes: []rune("/commit")})
	a = model.(*app)
	model, commitCmd := a.Update(agentui.KeyMsg{Type: agentui.KeyEnter})
	a = model.(*app)
	if commitCmd == nil {
		t.Fatal("enter should return submit command for /commit")
	}
	model, printCmd := a.Update(commitCmd())
	a = model.(*app)
	if printCmd == nil {
		t.Fatal("/commit should return print command for active turn")
	}
	if view := ansi.Strip(a.View()); strings.Contains(view, "command commit") {
		t.Fatalf("/commit should remove active turn from live view:\n%s", view)
	}
}

func TestMinimalSuggestionsShowForSlash(t *testing.T) {
	a := newApp()
	a.width = 48
	a.height = 18
	a.rebuild()

	model, _ := a.Update(agentui.KeyMsg{Type: agentui.KeyRunes, Runes: []rune("/")})
	a = model.(*app)
	if !a.suggest.Visible() {
		t.Fatal("suggestions should be visible after slash")
	}
	view := ansi.Strip(a.View())
	for _, want := range []string{"/popup", "/markdown", "Tab apply"} {
		if !strings.Contains(view, want) {
			t.Fatalf("suggestion view missing %q:\n%s", want, view)
		}
	}
	for _, line := range strings.Split(a.View(), "\n") {
		if width := ansi.StringWidth(line); width > a.width {
			t.Fatalf("line width = %d, want <= %d: %q\nview:\n%s", width, a.width, line, a.View())
		}
	}
}

func TestMinimalSuggestionTabAppliesCommand(t *testing.T) {
	a := newApp()
	a.rebuild()

	model, _ := a.Update(agentui.KeyMsg{Type: agentui.KeyRunes, Runes: []rune("/ma")})
	a = model.(*app)
	if !a.suggest.Visible() {
		t.Fatal("suggestions should be visible for /ma")
	}
	model, _ = a.Update(agentui.KeyMsg{Type: agentui.KeyTab})
	a = model.(*app)
	if got := a.input.Value(); got != "/markdown" {
		t.Fatalf("input after Tab = %q, want /markdown", got)
	}
}

func TestMinimalModeArgumentSuggestions(t *testing.T) {
	a := newApp()
	a.rebuild()

	model, _ := a.Update(agentui.KeyMsg{Type: agentui.KeyRunes, Runes: []rune("/mo")})
	a = model.(*app)
	model, _ = a.Update(agentui.KeyMsg{Type: agentui.KeyTab})
	a = model.(*app)
	if got := a.input.Value(); got != "/mode " {
		t.Fatalf("input after command Tab = %q, want /mode ", got)
	}
	if !a.suggest.Visible() {
		t.Fatal("mode argument suggestions should remain visible")
	}
	model, _ = a.Update(agentui.KeyMsg{Type: agentui.KeyDown})
	a = model.(*app)
	model, _ = a.Update(agentui.KeyMsg{Type: agentui.KeyTab})
	a = model.(*app)
	if got := a.input.Value(); got != "/mode agent" {
		t.Fatalf("input after argument Tab = %q, want /mode agent", got)
	}
}

func TestMinimalPopupHidesSuggestionsAndConsumesSuggestionKeys(t *testing.T) {
	a := newApp()
	a.rebuild()

	model, _ := a.Update(agentui.KeyMsg{Type: agentui.KeyRunes, Runes: []rune("/")})
	a = model.(*app)
	if !a.suggest.Visible() {
		t.Fatal("setup expected suggestions")
	}
	model, _ = a.Update(agentui.KeyMsg{Type: agentui.KeyCtrlO})
	a = model.(*app)
	if !a.popup.Open {
		t.Fatal("popup should be open")
	}
	if a.suggest.Visible() {
		t.Fatal("suggestions should hide while popup is open")
	}
	model, _ = a.Update(agentui.KeyMsg{Type: agentui.KeyTab})
	a = model.(*app)
	if got := a.input.Value(); got != "/" {
		t.Fatalf("popup should consume Tab without changing input, got %q", got)
	}
}

func TestMinimalPopupCommandOpensPopup(t *testing.T) {
	a := newApp()
	a.rebuild()

	model, _ := a.Update(agentui.KeyMsg{Type: agentui.KeyRunes, Runes: []rune("/popup")})
	a = model.(*app)
	model, cmd := a.Update(agentui.KeyMsg{Type: agentui.KeyEnter})
	a = model.(*app)
	if cmd == nil {
		t.Fatal("enter should produce submit command")
	}
	model, _ = a.Update(cmd())
	a = model.(*app)
	if !a.popup.Open {
		t.Fatal("/popup command should open popup")
	}
}
