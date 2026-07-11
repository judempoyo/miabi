// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"strings"
	"testing"

	"github.com/miabi-io/miabi/internal/models"
)

func TestFormatMessage(t *testing.T) {
	e := &models.AppEvent{
		ApplicationID: 12,
		Type:          models.EventDeployFailed,
		Message:       "image pull failed",
	}
	msg := formatMessage(e)
	if !strings.Contains(msg, "Deployment failed") {
		t.Errorf("missing title in %q", msg)
	}
	if !strings.Contains(msg, "#12") {
		t.Errorf("missing application id in %q", msg)
	}
	if !strings.Contains(msg, "image pull failed") {
		t.Errorf("missing message in %q", msg)
	}
}

func TestValidateEvents(t *testing.T) {
	if err := validateEvents([]string{string(models.EventContainerDied)}); err != nil {
		t.Errorf("notifiable event rejected: %v", err)
	}
	if err := validateEvents([]string{"bogus.event"}); err == nil {
		t.Error("unknown event should be rejected")
	}
}

func TestRegistrySupportedChannels(t *testing.T) {
	r := NewRegistry()
	for _, typ := range []models.NotificationChannelType{
		models.ChannelTelegram, models.ChannelSlack, models.ChannelDiscord,
	} {
		if _, ok := r.Sender(typ); !ok {
			t.Errorf("%s sender not registered", typ)
		}
	}
	if _, ok := r.Sender(models.NotificationChannelType("teams")); ok {
		t.Fatal("unexpected sender for unsupported type")
	}
}
