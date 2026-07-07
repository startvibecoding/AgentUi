package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/startvibecoding/agentui"
	"github.com/startvibecoding/agentui/editor"
	"github.com/startvibecoding/agentui/history"
	"github.com/startvibecoding/agentui/paste"
	"github.com/startvibecoding/agentui/style"
	"github.com/startvibecoding/agentui/viewport"
)

type app struct {
	width    int
	height   int
	input    editor.Model
	view     viewport.Model
	history  history.Model
	pastes   *paste.Manager
	messages []string
}

func newApp() *app {
	return &app{
		width:   80,
		height:  24,
		input:   editor.New(80).SetPlaceholder("Type a message..."),
		view:    viewport.New(80, 20),
		history: history.New(200),
		pastes:  paste.Default(),
		messages: []string{
			style.New().Foreground(style.Color("86")).Bold(true).Render("agentui minimal demo"),
			"Enter submits. Alt+Enter/Ctrl+J inserts a newline. Ctrl+C quits.",
		},
	}
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
		a.resize()
		a.rebuild()
		return a, nil
	case agentui.KeyMsg:
		switch msg.Type {
		case agentui.KeyCtrlC:
			return a, agentui.Quit
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
				return a, nil
			}
		}
		var cmd agentui.Cmd
		a.input, cmd = a.input.Update(msg)
		return a, cmd
	case editor.SubmitMsg:
		text := strings.TrimSpace(a.input.Value())
		if text == "" {
			return a, nil
		}
		text = a.pastes.Expand(text)
		a.history = a.history.Record(text)
		a.messages = append(a.messages, style.New().Foreground(style.Color("86")).Bold(true).Render("You: ")+text)
		a.messages = append(a.messages, style.New().Foreground(style.Color("240")).Render("Echo: ")+text)
		a.input = a.input.Reset()
		a.rebuild()
	}
	return a, nil
}

func (a *app) View() string {
	footer := style.New().Foreground(style.Color("240")).BorderTop(true).Width(a.width).Render(" Ctrl+C quit")
	return style.JoinVertical(a.view.View(), a.input.View(), footer)
}

func (a *app) resize() {
	footerHeight := 2
	inputHeight := style.Height(a.input.View())
	a.view = a.view.SetSize(a.width, max(1, a.height-inputHeight-footerHeight))
}

func (a *app) rebuild() {
	a.resize()
	a.view.SetContent(strings.Join(a.messages, "\n\n"))
	a.view.GotoBottom()
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
