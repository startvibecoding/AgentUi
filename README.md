# agentui

`agentui` is a small Go TUI toolkit focused on code-agent terminal UIs.

It intentionally covers a narrow set of capabilities:

- Bubble Tea-like `Model`, `Msg`, `Cmd`, `Program`, `Batch`, `Tick`, and `Quit`.
- Command composition with `Batch`, `Sequence`, `Tick`, `Repaint`, and `Quit`.
- Keyboard, mouse-wheel, paste, window-size, and focus messages.
- ANSI-aware width, wrapping, stripping, and truncation helpers.
- Minimal styling: colors, bold/italic/reverse, padding, borders, width/height.
- Code-agent widgets: multi-line editor, viewport, suggestions, menu, overlay.

The runtime and core widgets avoid Bubble Tea and Lip Gloss. The `renderutil`
package and minimal example use `github.com/startvibecoding/GoStreamingMarkdown`
and `github.com/charmbracelet/x/ansi` to mirror vibecoding's assistant Markdown
rendering path.

## Packages

- `agentui`: program runtime, commands, key/mouse/window messages.
- `ansi`: ANSI-aware width, wrapping, stripping, and truncation.
- `renderutil`: vibecoding-compatible ANSI wrapping and Markdown rendering.
- `style`: small terminal styling and layout helpers.
- `editor`: code-agent prompt editor with multi-line input.
- `viewport`: fixed-size transcript viewport.
- `suggest`: command suggestion dropdown.
- `menu`: reusable selectable dialog/list.
- `overlay`: reusable scrollable details overlay.
- `timer` and `spinner`: small replacements for stopwatch/spinner state.
- `paste` and `history`: code-agent input helpers.

## Runtime basics

An app implements `agentui.Model`:

```go
type Model interface {
    Init() agentui.Cmd
    Update(agentui.Msg) (agentui.Model, agentui.Cmd)
    View() string
}
```

`Batch` runs commands concurrently, while `Sequence` runs them one at a time
and forwards their messages in order. `Tick` schedules delayed work, `Repaint`
forces a render without calling `Update`, and `Quit` stops the program.

`Program` enables bracketed paste by default so large pasted blocks arrive as a
single paste message when the terminal supports it. Use
`agentui.WithoutBracketedPaste()` to disable that mode, and
`agentui.WithEscTimeout(d)` to tune how quickly a standalone Escape key is
reported.

## Platform Support

- Linux: raw input, bracketed paste, focus/mouse escape modes, and resize
  events are implemented.
- macOS: raw input and resize events are implemented with Darwin terminal
  ioctls. Build coverage is checked for `darwin/amd64` and `darwin/arm64`.
- Windows: packages compile, tests compile, and ANSI rendering works in modern
  ANSI-capable terminals. Native Windows console raw mode is not implemented
  yet, so interactive input is limited compared with Linux/macOS.

Use `make cross` on a Unix-like host to compile-check Linux, macOS, and Windows
targets.

Run tests:

```sh
go test ./...
```

Try the minimal demo:

```sh
go run ./examples/minimal
```

The demo mirrors the Markdown rendering path used by the `vibecoding` TUI:
assistant messages keep raw Markdown, render through `renderutil`, and then flow
into the viewport with ANSI-aware wrapping.
