package vault

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

type PathEntry struct {
	Name  string
	IsDir bool
}

type SecretEntry struct {
	Path     string
	Data     map[string]interface{}
	Metadata *SecretMetadata
}

type SecretMetadata struct {
	Version     int
	CreatedTime time.Time
	UpdatedTime time.Time
}

type VersionInfo struct {
	Version     int
	CreatedTime time.Time
	Deleted     bool
	Destroyed   bool
}

func (c *Client) ListPath(ctx context.Context, engine, path string, kvVersion int) ([]PathEntry, error) {
	var fullPath string
	if kvVersion == 2 {
		fullPath = engine + "metadata/" + path
	} else {
		fullPath = engine + path
	}

	secret, err := c.raw.Logical().ListWithContext(ctx, fullPath)
	if err != nil {
		return nil, wrapError("list", fullPath, err)
	}
	if secret == nil || secret.Data == nil {
		return nil, nil
	}

	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return nil, nil
	}

	entries := make([]PathEntry, 0, len(keys))
	for _, k := range keys {
		name := fmt.Sprintf("%v", k)
		entries = append(entries, PathEntry{
			Name:  name,
			IsDir: strings.HasSuffix(name, "/"),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		// Directories first, then alphabetical
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})

	return entries, nil
}

func (c *Client) GetSecret(ctx context.Context, engine, path string, kvVersion int) (*SecretEntry, error) {
	entry := &SecretEntry{Path: engine + path}

	if kvVersion == 2 {
		kv := c.raw.KVv2(strings.TrimSuffix(engine, "/"))
		secret, err := kv.Get(ctx, path)
		if err != nil {
			return nil, wrapError("get", engine+path, err)
		}
		entry.Data = secret.Data
		if secret.VersionMetadata != nil {
			entry.Metadata = &SecretMetadata{
				Version:     secret.VersionMetadata.Version,
				CreatedTime: secret.VersionMetadata.CreatedTime,
			}
		}
	} else {
		secret, err := c.raw.Logical().ReadWithContext(ctx, engine+path)
		if err != nil {
			return nil, wrapError("get", engine+path, err)
		}
		if secret == nil {
			return nil, wrapError("get", engine+path, ErrNotFound)
		}
		entry.Data = secret.Data
	}

	return entry, nil
}

func (c *Client) GetSecretVersion(ctx context.Context, engine, path string, version int) (*SecretEntry, error) {
	kv := c.raw.KVv2(strings.TrimSuffix(engine, "/"))
	secret, err := kv.GetVersion(ctx, path, version)
	if err != nil {
		return nil, wrapError("get", fmt.Sprintf("%s%s@v%d", engine, path, version), err)
	}

	entry := &SecretEntry{
		Path: engine + path,
		Data: secret.Data,
	}
	if secret.VersionMetadata != nil {
		entry.Metadata = &SecretMetadata{
			Version:     secret.VersionMetadata.Version,
			CreatedTime: secret.VersionMetadata.CreatedTime,
		}
	}
	return entry, nil
}

func (c *Client) PutSecret(ctx context.Context, engine, path string, kvVersion int, data map[string]interface{}) error {
	if kvVersion == 2 {
		kv := c.raw.KVv2(strings.TrimSuffix(engine, "/"))
		_, err := kv.Put(ctx, path, data)
		if err != nil {
			return wrapError("put", engine+path, err)
		}
	} else {
		_, err := c.raw.Logical().WriteWithContext(ctx, engine+path, data)
		if err != nil {
			return wrapError("put", engine+path, err)
		}
	}
	return nil
}

func (c *Client) DeleteSecret(ctx context.Context, engine, path string, kvVersion int) error {
	if kvVersion == 2 {
		kv := c.raw.KVv2(strings.TrimSuffix(engine, "/"))
		err := kv.Delete(ctx, path)
		if err != nil {
			return wrapError("delete", engine+path, err)
		}
	} else {
		_, err := c.raw.Logical().DeleteWithContext(ctx, engine+path)
		if err != nil {
			return wrapError("delete", engine+path, err)
		}
	}
	return nil
}

func (c *Client) GetVersions(ctx context.Context, engine, path string) ([]VersionInfo, error) {
	kv := c.raw.KVv2(strings.TrimSuffix(engine, "/"))
	meta, err := kv.GetMetadata(ctx, path)
	if err != nil {
		return nil, wrapError("metadata", engine+path, err)
	}

	versions := make([]VersionInfo, 0, len(meta.Versions))
	for _, v := range meta.Versions {
		versions = append(versions, VersionInfo{
			Version:     v.Version,
			CreatedTime: v.CreatedTime,
			Deleted:     !v.DeletionTime.IsZero(),
			Destroyed:   v.Destroyed,
		})
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})

	return versions, nil
}
