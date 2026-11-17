package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/rede-cultivar/plataform/internal/platform/config"
)

// Backend reúne as operações usadas pela aplicação para armazenar objetos.
type Backend interface {
	Ensure(ctx context.Context) error
	Ping(ctx context.Context) error
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Remove(ctx context.Context, key string) error
	Location() string
}

// NewBackend cria o backend de armazenamento configurado.
func NewBackend(cfg *config.Config) (Backend, error) {
	switch cfg.Storage.Driver {
	case config.StorageDriverFilesystem:
		return NewFilesystem(cfg.Storage.Path)
	case config.StorageDriverS3, "":
		return New(cfg.S3)
	default:
		return nil, fmt.Errorf("unsupported storage driver %q", cfg.Storage.Driver)
	}
}

// Client encapsula o acesso ao armazenamento de objetos compatível com S3.
type Client struct {
	mc     *minio.Client
	bucket string
	region string
}

// New cria um cliente S3 a partir da configuração da aplicação.
func New(cfg config.S3) (*Client, error) {
	endpoint := strings.TrimPrefix(strings.TrimPrefix(cfg.Endpoint, "http://"), "https://")
	if endpoint == "" {
		return nil, fmt.Errorf("s3 endpoint is required")
	}

	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}

	return &Client{
		mc:     mc,
		bucket: cfg.Bucket,
		region: cfg.Region,
	}, nil
}

// Ensure cria o bucket configurado caso ele ainda não exista.
func (c *Client) Ensure(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("check bucket %q: %w", c.bucket, err)
	}
	if exists {
		return nil
	}

	if err := c.mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{Region: c.region}); err != nil {
		return fmt.Errorf("create bucket %q: %w", c.bucket, err)
	}

	return nil
}

// Ping verifica a disponibilidade do bucket configurado.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("ping bucket %q: %w", c.bucket, err)
	}
	return nil
}

// Put grava um objeto no bucket configurado.
func (c *Client) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := c.mc.PutObject(ctx, c.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}
	return nil
}

// Get abre um objeto do bucket configurado.
func (c *Client) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	object, err := c.mc.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object %q: %w", key, err)
	}

	if _, err := object.Stat(); err != nil {
		_ = object.Close()
		return nil, fmt.Errorf("stat object %q: %w", key, err)
	}

	return object, nil
}

// Remove apaga um objeto do bucket configurado.
func (c *Client) Remove(ctx context.Context, key string) error {
	if err := c.mc.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove object %q: %w", key, err)
	}
	return nil
}

// Location retorna o nome do bucket configurado.
func (c *Client) Location() string {
	return c.bucket
}
