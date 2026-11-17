package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesystemPutGetRemove(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "media")
	store, err := NewFilesystem(root)
	if err != nil {
		t.Fatalf("NewFilesystem() error = %v", err)
	}
	if err := store.Ensure(ctx); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	content := []byte("imagem de teste")
	key := "media/pessoa/arquivo.jpg"
	if err := store.Put(ctx, key, bytes.NewReader(content), int64(len(content)), "image/jpeg"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, key)); err != nil {
		t.Fatalf("stored object not found: %v", err)
	}

	reader, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	got, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		t.Fatalf("read object error = %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("object content = %q, want %q", got, content)
	}

	if err := store.Remove(ctx, key); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := store.Get(ctx, key); err == nil {
		t.Error("Get() after Remove() error = nil, want error")
	}
	if err := store.Remove(ctx, key); err != nil {
		t.Errorf("Remove() missing object error = %v, want nil", err)
	}
}

func TestFilesystemRejectsTraversalKeys(t *testing.T) {
	ctx := context.Background()
	store, err := NewFilesystem(t.TempDir())
	if err != nil {
		t.Fatalf("NewFilesystem() error = %v", err)
	}

	for _, key := range []string{"", "../fora", "media/../../fora", ".."} {
		if err := store.Put(ctx, key, strings.NewReader("x"), 1, "text/plain"); err == nil {
			t.Errorf("Put(%q) error = nil, want invalid key error", key)
		}
		if _, err := store.Get(ctx, key); err == nil {
			t.Errorf("Get(%q) error = nil, want invalid key error", key)
		}
	}
}

func TestFilesystemPingMissingDirectory(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "ausente")
	store, err := NewFilesystem(root)
	if err != nil {
		t.Fatalf("NewFilesystem() error = %v", err)
	}
	if err := os.Remove(root); err != nil {
		t.Fatalf("remove storage directory: %v", err)
	}

	if err := store.Ping(ctx); err == nil {
		t.Error("Ping() error = nil, want error for missing directory")
	}
}
