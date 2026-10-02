package storage

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"

	"gin-micro-ihome/web/conf"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	clientOnce sync.Once
	client     *minio.Client
	clientErr  error
	bucketOnce sync.Once
	bucketErr  error
)

func getClient() (*minio.Client, error) {
	clientOnce.Do(func() {
		client, clientErr = minio.New(conf.MinioEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(conf.MinioAccessKey, conf.MinioSecretKey, ""),
			Secure: conf.MinioUseSSL,
		})
	})
	return client, clientErr
}

func ensureBucket(ctx context.Context) error {
	bucketOnce.Do(func() {
		c, err := getClient()
		if err != nil {
			bucketErr = err
			return
		}
		exists, err := c.BucketExists(ctx, conf.MinioBucket)
		if err != nil {
			bucketErr = err
			return
		}
		if !exists {
			bucketErr = c.MakeBucket(ctx, conf.MinioBucket, minio.MakeBucketOptions{})
		}
	})
	return bucketErr
}

// Upload 上传对象，返回可公网访问的 URL
func Upload(ctx context.Context, objectKey string, data []byte, contentType string) (string, error) {
	if err := ensureBucket(ctx); err != nil {
		return "", err
	}
	c, err := getClient()
	if err != nil {
		return "", err
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	objectKey = strings.TrimPrefix(objectKey, "/")
	_, err = c.PutObject(ctx, conf.MinioBucket, objectKey, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(conf.MinioPublicBase, "/"), conf.MinioBucket, objectKey), nil
}

// ContentTypeByExt 根据扩展名推断 Content-Type（ext 可带点）
func ContentTypeByExt(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	switch ext {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
