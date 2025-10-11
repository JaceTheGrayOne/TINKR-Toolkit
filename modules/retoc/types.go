package retoc

type Mod struct {
	Name        string
	DisplayName string
	Path        string
}

type BuildCompleteMsg struct {
	Log        string
	Err        error
	BuiltMods  []string
	FailedMods []string
}
