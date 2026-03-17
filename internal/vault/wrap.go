package vault

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type WrapResult struct {
	Token        string
	TTL          string
	CreationTime time.Time
}

func (c *Client) WrapSecret(ctx context.Context, engine, path string, kvVersion int, ttl string) (*WrapResult, error) {
	clone, err := c.raw.Clone()
	if err != nil {
		return nil, wrapError("wrap", engine+path, err)
	}
	clone.SetWrappingLookupFunc(func(operation, p string) string {
		return ttl
	})

	var fullPath string
	if kvVersion == 2 {
		fullPath = engine + "data/" + path
	} else {
		fullPath = engine + path
	}

	secret, err := clone.Logical().ReadWithContext(ctx, fullPath)
	if err != nil {
		return nil, wrapError("wrap", fullPath, err)
	}
	if secret == nil {
		return nil, wrapError("wrap", fullPath, ErrNotFound)
	}
	if secret.WrapInfo == nil {
		return nil, wrapError("wrap", fullPath, fmt.Errorf("no wrap info returned"))
	}

	return &WrapResult{
		Token:        secret.WrapInfo.Token,
		TTL:          fmt.Sprintf("%ds", secret.WrapInfo.TTL),
		CreationTime: secret.WrapInfo.CreationTime,
	}, nil
}

func (c *Client) WrapData(ctx context.Context, data map[string]interface{}, ttl string) (*WrapResult, error) {
	clone, err := c.raw.Clone()
	if err != nil {
		return nil, wrapError("wrap", "sys/wrapping/wrap", err)
	}
	clone.SetWrappingLookupFunc(func(operation, path string) string {
		return ttl
	})

	secret, err := clone.Logical().WriteWithContext(ctx, "sys/wrapping/wrap", data)
	if err != nil {
		return nil, wrapError("wrap", "sys/wrapping/wrap", err)
	}
	if secret == nil || secret.WrapInfo == nil {
		return nil, wrapError("wrap", "sys/wrapping/wrap", fmt.Errorf("no wrap info returned"))
	}

	return &WrapResult{
		Token:        secret.WrapInfo.Token,
		TTL:          fmt.Sprintf("%ds", secret.WrapInfo.TTL),
		CreationTime: secret.WrapInfo.CreationTime,
	}, nil
}

func (c *Client) UnwrapToken(ctx context.Context, wrappingToken string) (map[string]interface{}, error) {
	secret, err := c.raw.Logical().UnwrapWithContext(ctx, wrappingToken)
	if err != nil {
		return nil, wrapError("unwrap", strings.TrimSpace(wrappingToken)[:min(16, len(strings.TrimSpace(wrappingToken)))]+"...", err)
	}
	if secret == nil {
		return nil, wrapError("unwrap", "", fmt.Errorf("no data returned"))
	}

	// For KV v2 responses, data is nested under data.data
	if innerData, ok := secret.Data["data"]; ok {
		if nested, ok := innerData.(map[string]interface{}); ok {
			return nested, nil
		}
	}

	return secret.Data, nil
}
