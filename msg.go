package agentui

// Msg is an event delivered to a Model.
type Msg any

// Model is the core application interface.
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() string
}

// WindowSizeMsg reports the current terminal size in cells.
type WindowSizeMsg struct {
	Width  int
	Height int
}

// FocusMsg is emitted when the terminal reports focus gained.
type FocusMsg struct{}

// BlurMsg is emitted when the terminal reports focus lost.
type BlurMsg struct{}

// QuitMsg asks the Program to stop.
type QuitMsg struct{}

type repaintMsg struct{}
