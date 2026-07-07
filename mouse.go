package agentui

// MouseAction describes the kind of mouse event.
type MouseAction int

const (
	MouseActionUnknown MouseAction = iota
	MouseActionPress
	MouseActionRelease
)

// MouseButton describes the button involved in a mouse event.
type MouseButton int

const (
	MouseButtonUnknown MouseButton = iota
	MouseButtonLeft
	MouseButtonMiddle
	MouseButtonRight
	MouseButtonWheelUp
	MouseButtonWheelDown
)

// MouseMsg describes one mouse event.
type MouseMsg struct {
	X      int
	Y      int
	Action MouseAction
	Button MouseButton
}
