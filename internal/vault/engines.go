package vault

import "context"

type EngineInfo struct {
	Path        string
	Type        string
	Description string
	Version     int // 0 = unknown/not KV, 1 = KV v1, 2 = KV v2
}

func (c *Client) ListEngines(ctx context.Context) ([]EngineInfo, error) {
	mounts, err := c.raw.Sys().ListMountsWithContext(ctx)
	if err != nil {
		return nil, wrapError("list", "sys/mounts", err)
	}

	var engines []EngineInfo
	for path, mount := range mounts {
		e := EngineInfo{
			Path:        path,
			Type:        mount.Type,
			Description: mount.Description,
		}
		if mount.Type == "kv" {
			if v, ok := mount.Options["version"]; ok && v == "2" {
				e.Version = 2
			} else {
				e.Version = 1
			}
		}
		engines = append(engines, e)
	}

	return engines, nil
}
