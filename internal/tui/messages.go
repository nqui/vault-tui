package tui

import (
	"github.com/nq/hv-tui/internal/vault"
)

type EnginesLoadedMsg struct {
	Engines []vault.EngineInfo
	Err     error
}

type PathListedMsg struct {
	NodeID  string
	Entries []vault.PathEntry
	Err     error
}

type SecretLoadedMsg struct {
	Path   string
	Secret *vault.SecretEntry
	Err    error
}

type SecretSavedMsg struct {
	Path string
	Err  error
}

type SecretDeletedMsg struct {
	Path string
	Err  error
}

type VersionsLoadedMsg struct {
	Path     string
	Versions []vault.VersionInfo
	Err      error
}

type ClearErrorMsg struct{}
