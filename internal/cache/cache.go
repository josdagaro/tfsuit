package cache

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "io/ioutil"
    "os"
    "path/filepath"
)

type Cache struct {
    PathHashes map[string]string `json:"path_hashes"`
}

// Load reads .tfsuitcache (JSON) if present; otherwise returns empty cache.
func Load(root string) (*Cache, error) {
    c := &Cache{PathHashes: make(map[string]string)}
    path := filepath.Join(root, ".tfsuitcache")
    data, err := ioutil.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return c, nil
        }
        return nil, err
    }
    if err := json.Unmarshal(data, c); err != nil {
        // Ignore corrupted cache
        return &Cache{PathHashes: make(map[string]string)}, nil
    }
    return c, nil
}

// Save writes the cache back to disk.
func (c *Cache) Save(root string) error {
    data, err := json.MarshalIndent(c, "", "  ")
    if err != nil {
        return err
    }
    return ioutil.WriteFile(filepath.Join(root, ".tfsuitcache"), data, 0o644)
}

// Hash computes SHA-256 of content.
func Hash(b []byte) string {
    h := sha256.Sum256(b)
    return hex.EncodeToString(h[:])
}
