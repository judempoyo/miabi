// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/miabi-io/miabi/internal/models"
)

func TestParsePushGitHub(t *testing.T) {
	body := []byte(`{"ref":"refs/heads/main","after":"abc123","head_commit":{"id":"abc123","message":"fix: bug"}}`)
	branch, commit, msg := parsePush(body)
	if branch != "main" || commit != "abc123" || msg != "fix: bug" {
		t.Fatalf("parsePush github = (%q,%q,%q)", branch, commit, msg)
	}
}

func TestParsePushGitLab(t *testing.T) {
	body := []byte(`{"ref":"refs/heads/dev","checkout_sha":"def456"}`)
	branch, commit, _ := parsePush(body)
	if branch != "dev" || commit != "def456" {
		t.Fatalf("parsePush gitlab = (%q,%q)", branch, commit)
	}
}

func TestParsePushGarbage(t *testing.T) {
	branch, commit, _ := parsePush([]byte("not json"))
	if branch != "" || commit != "" {
		t.Fatalf("garbage payload should yield empty, got (%q,%q)", branch, commit)
	}
}

func TestVerifyWebhook(t *testing.T) {
	s := &Service{}
	p := &models.PipelineDefinition{WebhookSecret: "topsecret"}
	body := []byte(`{"ref":"refs/heads/main"}`)

	mac := hmac.New(sha256.New, []byte(p.WebhookSecret))
	mac.Write(body)
	good := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !s.VerifyWebhook(p, good, body) {
		t.Error("valid HMAC signature rejected")
	}
	if !s.VerifyWebhook(p, "topsecret", body) {
		t.Error("valid GitLab token rejected")
	}
	if s.VerifyWebhook(p, "sha256=deadbeef", body) {
		t.Error("forged signature accepted")
	}
	if s.VerifyWebhook(p, "", body) {
		t.Error("empty signature accepted")
	}
	if s.VerifyWebhook(&models.PipelineDefinition{}, good, body) {
		t.Error("pipeline with no secret accepted a signature")
	}
}
