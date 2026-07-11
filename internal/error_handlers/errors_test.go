// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package errorhandlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jkaninda/okapi"
	"github.com/miabi-io/miabi/internal/dto"
)

// invoke runs the CustomErrorHandler against a fresh context and decodes the
// resulting envelope.
func invoke(t *testing.T, code int, message string, err error) dto.ErrorInfo {
	t.Helper()
	o := okapi.New()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := okapi.NewContext(o, rec, req)
	if herr := CustomErrorHandler()(c, code, message, err); herr != nil {
		t.Fatalf("handler returned error: %v", herr)
	}
	var body dto.ErrorResponseBody
	if derr := json.Unmarshal(rec.Body.Bytes(), &body); derr != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), derr)
	}
	if body.Success {
		t.Fatalf("expected success=false, got true")
	}
	if body.Error == nil {
		t.Fatalf("expected error info, got nil")
	}
	return *body.Error
}

func TestErrorHandler_PromotesErrWhenMessageGeneric(t *testing.T) {
	// AbortWithError(code, err) passes http.StatusText(code) as message; the real
	// reason (in err) must surface as the human message.
	const reason = "this is the Miabi control-plane/agent container and cannot be modified from the containers list"
	got := invoke(t, http.StatusConflict, http.StatusText(http.StatusConflict), errors.New(reason))
	if got.StatusCode != http.StatusConflict {
		t.Errorf("status_code = %d, want %d", got.StatusCode, http.StatusConflict)
	}
	if got.Code != "CONFLICT" {
		t.Errorf("code = %q, want CONFLICT", got.Code)
	}
	if got.Message != reason {
		t.Errorf("message = %q, want the reason", got.Message)
	}
	if got.Error != reason {
		t.Errorf("error = %q, want the reason", got.Error)
	}
}

func TestErrorHandler_KeepsSpecificMessage(t *testing.T) {
	// Structured helpers (AbortBadRequest("...")) pass a specific message that must
	// be preserved, with the raw err kept as supplementary detail.
	got := invoke(t, http.StatusInternalServerError, "failed to list containers", errors.New("dial unix /var/run/docker.sock: connect: no such file"))
	if got.Message != "failed to list containers" {
		t.Errorf("message = %q, want the specific message", got.Message)
	}
	if got.Error == "" || got.Error == got.Message {
		t.Errorf("error = %q, want the underlying detail", got.Error)
	}
}

// codedMsgErr is a domain error exposing both a stable Code() and a curated
// user-facing Message() distinct from its raw Error() string.
type codedMsgErr struct{}

func (codedMsgErr) Error() string {
	return "pq: duplicate key value violates unique constraint \"uniq_slug\""
}
func (codedMsgErr) Code() string    { return "CONFLICT" }
func (codedMsgErr) Message() string { return "a workspace with that slug already exists" }

func TestErrorHandler_PrefersDomainMessage(t *testing.T) {
	// When the err exposes Message(), it becomes the human message while the raw
	// Error() string stays in `error`; Code() drives the machine code.
	got := invoke(t, http.StatusConflict, http.StatusText(http.StatusConflict), codedMsgErr{})
	if got.Code != "CONFLICT" {
		t.Errorf("code = %q, want CONFLICT", got.Code)
	}
	if got.Message != "a workspace with that slug already exists" {
		t.Errorf("message = %q, want the curated message", got.Message)
	}
	if got.Error == "" || got.Error == got.Message {
		t.Errorf("error = %q, want the raw Error() detail", got.Error)
	}
}

func TestErrorHandler_NoErrFallsBackToStatusText(t *testing.T) {
	got := invoke(t, http.StatusNotFound, "", nil)
	if got.Message != http.StatusText(http.StatusNotFound) {
		t.Errorf("message = %q, want %q", got.Message, http.StatusText(http.StatusNotFound))
	}
	if got.Code != "NOT_FOUND" {
		t.Errorf("code = %q, want NOT_FOUND", got.Code)
	}
}
