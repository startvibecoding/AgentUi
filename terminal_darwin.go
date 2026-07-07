//go:build darwin

package agentui

import (
	"os"
	"os/signal"
	"syscall"
	"unsafe"
)

type terminalState struct {
	fd  int
	old syscall.Termios
	raw bool
}

func makeRaw(f *os.File) (*terminalState, error) {
	fd := int(f.Fd())
	var old syscall.Termios
	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TIOCGETA), uintptr(unsafe.Pointer(&old)), 0, 0, 0); errno != 0 {
		return nil, errno
	}
	raw := old
	raw.Iflag &^= syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Cflag |= syscall.CS8
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN | syscall.ISIG
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TIOCSETA), uintptr(unsafe.Pointer(&raw)), 0, 0, 0); errno != 0 {
		return nil, errno
	}
	return &terminalState{fd: fd, old: old, raw: true}, nil
}

func (t *terminalState) restore() {
	if t == nil || !t.raw {
		return
	}
	_, _, _ = syscall.Syscall6(syscall.SYS_IOCTL, uintptr(t.fd), uintptr(syscall.TIOCSETA), uintptr(unsafe.Pointer(&t.old)), 0, 0, 0)
	t.raw = false
}

type winsize struct {
	rows uint16
	cols uint16
	x    uint16
	y    uint16
}

func terminalSize(f *os.File) (int, int, bool) {
	var ws winsize
	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws)), 0, 0, 0); errno != 0 {
		return 0, 0, false
	}
	if ws.cols == 0 || ws.rows == 0 {
		return 0, 0, false
	}
	return int(ws.cols), int(ws.rows), true
}

func (p *Program) watchResize(f *os.File) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	defer signal.Stop(ch)
	for {
		select {
		case <-p.done:
			return
		case <-ch:
			if w, h, ok := terminalSize(f); ok {
				p.Send(WindowSizeMsg{Width: w, Height: h})
			}
		}
	}
}
