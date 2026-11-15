package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/rede-cultivar/plataform/internal/platform/config"
)

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

// EnsureBucket cria o bucket configurado caso ele ainda não exista.
func (c *Client) EnsureBucket(ctx context.Context) error {
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

// Bucket retorna o nome do bucket configurado.
func (c *Client) Bucket() string {
	return c.bucket
}
