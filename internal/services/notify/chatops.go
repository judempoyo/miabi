// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/netguard"
	"github.com/miabi-io/miabi/internal/services/crypto"
)

// webhookChannelSender delivers notifications to an incoming-webhook style
// endpoint (Slack, Discord). The two differ only in the JSON field carrying the
// message text. The webhook URL is a secret, stored encrypted.
type webhookChannelSender struct {
	field  string // JSON body field for the message ("text" for Slack, "content" for Discord)
	label  string // human label for errors
	client *http.Client
}

// NewSlackSender posts to a Slack incoming webhook ({"text": ...}).
func NewSlackSender() *webhookChannelSender {
	return &webhookChannelSender{field: "text", label: "Slack", client: netguard.Client(10 * time.Second)}
}

// NewDiscordSender posts to a Discord webhook ({"content": ...}).
func NewDiscordSender() *webhookChannelSender {
	return &webhookChannelSender{field: "content", label: "Discord", client: netguard.Client(10 * time.Second)}
}

func (s *webhookChannelSender) Send(ctx context.Context, ch *models.NotificationChannel, e *models.AppEvent) error {
	url, err := s.url(ch)
	if err != nil {
		return err
	}
	return s.post(ctx, url, formatMessage(e))
}

func (s *webhookChannelSender) Validate(ctx context.Context, ch *models.NotificationChannel) error {
	url, err := s.url(ch)
	if err != nil {
		return err
	}
	return s.post(ctx, url, "✅ Miabi test notification — this channel is configured correctly.")
}

func (s *webhookChannelSender) url(ch *models.NotificationChannel) (string, error) {
	enc := strings.TrimSpace(ch.Config[models.ConfigWebhookURL])
	if enc == "" {
		return "", fmt.Errorf("%s webhook url is not configured", s.label)
	}
	url, err := crypto.Decrypt(enc)
	if err != nil {
		return "", fmt.Errorf("decrypt webhook url: %w", err)
	}
	return url, nil
}

func (s *webhookChannelSender) post(ctx context.Context, url, text string) error {
	body, err := json.Marshal(map[string]string{s.field: text})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s request: %w", s.label, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return fmt.Errorf("%s returned %d: %s", s.label, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	return nil
}
