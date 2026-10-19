package config

import (
	"fmt"
	"strconv"
	"strings"

	"safedrop/internal/storage"
	"safedrop/internal/validate"
)

type Upload struct {
	DatabaseURL           string
	Backend, LocalRoot    string
	S3                    storage.S3Config
	Limits                validate.Limits
	StubUserID, StubToken string
	SharePresign          bool
	JWTSecret             string
	Abuse                 Abuse
}

func loadUpload(lookup func(string) (string, bool)) (Upload, error) {
	value := func(key, fallback string) string {
		if v, ok := lookup(key); ok {
			return strings.TrimSpace(v)
		}
		return fallback
	}
	u := Upload{
		JWTSecret:   value("JWT_SECRET", ""),
		DatabaseURL: value("DATABASE_URL", ""), Backend: value("STORAGE_BACKEND", "local"),
		LocalRoot:  value("STORAGE_LOCAL_ROOT", "./data"),
		StubUserID: value("STUB_AUTH_USER_ID", ""), StubToken: value("STUB_AUTH_TOKEN", ""),
		S3: storage.S3Config{Endpoint: value("STORAGE_ENDPOINT", ""), Region: value("STORAGE_REGION", "us-east-1"),
			Bucket: value("STORAGE_BUCKET", "safedrop"), AccessKeyID: value("STORAGE_ACCESS_KEY_ID", ""),
			SecretAccessKey: value("STORAGE_SECRET_ACCESS_KEY", ""), SessionToken: value("STORAGE_SESSION_TOKEN", "")},
	}
	if u.Backend != "local" && u.Backend != "s3" {
		return u, fmt.Errorf("STORAGE_BACKEND must be local or s3")
	}
	if (u.StubUserID == "") != (u.StubToken == "") {
		return u, fmt.Errorf("both stub authentication settings are required")
	}
	var err error
	u.S3.ForcePathStyle, err = strconv.ParseBool(value("STORAGE_FORCE_PATH_STYLE", "true"))
	if err != nil {
		return u, fmt.Errorf("invalid STORAGE_FORCE_PATH_STYLE")
	}
	u.Limits.MinBytes, err = strconv.ParseInt(value("MIN_UPLOAD_BYTES", "1"), 10, 64)
	if err != nil || u.Limits.MinBytes < 1 {
		return u, fmt.Errorf("MIN_UPLOAD_BYTES must be positive")
	}
	u.Limits.MaxBytes, err = strconv.ParseInt(value("MAX_UPLOAD_BYTES", "26214400"), 10, 64)
	if err != nil || u.Limits.MaxBytes < u.Limits.MinBytes {
		return u, fmt.Errorf("MAX_UPLOAD_BYTES must be at least MIN_UPLOAD_BYTES")
	}
	u.Limits.MaxPixels, err = strconv.ParseUint(value("MAX_IMAGE_PIXELS", "20000000"), 10, 64)
	if err != nil || u.Limits.MaxPixels == 0 {
		return u, fmt.Errorf("MAX_IMAGE_PIXELS must be positive")
	}
	u.Limits.TempDir = value("UPLOAD_TEMP_DIR", "")
	u.SharePresign, err = strconv.ParseBool(value("SHARE_PRESIGN_GET", "false"))
	if err != nil {
		return u, fmt.Errorf("invalid SHARE_PRESIGN_GET")
	}
	u.Abuse, err = loadAbuse(lookup)
	return u, err
}
