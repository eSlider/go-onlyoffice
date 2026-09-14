package onlyoffice

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMinioObjectKey(t *testing.T) {
	cases := []struct {
		fileID   string
		folderID string
		want     string
	}{
		{"3785", "652", "00/00/01/files/folder_652/file_3785/v1/content.pdf"},
		{"1", "2", "00/00/01/files/folder_2/file_1/v1/content.pdf"},
		{"3666", "4000", "00/00/01/files/folder_4000/file_3666/v1/content.pdf"},
	}
	for _, tc := range cases {
		if got := minioObjectKey(tc.fileID, tc.folderID); got != tc.want {
			t.Errorf("minioObjectKey(%q, %q) = %q, want %q", tc.fileID, tc.folderID, got, tc.want)
		}
	}
}

func TestMinioObjectKeyFromURL(t *testing.T) {
	cases := []struct {
		name   string
		url    string
		bucket string
		want   string
		ok     bool
	}{
		{
			name:   "path style drops bucket segment",
			url:    "https://s3.us-east-1.amazonaws.com/office/00/00/01/files/folder_4000/file_3785/v1/content.pdf?AWSAccessKeyId=minio",
			bucket: "office",
			want:   "00/00/01/files/folder_4000/file_3785/v1/content.pdf",
			ok:     true,
		},
		{
			name:   "doubled bucket segment (portal serviceurl includes bucket)",
			url:    "https://s3.us-east-1.amazonaws.com/office/office/00/00/01/files/folder_4000/file_3785/v1/content.pdf?AWSAccessKeyId=minio",
			bucket: "office",
			want:   "00/00/01/files/folder_4000/file_3785/v1/content.pdf",
			ok:     true,
		},
		{
			name:   "virtual host style keeps path",
			url:    "https://office.s3.us-east-1.amazonaws.com/00/00/01/files/folder_4000/file_3785/v1/content.pdf",
			bucket: "office",
			want:   "00/00/01/files/folder_4000/file_3785/v1/content.pdf",
			ok:     true,
		},
		{
			name:   "foreign first segment kept",
			url:    "https://example.com/other/file_1/v1/content.pdf",
			bucket: "office",
			want:   "other/file_1/v1/content.pdf",
			ok:     true,
		},
		{name: "empty path", url: "https://example.com", bucket: "office", ok: false},
		{name: "traversal", url: "https://example.com/office/../etc/passwd", bucket: "office", ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := minioObjectKeyFromURL(tc.url, tc.bucket)
			if ok != tc.ok || got != tc.want {
				t.Errorf("minioObjectKeyFromURL(%q, %q) = (%q, %v), want (%q, %v)", tc.url, tc.bucket, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestIsStaleS3Redirect(t *testing.T) {
	cases := []struct {
		name string
		url  string
		body []byte
		want bool
	}{
		{
			name: "aws redirect with minio access key",
			url:  "https://s3.us-east-1.amazonaws.com/office/x/file_1?AWSAccessKeyId=minio&Expires=1",
			want: true,
		},
		{
			name: "aws redirect with minio x-amz-credential",
			url:  "https://office.s3.us-east-1.amazonaws.com/00/00/01/files/folder_1/file_1?X-Amz-Credential=minio%2F20260914",
			want: true,
		},
		{
			name: "invalid access key xml body",
			url:  "https://portal.internal/download/1",
			body: []byte(`<?xml version="1.0"?><Error><Code>InvalidAccessKeyId</Code><AWSAccessKeyId>minio</AWSAccessKeyId></Error>`),
			want: true,
		},
		{
			name: "regular pdf from portal",
			url:  "https://portal.internal/download/1",
			body: []byte("%PDF-1.7 data"),
			want: false,
		},
		{
			name: "aws redirect with foreign key",
			url:  "https://s3.us-east-1.amazonaws.com/office/x?AWSAccessKeyId=other",
			want: false,
		},
		{
			name: "amazonaws in path but foreign host",
			url:  "https://example.com/amazonaws.com/file?AWSAccessKeyId=minio",
			want: false,
		},
		{
			name: "empty",
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isStaleS3Redirect(tc.url, tc.body); got != tc.want {
				t.Errorf("isStaleS3Redirect(%q, %q) = %v, want %v", tc.url, tc.body, got, tc.want)
			}
		})
	}
}

const staleS3Body = `<?xml version="1.0" encoding="UTF-8"?>` +
	`<Error><Code>InvalidAccessKeyId</Code>` +
	`<Message>The AWS Access Key Id you provided does not exist in our records.</Message>` +
	`<AWSAccessKeyId>minio</AWSAccessKeyId></Error>`

func TestDownloadFileMinioFallback(t *testing.T) {
	const payload = "PDFDATA-3785"

	portal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/files/file/3785.json":
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"response":{"id":3785,"title":"04.pdf","folderId":655,"viewUrl":"/download/3785"}}`)
		case "/download/3785":
			http.Redirect(w, r, "/office/00/00/01/files/folder_4000/file_3785/v1/content.pdf?AWSAccessKeyId=minio", http.StatusTemporaryRedirect)
		case "/office/00/00/01/files/folder_4000/file_3785/v1/content.pdf":
			w.WriteHeader(http.StatusForbidden)
			io.WriteString(w, staleS3Body)
		default:
			http.NotFound(w, r)
		}
	}))
	defer portal.Close()

	var minioPath, minioAuth string
	minio := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		minioPath, minioAuth = r.URL.Path, r.Header.Get("Authorization")
		io.WriteString(w, payload)
	}))
	defer minio.Close()

	t.Setenv("MINIO_ENDPOINT", minio.URL)
	t.Setenv("MINIO_BUCKET", "office")
	t.Setenv("MINIO_ACCESS_KEY", "testkey")
	t.Setenv("MINIO_SECRET_KEY", "testsecret")

	c := &Client{
		client:      portal.Client(),
		credentials: &Credentials{Url: portal.URL},
		token:       &Token{Value: "Bearer test", Expires: Time(time.Now().Add(time.Hour))},
	}

	var buf bytes.Buffer
	n, err := c.DownloadFile(context.Background(), "3785", &buf)
	if err != nil {
		t.Fatalf("DownloadFile: %v", err)
	}
	if n != int64(len(payload)) || buf.String() != payload {
		t.Fatalf("got %d bytes %q, want %d bytes %q", n, buf.String(), len(payload), payload)
	}
	if want := "/office/00/00/01/files/folder_4000/file_3785/v1/content.pdf"; minioPath != want {
		t.Errorf("minio path = %q, want %q", minioPath, want)
	}
	if !strings.HasPrefix(minioAuth, "AWS4-HMAC-SHA256") {
		t.Errorf("minio request not SigV4-signed; Authorization=%q", minioAuth)
	}
}

func TestDownloadFileMinioFallbackWithoutCreds(t *testing.T) {
	portal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/files/file/3785.json":
			io.WriteString(w, `{"response":{"id":3785,"folderId":652,"viewUrl":"/download/3785"}}`)
		default:
			w.WriteHeader(http.StatusForbidden)
			io.WriteString(w, staleS3Body)
		}
	}))
	defer portal.Close()

	t.Setenv("MINIO_ACCESS_KEY", "")
	t.Setenv("MINIO_SECRET_KEY", "")

	c := &Client{
		client:      portal.Client(),
		credentials: &Credentials{Url: portal.URL},
		token:       &Token{Value: "Bearer test", Expires: Time(time.Now().Add(time.Hour))},
	}

	_, err := c.DownloadFile(context.Background(), "3785", io.Discard)
	if err == nil {
		t.Fatal("expected error without minio credentials")
	}
	if !strings.Contains(err.Error(), "MINIO_ACCESS_KEY/MINIO_SECRET_KEY not set") {
		t.Fatalf("unexpected error: %v", err)
	}
}
