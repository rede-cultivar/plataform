package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Filesystem grava os objetos em um diretório local.
//
// É uma opção para desenvolvimento e teste: não depende de um serviço externo
// e mantém as mídias entre reinícios da aplicação. Em produção, o backend
// compatível com S3 continua sendo o padrão.
type Filesystem struct {
	root string
}

// NewFilesystem cria o backend de arquivos, garantindo que o diretório exista.
func NewFilesystem(root string) (*Filesystem, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("storage path is required")
	}

	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve storage path %q: %w", root, err)
	}
	if err := os.MkdirAll(absolute, 0o755); err != nil {
		return nil, fmt.Errorf("create storage path %q: %w", absolute, err)
	}

	return &Filesystem{root: absolute}, nil
}

// Ensure garante que o diretório de armazenamento exista.
func (f *Filesystem) Ensure(ctx context.Context) error {
	return f.Ping(ctx)
}

// Ping verifica se o diretório de armazenamento está disponível.
func (f *Filesystem) Ping(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	info, err := os.Stat(f.root)
	if err != nil {
		return fmt.Errorf("stat storage path %q: %w", f.root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("storage path %q is not a directory", f.root)
	}

	return nil
}

// Put grava um objeto de forma atômica no diretório configurado.
func (f *Filesystem) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	path, err := f.resolve(key)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create object directory %q: %w", dir, err)
	}

	temp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return fmt.Errorf("create temporary object for %q: %w", key, err)
	}
	tempName := temp.Name()
	defer func() { _ = os.Remove(tempName) }()

	if _, err := io.Copy(temp, r); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write object %q: %w", key, err)
	}
	if err := temp.Chmod(0o644); err != nil {
		_ = temp.Close()
		return fmt.Errorf("set permissions on object %q: %w", key, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close object %q: %w", key, err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("store object %q: %w", key, err)
	}

	return nil
}

// Get abre um objeto gravado no diretório configurado.
func (f *Filesystem) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	path, err := f.resolve(key)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open object %q: %w", key, err)
	}

	return file, nil
}

// Remove apaga um objeto. A operação é idempotente, como no backend S3.
func (f *Filesystem) Remove(ctx context.Context, key string) error {
	path, err := f.resolve(key)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove object %q: %w", key, err)
	}

	return nil
}

// Location retorna o diretório configurado para os objetos.
func (f *Filesystem) Location() string {
	return f.root
}

func (f *Filesystem) resolve(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("storage key is required")
	}

	clean := filepath.Clean(strings.TrimPrefix(key, "/"))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." ||
		strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid storage key %q", key)
	}

	path := filepath.Join(f.root, clean)
	relative, err := filepath.Rel(f.root, path)
	if err != nil || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid storage key %q", key)
	}

	return path, nil
}
