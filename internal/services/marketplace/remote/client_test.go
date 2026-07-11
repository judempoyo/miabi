// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package remote

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCapTransportFailsClosed(t *testing.T) {
	const max = 1024
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), max*4)) // four times the budget
	}))
	defer srv.Close()

	hc := &http.Client{Transport: &capTransport{base: http.DefaultTransport, max: max}}
	resp, err := hc.Get(srv.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	_, err = io.ReadAll(resp.Body)
	if !errors.Is(err, errBundleTooLarge) {
		t.Fatalf("expected errBundleTooLarge, got %v", err)
	}
}

func TestCapTransportPassesUnderLimit(t *testing.T) {
	const max = 1024
	body := bytes.Repeat([]byte("y"), max/2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	hc := &http.Client{Transport: &capTransport{base: http.DefaultTransport, max: max}}
	resp, err := hc.Get(srv.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read under limit should succeed: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatal("body corrupted under the limit")
	}
}
