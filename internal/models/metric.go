// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import "time"

// MetricSample is a point-in-time resource sample for an application, captured
// by the background scraper for short-term history.
type MetricSample struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	ApplicationID uint      `json:"application_id" gorm:"index:idx_metric_app_time;not null"`
	RecordedAt    time.Time `json:"recorded_at" gorm:"index:idx_metric_app_time"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryBytes   uint64    `json:"memory_bytes"`
	MemoryPercent float64   `json:"memory_percent"`
	NetRxBytes    uint64    `json:"net_rx_bytes"`
	NetTxBytes    uint64    `json:"net_tx_bytes"`
}
