package vim

import (
	"github.com/c-bata/go-prompt"
)

type Mode int

const (
	InsertMode Mode = iota
	NormalMode
)

type VimState struct {
	Mode Mode
}

func NewVimState() *VimState {
	return &VimState{
		Mode: InsertMode,
	}
}

func (v *VimState) GetKeyBindings() []prompt.KeyBind {
	return []prompt.KeyBind{
		{
			Key: prompt.Escape,
			Fn: func(buf *prompt.Buffer) {
				v.Mode = NormalMode
			},
		},
		{
			Key: prompt.Any,
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == InsertMode {
					return
				}

				// Handle Normal Mode
				key := buf.Document().LastKeyStroke()
				switch key {
				case prompt.Any:
					// prompt.Any is a catch-all, but we need the actual char
					// go-prompt doesn't easily expose the raw char here in a clean way
					// but we can try to infer it if it's a single byte.
					// Actually, go-prompt's handleKeyBinding is called AFTER commonKeyBindings.
				}
			},
		},
	}
}

func (v *VimState) GetASCIICodeBindings() []prompt.ASCIICodeBind {
	return []prompt.ASCIICodeBind{
		{
			ASCIICode: []byte("h"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == NormalMode {
					buf.CursorLeft(1)
				} else {
					buf.InsertText("h", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("l"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == NormalMode {
					buf.CursorRight(1)
				} else {
					buf.InsertText("l", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("j"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == NormalMode {
					// In Normal mode, 'j' usually goes down in history or multiline
					// For now, let's just make it do nothing or go to next history
				} else {
					buf.InsertText("j", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("k"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == NormalMode {
					// In Normal mode, 'k' usually goes up in history
				} else {
					buf.InsertText("k", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("i"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == NormalMode {
					v.Mode = InsertMode
				} else {
					buf.InsertText("i", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("a"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == NormalMode {
					buf.CursorRight(1)
					v.Mode = InsertMode
				} else {
					buf.InsertText("a", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("0"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == NormalMode {
					x := []rune(buf.Document().TextBeforeCursor())
					buf.CursorLeft(len(x))
				} else {
					buf.InsertText("0", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("$"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == NormalMode {
					x := []rune(buf.Document().TextAfterCursor())
					buf.CursorRight(len(x))
				} else {
					buf.InsertText("$", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("x"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == NormalMode {
					buf.Delete(1)
				} else {
					buf.InsertText("x", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("w"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == NormalMode {
					buf.CursorRight(buf.Document().FindEndOfCurrentWordWithSpace())
				} else {
					buf.InsertText("w", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("b"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == NormalMode {
					buf.CursorLeft(len([]rune(buf.Document().TextBeforeCursor())) - buf.Document().FindStartOfPreviousWordWithSpace())
				} else {
					buf.InsertText("b", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("I"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == NormalMode {
					x := []rune(buf.Document().TextBeforeCursor())
					buf.CursorLeft(len(x))
					v.Mode = InsertMode
				} else {
					buf.InsertText("I", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("A"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == NormalMode {
					x := []rune(buf.Document().TextAfterCursor())
					buf.CursorRight(len(x))
					v.Mode = InsertMode
				} else {
					buf.InsertText("A", false, true)
				}
			},
		},
	}
}

func (v *VimState) GetLivePrefix() func() (string, bool) {
	return func() (string, bool) {
		if v.Mode == NormalMode {
			return "(normal) mysql> ", true
		}
		return "mysql> ", true
	}
}
