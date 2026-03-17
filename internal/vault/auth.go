package vault

import (
	"context"
	"fmt"
	"time"
)

type TokenInfo struct {
	Token     string
	TTL       time.Duration
	Renewable bool
}

func (c *Client) ValidateToken(ctx context.Context) (*TokenInfo, error) {
	secret, err := c.raw.Auth().Token().LookupSelfWithContext(ctx)
	if err != nil {
		return nil, wrapError("validate-token", "auth/token/lookup-self", err)
	}

	ttl, _ := secret.TokenTTL()
	renewable, _ := secret.TokenIsRenewable()

	return &TokenInfo{
		Token:     c.raw.Token(),
		TTL:       ttl,
		Renewable: renewable,
	}, nil
}

func (c *Client) LoginToken(ctx context.Context, token string) (*TokenInfo, error) {
	prev := c.raw.Token()
	c.raw.SetToken(token)

	info, err := c.ValidateToken(ctx)
	if err != nil {
		c.raw.SetToken(prev)
		return nil, fmt.Errorf("token login failed: %w", err)
	}

	return info, nil
}

func (c *Client) LoginUserpass(ctx context.Context, username, password string) (*TokenInfo, error) {
	return c.loginCredential(ctx, "userpass", username, password)
}

func (c *Client) LoginLDAP(ctx context.Context, username, password string) (*TokenInfo, error) {
	return c.loginCredential(ctx, "ldap", username, password)
}

func (c *Client) loginCredential(ctx context.Context, method, username, password string) (*TokenInfo, error) {
	path := fmt.Sprintf("auth/%s/login/%s", method, username)
	secret, err := c.raw.Logical().WriteWithContext(ctx, path, map[string]interface{}{
		"password": password,
	})
	if err != nil {
		return nil, wrapError("login", path, err)
	}
	if secret == nil || secret.Auth == nil {
		return nil, fmt.Errorf("login to %s returned no auth data", method)
	}

	c.raw.SetToken(secret.Auth.ClientToken)

	ttl := time.Duration(secret.Auth.LeaseDuration) * time.Second
	return &TokenInfo{
		Token:     secret.Auth.ClientToken,
		TTL:       ttl,
		Renewable: secret.Auth.Renewable,
	}, nil
}

func (c *Client) RenewToken(ctx context.Context) (*TokenInfo, error) {
	secret, err := c.raw.Auth().Token().RenewSelfWithContext(ctx, 0)
	if err != nil {
		return nil, wrapError("renew-token", "auth/token/renew-self", err)
	}

	ttl, _ := secret.TokenTTL()
	renewable, _ := secret.TokenIsRenewable()

	return &TokenInfo{
		Token:     c.raw.Token(),
		TTL:       ttl,
		Renewable: renewable,
	}, nil
}
