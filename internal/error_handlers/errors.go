// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package errorhandlers

import (
	"errors"
	"net/http"

	"github.com/jkaninda/okapi"
	"github.com/miabi-io/miabi/internal/dto"
)

// CustomErrorHandler returns an okapi.ErrorHandler that formats every error as
// the standard {success, data, error} envelope.
func CustomErrorHandler() okapi.ErrorHandler {
	return func(c *okapi.Context, code int, message string, err error) error {
		var errStr string
		errCode := httpStatusToCode(code)
		if err != nil {
			errStr = err.Error()
			// Errors carrying a stable machine code (e.g. quota.QuotaError ->
			// "QUOTA_EXCEEDED") override the generic status-derived code.
			var coder interface{ Code() string }
			if errors.As(err, &coder) {
				if cc := coder.Code(); cc != "" {
					errCode = cc
				}
			}
			// Keep message human-facing. AbortWithError(code, err) passes only the
			// generic status text (e.g. "Conflict") as message and the real reason
			// as err — promote it so the UI shows the reason, not the status word.
			if message == "" || message == http.StatusText(code) {
				message = errStr
				// A domain error may expose a curated user message distinct from its
				// raw Error() string — prefer it for the human-facing `message` while
				// `error` keeps the raw detail.
				var msgr interface{ Message() string }
				if errors.As(err, &msgr) {
					if m := msgr.Message(); m != "" {
						message = m
					}
				}
			}
		}
		if message == "" {
			message = http.StatusText(code)
		}
		return c.JSON(code, dto.ErrorResponseBody{
			Success: false,
			Data:    nil,
			Error: &dto.ErrorInfo{
				StatusCode: code,
				Code:       errCode,
				Message:    message,
				Error:      errStr,
			},
		})
	}
}

func httpStatusToCode(status int) string {
	switch status {
	case 400:
		return "BAD_REQUEST"
	case 401:
		return "UNAUTHORIZED"
	case 402:
		return "PAYMENT_REQUIRED"
	case 403:
		return "FORBIDDEN"
	case 404:
		return "NOT_FOUND"
	case 405:
		return "METHOD_NOT_ALLOWED"
	case 409:
		return "CONFLICT"
	case 422:
		return "UNPROCESSABLE_ENTITY"
	case 429:
		return "TOO_MANY_REQUESTS"
	case 500:
		return "INTERNAL_SERVER_ERROR"
	case 502:
		return "BAD_GATEWAY"
	case 503:
		return "SERVICE_UNAVAILABLE"
	default:
		return "ERROR"
	}
}
