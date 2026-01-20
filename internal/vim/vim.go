package vim

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

func (v *VimState) GetLivePrefix(basePrompt string) func() (string, bool) {
	return func() (string, bool) {
		if v.Mode == NormalMode {
			return "(normal) " + basePrompt, true
		}
		return basePrompt, true
	}
}
