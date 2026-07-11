// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package gitops

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/miabi-io/miabi/internal/models"
)

func TestVerifyWebhook(t *testing.T) {
	s := &Service{}
	src := &models.GitSource{WebhookSecret: "topsecret"}
	body := []byte(`{"ref":"refs/heads/main"}`)

	mac := hmac.New(sha256.New, []byte(src.WebhookSecret))
	mac.Write(body)
	good := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !s.VerifyWebhook(src, good, body) {
		t.Error("valid GitHub HMAC signature rejected")
	}
	if !s.VerifyWebhook(src, "topsecret", body) {
		t.Error("valid GitLab token rejected")
	}
	if s.VerifyWebhook(src, "sha256=deadbeef", body) {
		t.Error("forged signature accepted")
	}
	if s.VerifyWebhook(src, "", body) {
		t.Error("empty signature accepted")
	}
}
