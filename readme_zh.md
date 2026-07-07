# agentui

[English](README.md)

`agentui` 是一个面向代码智能体终端界面的轻量 Go TUI 工具库。它提供类似
Bubble Tea 的运行时、ANSI 宽度/布局工具，以及一组适合聊天、执行日志、命令
输入、弹窗和自动提示的组件；应用侧不需要依赖 Bubble Tea 或 Lip Gloss。

模块路径：

```sh
go get github.com/startvibecoding/agentui
```

`renderutil` 包和 minimal 示例使用
`github.com/startvibecoding/GoStreamingMarkdown` 以及 agentui 内部 ANSI
helper，用于对齐 vibecoding TUI 当前的 Markdown 渲染和 ANSI 折行路径。

## 功能

- 类 Bubble Tea 的 `Model`、`Msg`、`Cmd`、`Program`、`Batch`、`Sequence`、
  `Tick`、`Print`、`Println`、`Repaint`、`Quit`。
- Linux/macOS raw keyboard input，支持 bracketed paste、Escape 超时判定、
  focus 事件、鼠标滚轮和窗口 resize。
- ANSI 感知的宽度计算、折行、去 ANSI、截断和固定宽度渲染。
- 面向代码智能体的 UI 组件：多行输入框、 transcript viewport、命令自动提
  示、菜单、固定高度弹窗、大粘贴管理、历史记录、spinner、timer。
- 方便测试：可以直接给 model 发送消息，检查组件状态和 `View()` 输出，也可
  用 `ParseInput` 单测终端输入字节序列。

## 快速开始

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

运行仓库内置的回归/演示程序：

```sh
go run ./examples/minimal
```

minimal 示例包含带边框输入框、slash 命令自动提示、Markdown 渲染、可滚动弹
窗、大粘贴处理，以及用于人工回归的 CJK、长路径、样式和宽度内容。

## 运行时模型

应用实现 `agentui.Model`：

```go
type Model interface {
	Init() agentui.Cmd
	Update(agentui.Msg) (agentui.Model, agentui.Cmd)
	View() string
}
```

`Update` 接收消息，返回下一个 model 和可选命令。命令异步运行，并可以返回新
消息。

常用命令：

- `agentui.Batch(cmds...)`：并发运行多个命令。
- `agentui.Sequence(cmds...)`：按顺序运行多个命令，并保持消息顺序。
- `agentui.Tick(d, fn)`：延迟发送消息。
- `agentui.Print(text)` 和 `agentui.Println(text)`：把已完成内容写到
  managed render tree 之外，让它进入终端原生 scrollback。
- `agentui.Repaint`：不调用 `Update`，只触发重新渲染。
- `agentui.Quit`：退出程序。

常用程序选项：

- `agentui.WithInput(r)` 和 `agentui.WithOutput(w)`：用于测试或嵌入场景。
- `agentui.WithInputTTY()`：当 input 是 `*os.File` 时请求 raw input。
- `agentui.WithReportFocus()`：启用 focus report。
- `agentui.WithMouse()`：启用 SGR mouse tracking。
- `agentui.WithAltScreen()`：使用终端 alternate screen。
- `agentui.WithLiveHeight(h)`：在测试或嵌入渲染里限制 normal-screen live
  层高度；超出的顶部行会写入终端原生 scrollback。
- `agentui.WithoutBracketedPaste()`：关闭 bracketed paste。
- `agentui.WithEscTimeout(d)`：调整单独 Escape 键的判定等待时间。
- `agentui.WithoutRenderer()`：关闭自动渲染，手动驱动运行时。

## 包说明

| 包 | 用途 |
| --- | --- |
| `agentui` | 运行时、命令、程序选项、键盘/鼠标/窗口消息。 |
| `ansi` | ANSI 感知的宽度、折行、去 ANSI 和截断。 |
| `renderutil` | Markdown 渲染和 vibecoding 兼容的 ANSI 折行。 |
| `style` | 小型终端样式和布局工具。 |
| `editor` | 面向代码智能体输入的多行编辑器。 |
| `viewport` | 固定尺寸 transcript viewport，支持折行和 item 追加。 |
| `suggest` | 前缀过滤的命令自动提示下拉框。 |
| `menu` | 紧凑的可选菜单/列表弹窗。 |
| `overlay` | 固定高度、可滚动的详情弹窗。 |
| `paste` | 大粘贴 marker 存储和 split paste 合并辅助函数。 |
| `history` | 输入历史和草稿恢复。 |
| `spinner` | 小型 spinner 状态模型。 |
| `timer` | 秒表/计时器状态模型。 |

## 常用组合方式

### Viewport

```go
vp := viewport.New(width, height)
vp.SetContent("line 1\nline 2")
vp.GotoBottom()

// transcript item 风格：
vp = viewport.New(width, height).SetItems([]string{"User: hello"})
vp = vp.AppendItem("Assistant: hi")
```

### 自动提示

```go
s := suggest.New(width).SetItems([]suggest.Item{
	{Label: "/help", Value: "/help", Description: "show help"},
	{Label: "/mode", Value: "/mode ", Description: "choose a mode"},
})

s = s.Update(inputValue)
if s.Visible() {
	// 渲染 s.View()；
	// 用 CursorUp/CursorDown 处理 Up/Down；
	// 用 s.Selected() 在 Tab 时应用选中项。
}
```

### 弹窗

```go
o := overlay.New("Details", width, height).
	SetLines([]string{"line 1", "line 2", "line 3"})

o, handled := o.Update(msg)
if handled {
	// 弹窗已经消费了这个键盘或鼠标消息
}
```

### Markdown 渲染

```go
raw := "用户查看 *AGENTS* 文件内容，并打开 `internal/tui/renderutil/ansi_wrap.go`。"
rendered := renderutil.RenderMarkdown(raw, width)
wrapped := renderutil.WrapANSI(rendered, width)
```

如果希望普通文本继续走更简单的折行路径，可以先用
`renderutil.LooksLikeMarkdown` 判断。

### 终端 Scrollback

在普通终端屏幕里，`Program` 按两层渲染：

- 上层是终端原生 scrollback。
- 下层是 `View()` 返回的 managed live view。

当 live view 高度超过终端高度时，renderer 会自动把顶部溢出的行打印到终端
原生 scrollback，只继续重绘底部剩余行作为 live 层。这样鼠标滚动和终端选择
使用的是终端自己的 scrollback，而不是应用内部 transcript viewport。开启
alternate screen 时不会写入终端原生 scrollback。

如果有明确已完成的 block，也仍然可以返回 `Print` 或 `Println`：

```go
func (a *app) completeTurn(user, assistant string) agentui.Cmd {
	a.live = "" // 从 managed view 移除已完成内容
	return agentui.Sequence(
		agentui.Println("You: "+user),
		agentui.Println("Assistant: "+assistant),
	)
}
```

minimal 示例展示两条路径：Enter 会持续向 managed live 层追加 turn，live
层溢出后会自动拼接进终端原生 scrollback；`Ctrl+P` 或 `/commit` 可以手动
flush 当前 live turns。

## 测试

运行普通测试：

```sh
go test ./...
```

仓库提供 Makefile：

```sh
make check   # fmt, vet, test
make ci      # vet, test, race
make cross   # 编译检查 Linux、macOS、Windows 目标
make run     # 运行 examples/minimal
```

TUI 逻辑建议优先写 model 级单元测试：

- 向 `Update` 发送 `agentui.KeyMsg`、`agentui.MouseMsg`、`agentui.WindowSizeMsg`。
- 检查组件状态，并对 `View()` 做 ANSI strip 后断言。
- 用 `agentui.ParseInput` 测试终端字节序列。
- 当 Markdown/ANSI 样式很重要时，用语义方式检查样式作用域。

`examples/minimal` 的测试覆盖了 Markdown 顺序、inline emphasis、输入框宽
度、弹窗行为、自动提示、命令应用等场景，可作为写 TUI 测试的参考。

## 平台支持

- Linux：已实现 raw input、bracketed paste、focus/mouse escape mode 和
  resize 事件。
- macOS：已通过 Darwin terminal ioctl 实现 raw input 和 resize 事件，并编
  译检查 `darwin/amd64`、`darwin/arm64`。
- Windows：包和测试可以编译，现代支持 ANSI 的终端中可以使用 ANSI 渲染。原
  生 Windows console raw mode 还没有实现，因此交互输入能力和 Linux/macOS
  不完全等价。

可在类 Unix 主机上运行 `make cross` 来编译检查 Linux、macOS、Windows 目标。

## 版本说明

这个库会作为其他 Go 程序的依赖使用。发布 tag 后，建议按普通 Go module 方
式固定版本：

```sh
go get github.com/startvibecoding/agentui@latest
```

在稳定 `v1` tag 之前，升级前建议先查看变更记录或 diff。
