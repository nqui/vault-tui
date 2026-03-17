package tui

import (
	"github.com/nqui/vault-tui/internal/vault"
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

type SecretWrappedMsg struct {
	Path  string
	Token string
	TTL   string
	Err   error
}

type SecretUnwrappedMsg struct {
	Data map[string]interface{}
	Err  error
}

type TokenValidatedMsg struct {
	Info *vault.TokenInfo
	Err  error
}

type LoginCompleteMsg struct {
	Info *vault.TokenInfo
	Save bool
	Err  error
}

type TokenRenewedMsg struct {
	Info *vault.TokenInfo
	Err  error
}

type TokenRenewTickMsg struct{}

type ClearErrorMsg struct{}
