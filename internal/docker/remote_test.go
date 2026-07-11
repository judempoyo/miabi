// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package docker

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestNewRemoteOverTransport proves a remote Docker client built on an arbitrary
// transport (here a TCP dial; in production the agent tunnel) reaches the engine.
// A stub server emulates the Docker Engine API the SDK calls for Info().
func TestNewRemoteOverTransport(t *testing.T) {
	mux := http.NewServeMux()
	ping := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Api-Version", "1.43")
		w.Header().Set("Ostype", "linux")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}
	mux.HandleFunc("/_ping", ping)
	// Match any version-prefixed path; serve /info and /_ping.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/_ping"):
			ping(w, r)
		case strings.HasSuffix(r.URL.Path, "/info"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ServerVersion":"27.5.1","OperatingSystem":"Test OS","Architecture":"x86_64","NCPU":4,"MemTotal":2048,"Containers":3,"ContainersRunning":2,"Images":5}`))
		default:
			http.NotFound(w, r)
		}
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "http://")

	cl, err := NewRemote(func(ctx context.Context) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "tcp", addr)
	})
	if err != nil {
		t.Fatalf("NewRemote: %v", err)
	}
	defer cl.Close()

	info, err := cl.Info(context.Background())
	if err != nil {
		t.Fatalf("Info over remote transport: %v", err)
	}
	if info.Version != "27.5.1" || info.CPUs != 4 || info.Containers != 3 {
		t.Fatalf("unexpected info: %+v", info)
	}
	if info.APIVersion == "" {
		t.Error("expected negotiated API version")
	}
}
