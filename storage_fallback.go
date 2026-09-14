package onlyoffice

// MinIO download fallback for the portal's stale AWS S3 consumer.
//
// On the Fibu EDL portal some older Documents files live in S3/MinIO, but the
// portal's storage consumer still points at s3.us-east-1.amazonaws.com with
// access key "minio". Downloads of those files answer 403 InvalidAccessKeyId.
// The bytes are present in the local MinIO store under a deterministic object
// key, so the client retries the GET there.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

const (
	defaultMinioEndpoint = "http://192.168.188.10:9000"
	defaultMinioBucket   = "office"
	minioRegion          = "us-east-1"
)

// minioObjectKey is the fallback object key layout the portal's S3 consumer
// writes for Documents files: 00/00/01/files/folder_<folderId>/file_<fileId>/v1/content.pdf.
// Prefer minioObjectKeyFromURL: the portal stores all files below its storage
// root folder, which is not the API folderId returned by GetFile.
func minioObjectKey(fileID, folderID string) string {
	return "00/00/01/files/folder_" + folderID + "/file_" + fileID + "/v1/content.pdf"
}

// minioObjectKeyFromURL extracts the object key from an S3 download URL. Path
// style URLs (bucket as first path segment) have that segment removed; virtual
// host style URLs are returned as-is. This is authoritative: the portal signs
// the exact key, so no folder-id guessing is needed.
func minioObjectKeyFromURL(rawURL, bucket string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Path == "" {
		return "", false
	}
	segs := strings.Split(strings.Trim(u.Path, "/"), "/")
	// Path-style URLs carry the bucket as leading segment; the portal's S3
	// consumer can emit it twice (serviceurl already includes the bucket), so
	// strip every leading segment equal to the bucket.
	for len(segs) > 0 && bucket != "" && segs[0] == bucket {
		segs = segs[1:]
	}
	if len(segs) == 0 {
		return "", false
	}
	for _, s := range segs {
		if s == "" || s == "." || s == ".." {
			return "", false
		}
	}
	return strings.Join(segs, "/"), true
}

// isStaleS3Redirect reports whether a download landed on the portal's stale AWS
// S3 consumer. Such responses either carry an S3 InvalidAccessKeyId XML body or
// point at amazonaws.com with the "minio" access key id in the query.
func isStaleS3Redirect(rawURL string, body []byte) bool {
	if strings.Contains(strings.ToLower(string(body)), "invalidaccesskeyid") {
		return true
	}
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Host)
	if !strings.Contains(host, "amazonaws.com") {
		return false
	}
	q := strings.ToLower(u.RawQuery)
	return strings.Contains(q, "accesskeyid=minio") || strings.Contains(q, "x-amz-credential=minio")
}

// minioConfig is the runtime configuration for the local MinIO fallback.
type minioConfig struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
}

// loadMinioConfig reads the fallback configuration from the environment.
// Secrets are never defaulted; without access/secret keys the fallback is off.
func loadMinioConfig() minioConfig {
	return minioConfig{
		Endpoint:  strings.TrimRight(firstNonEmpty(os.Getenv("MINIO_ENDPOINT"), defaultMinioEndpoint), "/"),
		Bucket:    firstNonEmpty(os.Getenv("MINIO_BUCKET"), defaultMinioBucket),
		AccessKey: os.Getenv("MINIO_ACCESS_KEY"),
		SecretKey: os.Getenv("MINIO_SECRET_KEY"),
	}
}

// downloadFileEntry downloads f's bytes to dst. It transparently falls back to
// the local MinIO store when the portal redirects the download to its stale AWS
// S3 consumer.
func (c *Client) downloadFileEntry(ctx context.Context, f *FileEntry, dst io.Writer) (int64, error) {
	if f == nil {
		return 0, fmt.Errorf("onlyoffice: download: nil file entry")
	}
	if f.ViewURL == nil || *f.ViewURL == "" {
		return 0, fmt.Errorf("onlyoffice: file has no viewUrl")
	}
	downloadURL := c.resolveAPIURL(*f.ViewURL)
	auth, err := c.authHeader()
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", auth)
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		finalURL := downloadURL
		if resp.Request != nil && resp.Request.URL != nil {
			finalURL = resp.Request.URL.String()
		}
		if isStaleS3Redirect(finalURL, b) {
			key, ok := minioObjectKeyFromURL(finalURL, loadMinioConfig().Bucket)
			if !ok {
				fileID := ""
				if f.ID != nil {
					fileID = f.ID.String()
				}
				key = minioObjectKey(fileID, FileFolderID(f))
			}
			n, merr := c.downloadFromMinio(ctx, key, dst)
			if merr == nil {
				return n, nil
			}
			return 0, fmt.Errorf("GET viewUrl: %d (stale S3) and minio fallback: %w", resp.StatusCode, merr)
		}
		return 0, fmt.Errorf("GET viewUrl: %d %s", resp.StatusCode, truncate(string(b), 400))
	}
	return io.Copy(dst, resp.Body)
}

// downloadFromMinio streams objectKey from the configured MinIO bucket.
func (c *Client) downloadFromMinio(ctx context.Context, objectKey string, dst io.Writer) (int64, error) {
	if objectKey == "" {
		return 0, fmt.Errorf("onlyoffice: minio fallback: empty object key")
	}
	cfg := loadMinioConfig()
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return 0, fmt.Errorf("onlyoffice: minio fallback: MINIO_ACCESS_KEY/MINIO_SECRET_KEY not set")
	}
	base, err := url.Parse(cfg.Endpoint)
	if err != nil || base.Host == "" {
		return 0, fmt.Errorf("onlyoffice: minio fallback: bad MINIO_ENDPOINT %q", cfg.Endpoint)
	}
	u := *base
	u.Path = "/" + cfg.Bucket + "/" + objectKey
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, err
	}
	if err := signMinioRequest(ctx, cfg, req); err != nil {
		return 0, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("onlyoffice: minio fallback: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return 0, fmt.Errorf("onlyoffice: minio fallback: %d %s", resp.StatusCode, truncate(string(b), 300))
	}
	return io.Copy(dst, resp.Body)
}

// signMinioRequest signs req with AWS Signature V4 for the S3 service.
func signMinioRequest(ctx context.Context, cfg minioConfig, req *http.Request) error {
	sum := sha256.Sum256(nil)
	creds := aws.Credentials{AccessKeyID: cfg.AccessKey, SecretAccessKey: cfg.SecretKey}
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), "s3", minioRegion, time.Now()); err != nil {
		return fmt.Errorf("onlyoffice: minio fallback: sign: %w", err)
	}
	return nil
}
