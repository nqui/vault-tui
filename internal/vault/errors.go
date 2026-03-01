package vault

import (
	"errors"
	"fmt"
	"net/http"

	vaultapi "github.com/hashicorp/vault/api"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrPermissionDenied = errors.New("permission denied")
	ErrConnectionFailed = errors.New("connection failed")
)

type VaultError struct {
	Op      string
	Path    string
	Wrapped error
}

func (e *VaultError) Error() string {
	return fmt.Sprintf("vault %s %s: %v", e.Op, e.Path, e.Wrapped)
}

func (e *VaultError) Unwrap() error { return e.Wrapped }

func wrapError(op, path string, err error) error {
	if err == nil {
		return nil
	}
	wrapped := classifyError(err)
	return &VaultError{Op: op, Path: path, Wrapped: wrapped}
}

func classifyError(err error) error {
	var respErr *vaultapi.ResponseError
	if errors.As(err, &respErr) {
		switch respErr.StatusCode {
		case http.StatusNotFound:
			return fmt.Errorf("%w: %s", ErrNotFound, respErr.Error())
		case http.StatusForbidden:
			return fmt.Errorf("%w: %s", ErrPermissionDenied, respErr.Error())
		}
	}
	return err
}
