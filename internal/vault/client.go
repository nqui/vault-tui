package vault

import (
	vaultapi "github.com/hashicorp/vault/api"
)

type Client struct {
	raw *vaultapi.Client
}

func New(addr, token string) (*Client, error) {
	cfg := vaultapi.DefaultConfig()
	cfg.Address = addr

	raw, err := vaultapi.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	raw.SetToken(token)

	return &Client{raw: raw}, nil
}

func (c *Client) Addr() string {
	return c.raw.Address()
}

func (c *Client) SetToken(token string) {
	c.raw.SetToken(token)
}

func (c *Client) HasToken() bool {
	return c.raw.Token() != ""
}
