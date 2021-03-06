// SPDX-License-Identifier: Unlicense OR MIT

package key

import (
	"gioui.org/ui"
	"gioui.org/ui/input"
	"gioui.org/ui/internal/ops"
)

type HandlerOp struct {
	Key   input.Key
	Focus bool
}

type HideInputOp struct{}

type FocusEvent struct {
	Focus bool
}

type ChordEvent struct {
	Name      rune
	Modifiers Modifiers
}

type EditEvent struct {
	Text string
}

type Modifiers uint32

const (
	ModCommand Modifiers = 1 << iota
	ModShift
)

const (
	NameLeftArrow      = '←'
	NameRightArrow     = '→'
	NameUpArrow        = '↑'
	NameDownArrow      = '↓'
	NameReturn         = '⏎'
	NameEnter          = '⌤'
	NameEscape         = '⎋'
	NameHome           = '⇱'
	NameEnd            = '⇲'
	NameDeleteBackward = '⌫'
	NameDeleteForward  = '⌦'
	NamePageUp         = '⇞'
	NamePageDown       = '⇟'
)

func (m Modifiers) Contain(m2 Modifiers) bool {
	return m&m2 == m2
}

func (h HandlerOp) Add(o *ui.Ops) {
	data := make([]byte, ops.TypeKeyHandlerLen)
	data[0] = byte(ops.TypeKeyHandler)
	if h.Focus {
		data[1] = 1
	}
	o.Write(data, h.Key)
}

func (h *HandlerOp) Decode(d []byte, refs []interface{}) {
	if ops.OpType(d[0]) != ops.TypeKeyHandler {
		panic("invalid op")
	}
	*h = HandlerOp{
		Focus: d[1] != 0,
		Key:   refs[0].(input.Key),
	}
}

func (h HideInputOp) Add(o *ui.Ops) {
	data := make([]byte, ops.TypeHideInputLen)
	data[0] = byte(ops.TypeHideInput)
	o.Write(data)
}

func (EditEvent) ImplementsEvent()       {}
func (ChordEvent) ImplementsEvent()      {}
func (FocusEvent) ImplementsEvent()      {}
func (EditEvent) ImplementsInputEvent()  {}
func (ChordEvent) ImplementsInputEvent() {}
func (FocusEvent) ImplementsInputEvent() {}
