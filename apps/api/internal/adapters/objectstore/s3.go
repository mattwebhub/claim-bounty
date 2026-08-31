package objectstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3 struct {
	client *minio.Client
	bucket string
}

func NewS3(ctx context.Context, endpoint, region, bucket, accessKey, secretKey string, secure, createBucket bool) (*S3, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || parsed.Path != "" {
		return nil, errors.New("objectstore: endpoint must contain only scheme and host")
	}
	client, err := minio.New(parsed.Host, &minio.Options{Creds: credentials.NewStaticV4(accessKey, secretKey, ""), Secure: secure, Region: region})
	if err != nil {
		return nil, errors.New("objectstore: client initialization failed")
	}
	store := &S3{client: client, bucket: bucket}
	if createBucket {
		exists, err := client.BucketExists(ctx, bucket)
		if err != nil {
			return nil, errors.New("objectstore: bucket check failed")
		}
		if !exists {
			if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: region}); err != nil {
				return nil, errors.New("objectstore: private bucket creation failed")
			}
		}
		if err := client.EnableVersioning(ctx, bucket); err != nil {
			return nil, errors.New("objectstore: bucket versioning is required")
		}
	}
	return store, nil
}

func (store *S3) PutWriteOnce(ctx context.Context, key string, reader io.Reader, size int64, mediaType, expectedSHA string) (ports.ObjectMetadata, error) {
	if !safeKey(key) || size < 1 {
		return ports.ObjectMetadata{}, errors.New("objectstore: invalid write request")
	}
	if existing, err := store.client.StatObject(ctx, store.bucket, key, minio.StatObjectOptions{}); err == nil {
		metadata := metadataFromStat(existing)
		if metadata.SizeBytes == size && metadata.SHA256 == expectedSHA && metadata.Generation != "" {
			return metadata, nil
		}
		return ports.ObjectMetadata{}, errors.New("objectstore: write-once key already exists with different content")
	}
	hash := sha256.New()
	limited := &io.LimitedReader{R: reader, N: size + 1}
	stream := io.TeeReader(limited, hash)
	options := minio.PutObjectOptions{ContentType: mediaType, UserMetadata: map[string]string{"sha256": expectedSHA}, DisableMultipart: true}
	options.SetMatchETagExcept("*")
	info, err := store.client.PutObject(ctx, store.bucket, key, stream, size, options)
	if err != nil {
		if existing, statErr := store.client.StatObject(ctx, store.bucket, key, minio.StatObjectOptions{}); statErr == nil {
			metadata := metadataFromStat(existing)
			if metadata.SizeBytes == size && metadata.SHA256 == expectedSHA && metadata.Generation != "" {
				return metadata, nil
			}
		}
		return ports.ObjectMetadata{}, errors.New("objectstore: upload failed")
	}
	extra := make([]byte, 1)
	n, readErr := stream.Read(extra)
	actualSHA := hex.EncodeToString(hash.Sum(nil))
	if (readErr != nil && !errors.Is(readErr, io.EOF)) || n != 0 || actualSHA != expectedSHA || info.Size != size || info.VersionID == "" {
		_ = store.client.RemoveObject(ctx, store.bucket, key, minio.RemoveObjectOptions{VersionID: info.VersionID, ForceDelete: true})
		return ports.ObjectMetadata{}, domain.NewValidationError(domain.FieldIssue{Field: "file", Code: "mismatch", Message: "uploaded bytes do not match size and SHA-256"})
	}
	return ports.ObjectMetadata{SizeBytes: info.Size, ETag: strings.Trim(info.ETag, `"`), MediaType: mediaType, SHA256: actualSHA, Generation: info.VersionID, ModifiedAt: time.Now().UTC()}, nil
}

func (store *S3) Open(ctx context.Context, key, generation string) (ports.ObjectReader, ports.ObjectMetadata, error) {
	if !safeKey(key) || generation == "" {
		return nil, ports.ObjectMetadata{}, errors.New("objectstore: invalid read request")
	}
	options := minio.GetObjectOptions{VersionID: generation}
	stat, err := store.client.StatObject(ctx, store.bucket, key, minio.StatObjectOptions{VersionID: generation})
	if err != nil {
		return nil, ports.ObjectMetadata{}, errors.New("objectstore: private object not found")
	}
	object, err := store.client.GetObject(ctx, store.bucket, key, options)
	if err != nil {
		return nil, ports.ObjectMetadata{}, errors.New("objectstore: private object open failed")
	}
	return object, metadataFromStat(stat), nil
}

func (store *S3) DeleteVersion(ctx context.Context, key, generation string) error {
	if !safeKey(key) || generation == "" {
		return errors.New("objectstore: invalid delete request")
	}
	if err := store.client.RemoveObject(ctx, store.bucket, key, minio.RemoveObjectOptions{VersionID: generation, ForceDelete: true}); err != nil {
		return errors.New("objectstore: private object deletion failed")
	}
	return nil
}

func safeKey(key string) bool {
	return (strings.HasPrefix(key, "quarantine/") || strings.HasPrefix(key, "accepted/sha256/") || strings.HasPrefix(key, "exports/")) && !strings.Contains(key, "..") && !strings.ContainsAny(key, "\\\x00")
}

type Scope struct {
	store          ports.PrivateObjectStore
	readPrefixes   []string
	writePrefixes  []string
	deletePrefixes []string
}

func NewScope(store ports.PrivateObjectStore, readPrefixes, writePrefixes, deletePrefixes []string) (*Scope, error) {
	if store == nil || !validPrefixes(readPrefixes) || !validPrefixes(writePrefixes) || !validPrefixes(deletePrefixes) {
		return nil, errors.New("objectstore: invalid storage scope")
	}
	return &Scope{store: store, readPrefixes: readPrefixes, writePrefixes: writePrefixes, deletePrefixes: deletePrefixes}, nil
}

func validPrefixes(prefixes []string) bool {
	for _, prefix := range prefixes {
		if prefix != "quarantine/" && prefix != "accepted/sha256/" && prefix != "exports/" {
			return false
		}
	}
	return true
}

func allowedPrefix(key string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func (scope *Scope) Open(ctx context.Context, key, generation string) (ports.ObjectReader, ports.ObjectMetadata, error) {
	if !allowedPrefix(key, scope.readPrefixes) {
		return nil, ports.ObjectMetadata{}, errors.New("objectstore: read denied by process scope")
	}
	return scope.store.Open(ctx, key, generation)
}

func (scope *Scope) PutWriteOnce(ctx context.Context, key string, reader io.Reader, size int64, mediaType, expectedSHA string) (ports.ObjectMetadata, error) {
	if !allowedPrefix(key, scope.writePrefixes) {
		return ports.ObjectMetadata{}, errors.New("objectstore: write denied by process scope")
	}
	return scope.store.PutWriteOnce(ctx, key, reader, size, mediaType, expectedSHA)
}

func (scope *Scope) DeleteVersion(ctx context.Context, key, generation string) error {
	if !allowedPrefix(key, scope.deletePrefixes) {
		return errors.New("objectstore: delete denied by process scope")
	}
	return scope.store.DeleteVersion(ctx, key, generation)
}

func metadataValue(metadata map[string]string, name string) string {
	for key, value := range metadata {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}

func metadataFromStat(stat minio.ObjectInfo) ports.ObjectMetadata {
	return ports.ObjectMetadata{SizeBytes: stat.Size, ETag: strings.Trim(stat.ETag, `"`), MediaType: stat.ContentType, SHA256: metadataValue(stat.UserMetadata, "X-Amz-Meta-Sha256"), Generation: stat.VersionID, ModifiedAt: stat.LastModified.UTC()}
}
