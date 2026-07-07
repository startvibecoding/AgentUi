# agentui

`agentui` is a small, stdlib-only Go TUI toolkit focused on code-agent terminal UIs.

It intentionally covers a narrow set of capabilities:

- Bubble Tea-like `Model`, `Msg`, `Cmd`, `Program`, `Batch`, `Tick`, and `Quit`.
- Keyboard, mouse-wheel, paste, window-size, and focus messages.
- ANSI-aware width, wrapping, stripping, and truncation helpers.
- Minimal styling: colors, bold/italic/reverse, padding, borders, width/height.
- Code-agent widgets: multi-line editor, viewport, suggestions, menu, overlay.

The project has no third-party dependencies.

## Packages

- `agentui`: program runtime, commands, key/mouse/window messages.
- `ansi`: ANSI-aware width, wrapping, stripping, and truncation.
- `style`: small terminal styling and layout helpers.
- `editor`: code-agent prompt editor with multi-line input.
- `viewport`: fixed-size transcript viewport.
- `suggest`: command suggestion dropdown.
- `menu`: reusable selectable dialog/list.
- `overlay`: reusable scrollable details overlay.
- `timer` and `spinner`: small replacements for stopwatch/spinner state.
- `paste` and `history`: code-agent input helpers.

Run tests:

```sh
go test ./...
```

Try the minimal demo:

```sh
go run ./examples/minimal
```
