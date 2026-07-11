// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package backup

import (
	"strings"
	"testing"

	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/services/crypto"
)

// mustEncrypt stores a value the way the DB layer does; with no key configured in
// tests crypto round-trips via base64 so connEnv can decrypt it back.
func mustEncrypt(t *testing.T, v string) string {
	t.Helper()
	enc, err := crypto.Encrypt(v)
	if err != nil {
		t.Fatal(err)
	}
	return enc
}

func TestS3Env(t *testing.T) {
	env := s3Env(&S3Config{
		Endpoint: "http://minio:9000", Bucket: "backups", Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", UseSSL: false, ForcePathStyle: true,
	})
	want := map[string]string{
		"AWS_S3_ENDPOINT":      "http://minio:9000",
		"AWS_S3_BUCKET_NAME":   "backups",
		"AWS_REGION":           "us-east-1",
		"AWS_ACCESS_KEY":       "ak",
		"AWS_SECRET_KEY":       "sk",
		"AWS_DISABLE_SSL":      "true", // UseSSL=false => disabled
		"AWS_FORCE_PATH_STYLE": "true",
	}
	got := map[string]string{}
	for _, kv := range env {
		parts := strings.SplitN(kv, "=", 2)
		got[parts[0]] = parts[1]
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestBkupImage(t *testing.T) {
	s := &Service{} // nil resolver -> built-in defaults
	if img, ok := s.bkupImage("postgres"); !ok || !strings.Contains(img, "pg-bkup") {
		t.Errorf("postgres image = %q, ok=%v", img, ok)
	}
	if img, ok := s.bkupImage("mariadb"); !ok || !strings.Contains(img, "mysql-bkup") {
		t.Errorf("mariadb image = %q, ok=%v", img, ok)
	}
	if img, ok := s.bkupImage("mongodb"); !ok || !strings.Contains(img, "mongodb-bkup") {
		t.Errorf("mongodb image = %q, ok=%v", img, ok)
	}
	if img, ok := s.bkupImage("libsql"); !ok || !strings.Contains(img, "libsql-bkup") {
		t.Errorf("libsql image = %q, ok=%v", img, ok)
	}
	if _, ok := s.bkupImage("redis"); ok {
		t.Error("redis should be unsupported for backups")
	}
}

func TestConnEnvLibSQLUsesToken(t *testing.T) {
	s := &Service{}
	inst := &models.DatabaseInstance{Engine: models.DBEngineLibSQL, Host: "mb-db-1", Port: 8080}
	db := &models.Database{Name: "default", Username: "libsql", PasswordEnc: mustEncrypt(t, "tok123")}
	env, err := s.connEnv(inst, db)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(env, " ")
	for _, want := range []string{"DB_URL=http://mb-db-1:8080", "LIBSQL_AUTH_TOKEN=tok123", "DB_NAME=default"} {
		if !strings.Contains(joined, want) {
			t.Errorf("env missing %q: %v", want, env)
		}
	}
	if strings.Contains(joined, "DB_PASSWORD=") || strings.Contains(joined, "DB_USERNAME=") {
		t.Errorf("libsql connEnv must not emit user/password: %v", env)
	}
}

func TestArtifactRegexMatchesMongoArchive(t *testing.T) {
	for _, name := range []string{"shop_20260101_120000.sql.gz", "shop_20260101_120000.archive.gz"} {
		if got := artifactRe.FindString("backup written to " + name); got != name {
			t.Errorf("artifactRe on %q = %q, want %q", name, got, name)
		}
	}
}
