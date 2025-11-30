package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFileReturnsEmptyCache(t *testing.T) {
	dir := t.TempDir()
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.PathHashes) != 0 {
		t.Fatalf("expected empty cache, got %v", c.PathHashes)
	}
}

func TestLoadCorruptedCacheFallsBack(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".tfsuitcache"), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write corrupted: %v", err)
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.PathHashes) != 0 {
		t.Fatalf("expected empty cache after corruption, got %v", c.PathHashes)
	}
}

func TestSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	cache := &Cache{PathHashes: map[string]string{"a.tf": "hash"}}
	if err := cache.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}
	res, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := res.PathHashes["a.tf"]; got != "hash" {
		t.Fatalf("expected hash, got %s", got)
	}
}

func TestHashDeterministic(t *testing.T) {
	h1 := Hash([]byte("hello"))
	h2 := Hash([]byte("hello"))
	if h1 != h2 || len(h1) != 64 {
		t.Fatalf("hash should be deterministic hex string")
	}
	if Hash([]byte("world")) == h1 {
		t.Fatalf("hash should vary with input")
	}
}
