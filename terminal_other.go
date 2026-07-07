//go:build !linux

package agentui

import "os"

type terminalState struct{}

func makeRaw(*os.File) (*terminalState, error) { return nil, nil }
func (*terminalState) restore()                {}

func terminalSize(*os.File) (int, int, bool) { return 0, 0, false }

func (p *Program) watchResize(*os.File) {
	<-p.done
}
