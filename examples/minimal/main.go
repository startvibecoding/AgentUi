package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/startvibecoding/agentui"
	"github.com/startvibecoding/agentui/ansi"
	"github.com/startvibecoding/agentui/editor"
	"github.com/startvibecoding/agentui/history"
	"github.com/startvibecoding/agentui/overlay"
	"github.com/startvibecoding/agentui/paste"
	"github.com/startvibecoding/agentui/renderutil"
	"github.com/startvibecoding/agentui/style"
	"github.com/startvibecoding/agentui/suggest"
	"github.com/startvibecoding/agentui/viewport"
)

type app struct {
	width    int
	height   int
	input    editor.Model
	view     viewport.Model
	popup    overlay.Model
	suggest  suggest.Model
	history  history.Model
	pastes   *paste.Manager
	messages []string
	turns    []turn

	mdRenderer        *renderutil.MarkdownRenderer
	assistantRaw      map[int]string
	assistantRendered map[int]string
	assistantDirty    map[int]bool
}

type turn struct {
	user         string
	assistantRaw string
}

func newApp() *app {
	a := &app{
		width:   80,
		height:  24,
		input:   newInput(80),
		view:    viewport.New(80, 20),
		suggest: suggest.New(80).SetMaxVisible(5).SetItems(commandSuggestionItems()),
		history: history.New(200),
		pastes:  paste.Default(),
		messages: []string{
			style.New().Foreground(style.Color("86")).Bold(true).Render("agentui minimal demo"),
			style.New().Foreground(style.Color("240")).Render("Completed turns print to terminal scrollback. The managed view keeps only active content."),
			style.New().Foreground(style.Color("240")).Render("Enter starts a live turn. Ctrl+P or /commit prints it to scrollback. Ctrl+C quits."),
		},
		assistantRaw:      make(map[int]string),
		assistantRendered: make(map[int]string),
		assistantDirty:    make(map[int]bool),
	}
	a.configureMarkdownRenderer()
	a.addAssistantMarkdown("### Markdown rendering\n\nAssistant messages in this demo are rendered with `github.com/startvibecoding/GoStreamingMarkdown`.\n\n- Lists\n- **Bold text**\n- `inline code`\n")
	return a
}

func (a *app) Init() agentui.Cmd {
	a.rebuild()
	return nil
}

func (a *app) Update(msg agentui.Msg) (agentui.Model, agentui.Cmd) {
	switch msg := msg.(type) {
	case agentui.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.input = a.input.SetWidth(msg.Width)
		a.suggest = a.suggest.SetWidth(msg.Width)
		a.configureMarkdownRenderer()
		a.markAssistantDirty()
		a.updateSuggestions()
		a.resize()
		a.refreshPopup()
		a.rebuild()
		return a, nil
	case agentui.KeyMsg:
		if msg.Type == agentui.KeyCtrlC {
			return a, agentui.Quit
		}
		if a.popup.Open {
			var handled bool
			a.popup, handled = a.popup.Update(msg)
			if handled {
				return a, nil
			}
			return a, nil
		}
		switch msg.Type {
		case agentui.KeyCtrlO:
			if a.popup.Open {
				a.popup.Open = false
			} else {
				a.openPopup()
			}
			return a, nil
		case agentui.KeyCtrlP:
			cmd := a.commitLiveTurns()
			a.rebuild()
			return a, cmd
		case agentui.KeyUp:
			if a.suggest.Visible() {
				a.suggest = a.suggest.CursorUp()
				return a, nil
			}
		case agentui.KeyDown:
			if a.suggest.Visible() {
				a.suggest = a.suggest.CursorDown()
				return a, nil
			}
		case agentui.KeyTab:
			if a.applySelectedSuggestion() {
				return a, nil
			}
		}
		switch msg.Type {
		case agentui.KeyUp:
			if a.input.AtFirstLine() {
				nextHistory, value, ok := a.history.Prev(a.input.Value())
				if ok {
					a.history = nextHistory
					a.input = a.input.SetValue(value)
					return a, nil
				}
			}
		case agentui.KeyDown:
			if a.input.AtLastLine() {
				nextHistory, value, ok := a.history.Next(a.input.Value())
				if ok {
					a.history = nextHistory
					a.input = a.input.SetValue(value)
					return a, nil
				}
			}
		case agentui.KeyRunes:
			if msg.Paste {
				a.input = a.input.InsertString(a.pastes.Insert(string(msg.Runes)))
				a.updateSuggestions()
				return a, nil
			}
		}
		var cmd agentui.Cmd
		a.input, cmd = a.input.Update(msg)
		a.updateSuggestions()
		return a, cmd
	case editor.SubmitMsg:
		text := strings.TrimSpace(a.input.Value())
		if text == "" {
			return a, nil
		}
		text = a.pastes.Expand(text)
		if text == "/commit" {
			cmd := a.commitLiveTurns()
			a.input = a.input.Reset()
			a.updateSuggestions()
			a.rebuild()
			return a, cmd
		}
		if a.handleCommand(text) {
			a.input = a.input.Reset()
			a.updateSuggestions()
			a.rebuild()
			return a, nil
		}
		a.history = a.history.Record(text)
		a.appendLiveTurn(text)
		a.input = a.input.Reset()
		a.updateSuggestions()
		a.rebuild()
		return a, nil
	}
	return a, nil
}

func (a *app) View() string {
	body := renderutil.WrapANSI(a.view.Content(), a.width)
	if a.popup.Open {
		body = a.popup.View()
	}
	footerText := ansi.Truncate(" / suggestions  Tab apply  Ctrl+O popup  Ctrl+P commit  Ctrl+C quit", a.width, "")
	footer := style.New().Foreground(style.Color("240")).BorderTop(true).Width(a.width).Render(footerText)
	return style.JoinVertical(body, a.suggest.View(), a.input.View(), footer)
}

func (a *app) resize() {
	a.input = a.input.SetWidth(a.width)
	a.suggest = a.suggest.SetWidth(a.width)
	footerHeight := 2
	inputHeight := style.Height(a.input.View())
	suggestHeight := style.Height(a.suggest.View())
	a.view = a.view.SetSize(a.width, max(1, a.height-inputHeight-suggestHeight-footerHeight))
}

func (a *app) rebuild() {
	a.resize()
	a.view.SetContent(a.renderTranscriptContent())
	a.view.GotoBottom()
}

func newInput(width int) editor.Model {
	box := style.New().
		Background(style.Color("236")).
		Border(style.RoundedBorder()).
		BorderForeground(style.Color("63")).
		Padding(0, 1)
	cursor := style.New().Background(style.Color("236")).Reverse(true)
	return editor.New(width).
		SetPlaceholder("Type here. Try 1111*22222*3333 or paste multiple lines.").
		SetPrompt("> ").
		SetStyle(box).
		SetCursorStyle(cursor)
}

func (a *app) openPopup() {
	a.popup = a.newPopup()
	a.suggest = a.suggest.Update("")
}

func (a *app) refreshPopup() {
	if !a.popup.Open {
		return
	}
	offset := a.popup.Offset
	pinned := a.popup.PinnedBottom
	a.popup = a.newPopup()
	a.popup.Offset = offset
	a.popup.PinnedBottom = pinned
	a.popup = a.popup.SetLines(a.popup.Lines)
}

func (a *app) newPopup() overlay.Model {
	width := a.width
	if width <= 0 {
		width = 80
	}
	height := a.view.Height
	if height <= 0 {
		height = max(8, a.height-6)
	}
	popup := overlay.New("Regression popup: 输入框 / Markdown / 宽度", width, max(6, height))
	popup.PinnedBottom = false
	popup.Status = "Esc/q/Ctrl+O close  Up/Down scroll  Home/End jump"
	return popup.SetLines(a.popupLines(width - 6))
}

func (a *app) popupLines(width int) []string {
	if width < 20 {
		width = 20
	}
	raw := strings.Join([]string{
		"### 弹框回归内容",
		"",
		"- Inline emphasis: 1111*22222*3333",
		"- CJK mixed path: 用户查看 `internal/tui/renderutil/ansi_wrap.go` 文件内容",
		"- Long token: `/home/free/src/vibecoding/internal/tui/components/editor/editor.go`",
		"",
		"| Area | Case |",
		"| --- | --- |",
		"| input | Home/End, Alt+Enter, Ctrl+J, paste marker |",
		"| popup | fixed height, scroll, CJK width, status truncation |",
	}, "\n")
	rendered := renderutil.WrapANSI(renderutil.RenderMarkdown(raw, width), width)
	lines := strings.Split(rendered, "\n")
	lines = append(lines, "")
	lines = append(lines, strings.Split(renderutil.WrapPlainText("Plain wrap: 用户查看想 AGENTS 文件内容，并继续检查一个很长的路径 internal/tui/tool_modal.go 是否保持顺序。", width), "\n")...)
	for i := 1; i <= 18; i++ {
		line := "scroll row " + strconv.Itoa(i) + ": 用户查看 AGENTS.md and examples/minimal without losing the first character"
		lines = append(lines, strings.Split(renderutil.WrapPlainText(line, width), "\n")...)
	}
	return lines
}

func commandSuggestionItems() []suggest.Item {
	return []suggest.Item{
		{Label: "/popup", Value: "/popup", Description: "open regression popup"},
		{Label: "/commit", Value: "/commit", Description: "print active turn to scrollback"},
		{Label: "/markdown", Value: "/markdown", Description: "append markdown regression text"},
		{Label: "/mode", Value: "/mode ", Description: "show argument suggestions"},
		{Label: "/clear", Value: "/clear", Description: "clear transcript"},
		{Label: "/help", Value: "/help", Description: "append command help"},
	}
}

func modeSuggestionItems() []suggest.Item {
	return []suggest.Item{
		{Label: "/mode plan", Value: "/mode plan", Description: "argument suggestion"},
		{Label: "/mode agent", Value: "/mode agent", Description: "argument suggestion"},
		{Label: "/mode yolo", Value: "/mode yolo", Description: "argument suggestion"},
	}
}

func (a *app) updateSuggestions() {
	value := a.input.Value()
	if a.popup.Open || !strings.HasPrefix(value, "/") || strings.Contains(value, "\n") {
		a.suggest = a.suggest.SetItems(commandSuggestionItems()).Update("")
		return
	}
	if strings.HasPrefix(value, "/mode ") {
		a.suggest = a.suggest.SetItems(modeSuggestionItems()).Update(value)
		return
	}
	if strings.ContainsAny(value, " \t") {
		a.suggest = a.suggest.SetItems(commandSuggestionItems()).Update("")
		return
	}
	a.suggest = a.suggest.SetItems(commandSuggestionItems()).Update(value)
}

func (a *app) applySelectedSuggestion() bool {
	item, ok := a.suggest.Selected()
	if !ok || item.Value == "" || item.Value == a.input.Value() {
		return false
	}
	a.input = a.input.SetValue(item.Value).CursorEnd()
	a.updateSuggestions()
	a.resize()
	return true
}

func (a *app) handleCommand(text string) bool {
	switch strings.TrimSpace(text) {
	case "/popup":
		a.openPopup()
	case "/markdown":
		a.addAssistantMarkdown("### Manual markdown regression\n\n用户查看 *AGENTS* 文件内容，并检查 `internal/tui/renderutil/ansi_wrap.go`。\n\n- 1111*22222*3333\n- `/home/free/src/vibecoding/internal/tui/components/editor/editor.go`\n")
	case "/clear":
		a.messages = nil
		a.turns = nil
		a.assistantRaw = make(map[int]string)
		a.assistantRendered = make(map[int]string)
		a.assistantDirty = make(map[int]bool)
		a.addAssistantMarkdown("Transcript cleared.")
	case "/help":
		a.addAssistantMarkdown("### Minimal commands\n\n- `/popup` opens the regression popup.\n- `/commit` prints the active turn to terminal scrollback.\n- `/markdown` appends Markdown/CJK/path content.\n- `/mode ` shows argument suggestions.\n- `/clear` clears the transcript.\n")
	case "/mode plan", "/mode agent", "/mode yolo":
		a.addAssistantMarkdown("Mode suggestion applied: `" + text + "`")
	default:
		return false
	}
	return true
}

func (a *app) appendLiveTurn(text string) {
	a.turns = append(a.turns, turn{
		user:         text,
		assistantRaw: echoMarkdown(text),
	})
}

func (a *app) commitLiveTurns() agentui.Cmd {
	if len(a.turns) == 0 {
		return nil
	}
	blocks := make([]string, 0, len(a.turns))
	for _, turn := range a.turns {
		blocks = append(blocks, a.renderTurn(turn))
	}
	a.turns = nil
	return agentui.Println(strings.Join(blocks, "\n\n"))
}

func (a *app) renderCompletedAssistantMessage(raw string) string {
	prefix := style.New().Foreground(style.Color("15")).Render("Assistant: ")
	width := a.assistantMarkdownWidth()
	if renderutil.LooksLikeMarkdown(raw) {
		rendered := renderutil.RenderMarkdown(raw, width)
		if strings.TrimSpace(renderutil.StripANSI(rendered)) != "" {
			return prefix + renderutil.WrapANSI(rendered, width)
		}
	}
	return prefix + renderutil.WrapPlainText(raw, width)
}

func (a *app) renderTranscriptContent() string {
	blocks := make([]string, 0, len(a.messages))
	for idx := range a.messages {
		rendered := strings.TrimRight(a.renderMessageAt(idx), "\n")
		if strings.TrimSpace(renderutil.StripANSI(rendered)) == "" {
			continue
		}
		blocks = append(blocks, rendered)
	}
	if live := a.renderLiveTurns(); live != "" {
		blocks = append(blocks, live)
	}
	return strings.Join(blocks, "\n\n")
}

func (a *app) renderLiveTurns() string {
	if len(a.turns) == 0 {
		return ""
	}
	blocks := make([]string, 0, len(a.turns))
	for _, turn := range a.turns {
		blocks = append(blocks, a.renderTurn(turn))
	}
	return strings.Join(blocks, "\n\n")
}

func (a *app) renderTurn(turn turn) string {
	user := style.New().Foreground(style.Color("86")).Bold(true).Render("You: ") + turn.user
	return user + "\n" + a.renderCompletedAssistantMessage(turn.assistantRaw)
}

func (a *app) renderMessageAt(idx int) string {
	if _, ok := a.assistantRaw[idx]; ok {
		return a.renderAssistantMessage(idx)
	}
	if idx >= 0 && idx < len(a.messages) {
		return a.messages[idx]
	}
	return ""
}

func (a *app) addAssistantMarkdown(raw string) {
	idx := len(a.messages)
	a.messages = append(a.messages, "")
	a.assistantRaw[idx] = raw
	a.assistantDirty[idx] = true
}

func (a *app) renderAssistantMessage(idx int) string {
	raw := a.assistantRaw[idx]
	if raw == "" {
		return ""
	}
	prefix := style.New().Foreground(style.Color("15")).Render("Assistant: ")
	width := a.assistantMarkdownWidth()
	if renderutil.LooksLikeMarkdown(raw) {
		if a.assistantDirty[idx] && a.mdRenderer != nil {
			a.assistantRendered[idx] = a.mdRenderer.Render(raw)
			a.assistantDirty[idx] = false
		}
		if rendered := a.assistantRendered[idx]; strings.TrimSpace(renderutil.StripANSI(rendered)) != "" {
			return prefix + renderutil.WrapANSI(rendered, width)
		}
	}
	return prefix + renderutil.WrapPlainText(raw, width)
}

func (a *app) configureMarkdownRenderer() {
	a.mdRenderer = renderutil.NewMarkdownRenderer(a.assistantMarkdownWidth())
}

func (a *app) markAssistantDirty() {
	for idx := range a.assistantRaw {
		a.assistantDirty[idx] = true
	}
}

func (a *app) assistantMarkdownWidth() int {
	width := a.width
	if width <= 0 {
		width = 80
	}
	width -= renderutil.VisibleWidth("Assistant: ")
	if width < 1 {
		return 1
	}
	return width
}

func echoMarkdown(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		text = "(empty)"
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return "### Echo\n\nYou wrote:\n\n" + text + "\n\n- Rendered through `GoStreamingMarkdown`.\n"
}

func main() {
	p := agentui.NewProgram(newApp(), agentui.WithReportFocus())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
