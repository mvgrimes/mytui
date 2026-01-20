package repl

import (
	"github.com/c-bata/go-prompt"
	"github.com/mvgrimes/mycli-go/internal/vim"
)

func (r *REPL) getVimKeyBindings() []prompt.KeyBind {
	return []prompt.KeyBind{
		{
			Key: prompt.Escape,
			Fn: func(buf *prompt.Buffer) {
				r.vimState.Mode = vim.NormalMode
			},
		},
	}
}

func (r *REPL) getVimASCIICodeBindings() []prompt.ASCIICodeBind {
	v := r.vimState
	return []prompt.ASCIICodeBind{
		{
			ASCIICode: []byte("h"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == vim.NormalMode {
					buf.CursorLeft(1)
				} else {
					buf.InsertText("h", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("l"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == vim.NormalMode {
					buf.CursorRight(1)
				} else {
					buf.InsertText("l", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("j"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == vim.NormalMode {
					// In Normal mode, 'j' usually goes down in history or multiline
				} else {
					buf.InsertText("j", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("k"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == vim.NormalMode {
					// In Normal mode, 'k' usually goes up in history
				} else {
					buf.InsertText("k", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("i"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == vim.NormalMode {
					v.Mode = vim.InsertMode
				} else {
					buf.InsertText("i", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("a"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == vim.NormalMode {
					buf.CursorRight(1)
					v.Mode = vim.InsertMode
				} else {
					buf.InsertText("a", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("0"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == vim.NormalMode {
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
				if v.Mode == vim.NormalMode {
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
				if v.Mode == vim.NormalMode {
					buf.Delete(1)
				} else {
					buf.InsertText("x", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("w"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == vim.NormalMode {
					buf.CursorRight(buf.Document().FindEndOfCurrentWordWithSpace())
				} else {
					buf.InsertText("w", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("b"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == vim.NormalMode {
					buf.CursorLeft(len([]rune(buf.Document().TextBeforeCursor())) - buf.Document().FindStartOfPreviousWordWithSpace())
				} else {
					buf.InsertText("b", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("I"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == vim.NormalMode {
					x := []rune(buf.Document().TextBeforeCursor())
					buf.CursorLeft(len(x))
					v.Mode = vim.InsertMode
				} else {
					buf.InsertText("I", false, true)
				}
			},
		},
		{
			ASCIICode: []byte("A"),
			Fn: func(buf *prompt.Buffer) {
				if v.Mode == vim.NormalMode {
					x := []rune(buf.Document().TextAfterCursor())
					buf.CursorRight(len(x))
					v.Mode = vim.InsertMode
				} else {
					buf.InsertText("A", false, true)
				}
			},
		},
	}
}
