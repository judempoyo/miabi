// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package edgegateway

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/miabi-io/miabi/internal/models"
)

// gatewayWebPort is the edge gateway's published HTTP entry point (see
// RenderConfig entryPoints.web), where the reload endpoint is served.
const gatewayWebPort = 80

// reloadEndpointPath is Goma's default on-demand reload endpoint path.
const reloadEndpointPath = "/gateway/reload"

// reloadHTTPClient is the client used for reload calls: a short timeout so a
// slow/unreachable node never blocks the caller for long.
var reloadHTTPClient = &http.Client{Timeout: 10 * time.Second}

// Reload tells the edge gateway fronting srv to pull and apply its configuration
// immediately (POST /reload), instead of waiting for the HTTP-provider poll
// interval. It is a no-op for the local manager, which uses a watched file
// provider that already reloads on write. token is the node's gateway token (the
// same value injected as GOMA_RELOAD_TOKEN on the gateway).
func (s *Service) Reload(ctx context.Context, srv *models.Server, token string) error {
	if usesFileProvider(srv) {
		return nil
	}
	if srv == nil || srv.Address == "" {
		return fmt.Errorf("edgegateway: server has no address to reach its gateway")
	}
	baseURL := "http://" + net.JoinHostPort(srv.Address, strconv.Itoa(gatewayWebPort))
	if err := postReload(ctx, reloadHTTPClient, baseURL, token); err != nil {
		return fmt.Errorf("edgegateway: reload %q: %w", srv.Name, err)
	}
	return nil
}

// postReload issues the authenticated POST <baseURL>/reload request.
func postReload(ctx context.Context, client *http.Client, baseURL, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+reloadEndpointPath, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("gateway returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
