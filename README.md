# agentui

[中文文档](readme_zh.md)

`agentui` is a compact Go TUI toolkit for code-agent style terminal
applications. It gives you a Bubble Tea-like runtime, ANSI-aware layout helpers,
and a focused set of widgets for chat/transcript UIs without requiring Bubble
Tea or Lip Gloss in your application.

The module path is:

```sh
go get github.com/startvibecoding/agentui
```

`renderutil` and the minimal example use
`github.com/startvibecoding/GoStreamingMarkdown` plus agentui's internal ANSI
helpers for Markdown rendering compatible with the vibecoding TUI path.

## Features

- Bubble Tea-like `Model`, `Msg`, `Cmd`, `Program`, `Batch`, `Sequence`,
  `Tick`, `Print`, `Println`, `Repaint`, and `Quit`.
- Raw keyboard input on Linux and macOS, including bracketed paste, Escape
  timeout handling, focus events, mouse wheel events, and resize events.
- ANSI-aware width, wrapping, stripping, truncation, and fixed-width rendering.
- Code-agent UI widgets: multi-line editor, transcript viewport, command
  suggestions, selectable menu, fixed-height overlay, paste manager, history,
  spinner, and timer.
- Test-friendly design: drive models with messages, inspect `View()` output,
  and use `ParseInput` to unit-test terminal input parsing.

## Quick Start

```go
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/startvibecoding/agentui"
	"github.com/startvibecoding/agentui/editor"
	"github.com/startvibecoding/agentui/style"
)

type app struct {
	input editor.Model
	lines []string
}

func newApp() *app {
	return &app{
		input: editor.New(80).SetPlaceholder("Type a message..."),
		lines: []string{"agentui demo"},
	}
}

func (a *app) Init() agentui.Cmd { return nil }

func (a *app) Update(msg agentui.Msg) (agentui.Model, agentui.Cmd) {
	switch msg := msg.(type) {
	case agentui.KeyMsg:
		if msg.Type == agentui.KeyCtrlC {
			return a, agentui.Quit
		}
		var cmd agentui.Cmd
		a.input, cmd = a.input.Update(msg)
		return a, cmd
	case editor.SubmitMsg:
		text := strings.TrimSpace(a.input.Value())
		if text != "" {
			a.lines = append(a.lines, "You: "+text)
		}
		a.input = a.input.Reset()
		return a, nil
	}
	return a, nil
}

func (a *app) View() string {
	header := style.New().Foreground(style.Color("86")).Bold(true).Render("agentui")
	return style.JoinVertical(header, strings.Join(a.lines, "\n"), a.input.View())
}

func main() {
	if _, err := agentui.NewProgram(newApp()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

Run the included regression/demo app:

```sh
go run ./examples/minimal
```

The minimal example includes a bordered input box, slash-command suggestions,
Markdown rendering, a scrollable popup overlay, paste handling, and width
regression content for manual testing.

## Runtime Model

An app implements `agentui.Model`:

```go
type Model interface {
	Init() agentui.Cmd
	Update(agentui.Msg) (agentui.Model, agentui.Cmd)
	View() string
}
```

`Update` receives messages and returns the next model plus an optional command.
Commands run asynchronously and may return another message.

Useful command helpers:

- `agentui.Batch(cmds...)`: run commands concurrently.
- `agentui.Sequence(cmds...)`: run commands one at a time and preserve order.
- `agentui.Tick(d, fn)`: send a delayed message.
- `agentui.Print(text)` and `agentui.Println(text)`: write completed content
  outside the managed render tree so it can live in terminal scrollback.
- `agentui.Repaint`: render again without calling `Update`.
- `agentui.Quit`: stop the program.

Useful program options:

- `agentui.WithInput(r)` and `agentui.WithOutput(w)` for tests or embedded use.
- `agentui.WithInputTTY()` to request raw input for an `*os.File`.
- `agentui.WithReportFocus()` to enable focus reports.
- `agentui.WithMouse()` to enable SGR mouse tracking.
- `agentui.WithAltScreen()` to use the terminal alternate screen.
- `agentui.WithLiveHeight(h)` to cap the normal-screen live layer in tests or
  embedded renderers; overflowing top rows are written to terminal scrollback.
- `agentui.WithoutBracketedPaste()` to disable bracketed paste mode.
- `agentui.WithEscTimeout(d)` to tune standalone Escape detection.
- `agentui.WithoutRenderer()` to drive the runtime without automatic rendering.

## Packages

| Package | Purpose |
| --- | --- |
| `agentui` | Runtime, commands, program options, key/mouse/window messages. |
| `ansi` | ANSI-aware width, wrapping, stripping, and truncation. |
| `renderutil` | Markdown rendering and vibecoding-compatible ANSI wrapping. |
| `style` | Small terminal styling and layout helpers. |
| `editor` | Multi-line prompt editor for code-agent input. |
| `viewport` | Fixed-size transcript viewport with wrapping and item append APIs. |
| `suggest` | Prefix-filtered command suggestion dropdown. |
| `menu` | Compact selectable modal/list. |
| `overlay` | Fixed-height scrollable details overlay. |
| `paste` | Large-paste marker storage and split-paste coalescing helpers. |
| `history` | Input history with draft restoration. |
| `spinner` | Small spinner state model. |
| `timer` | Stopwatch/timer state model. |

## Common Patterns

### Viewport

```go
vp := viewport.New(width, height)
vp.SetContent("line 1\nline 2")
vp.GotoBottom()

// Transcript-style item blocks:
vp = viewport.New(width, height).SetItems([]string{"User: hello"})
vp = vp.AppendItem("Assistant: hi")
```

### Suggestions

```go
s := suggest.New(width).SetItems([]suggest.Item{
	{Label: "/help", Value: "/help", Description: "show help"},
	{Label: "/mode", Value: "/mode ", Description: "choose a mode"},
})

s = s.Update(inputValue)
if s.Visible() {
	// render s.View(), handle Up/Down with CursorUp/CursorDown,
	// and apply s.Selected() on Tab.
}
```

### Overlay

```go
o := overlay.New("Details", width, height).
	SetLines([]string{"line 1", "line 2", "line 3"})

o, handled := o.Update(msg)
if handled {
	// overlay consumed the key or mouse message
}
```

### Markdown Rendering

```go
raw := "用户查看 *AGENTS* 文件内容，并打开 `internal/tui/renderutil/ansi_wrap.go`。"
rendered := renderutil.RenderMarkdown(raw, width)
wrapped := renderutil.WrapANSI(rendered, width)
```

Use `renderutil.LooksLikeMarkdown` if you want to keep plain prose on a simpler
text-wrapping path.

### Terminal Scrollback

In the normal terminal screen, `Program` renders with two layers:

- The upper layer is the terminal's native scrollback.
- The lower layer is the managed live view returned by `View()`.

When the live view grows beyond the terminal height, the renderer automatically
prints the overflowing top rows to native scrollback and keeps repainting the
remaining bottom rows as the live layer. This keeps mouse scrolling and terminal
selection on the terminal's own scrollback instead of an internal transcript
viewport. Alternate screen applications do not write to native scrollback.

For explicit completed blocks, you can still return `Print` or `Println`:

```go
func (a *app) completeTurn(user, assistant string) agentui.Cmd {
	a.live = "" // remove completed content from the managed view
	return agentui.Sequence(
		agentui.Println("You: "+user),
		agentui.Println("Assistant: "+assistant),
	)
}
```

The minimal example demonstrates both paths: Enter appends live turns to the
managed layer, overflow is automatically stitched into native scrollback, and
`Ctrl+P` or `/commit` can manually flush live turns.

## Testing

Run the normal checks:

```sh
go test ./...
```

This repository also includes a Makefile:

```sh
make check   # fmt, vet, test
make ci      # vet, test, race
make cross   # compile-test linux, macOS, and Windows targets
make run     # run examples/minimal
```

For TUI logic, prefer model-level tests:

- Send `agentui.KeyMsg`, `agentui.MouseMsg`, and `agentui.WindowSizeMsg` to
  `Update`.
- Assert component state and inspect ANSI-stripped `View()` output.
- Use `agentui.ParseInput` to test terminal byte sequences.
- Test style scope semantically when Markdown/ANSI styling matters.

The `examples/minimal` tests demonstrate this approach for Markdown order,
inline emphasis, input box width, popup behavior, suggestions, and command
application.

## Platform Support

- Linux: raw input, bracketed paste, focus/mouse escape modes, and resize events
  are implemented.
- macOS: raw input and resize events are implemented with Darwin terminal
  ioctls. Build coverage is checked for `darwin/amd64` and `darwin/arm64`.
- Windows: packages compile, tests compile, and ANSI rendering works in modern
  ANSI-capable terminals. Native Windows console raw mode is not implemented
  yet, so interactive input is limited compared with Linux/macOS.

Use `make cross` on a Unix-like host to compile-check Linux, macOS, and Windows
targets.

## Versioning Notes

This library is intended to be imported by other Go programs. Keep your
dependency pinned with normal Go module versions once releases are tagged:

```sh
go get github.com/startvibecoding/agentui@latest
```

Until a stable `v1` tag exists, review changelogs or diffs before upgrading.
