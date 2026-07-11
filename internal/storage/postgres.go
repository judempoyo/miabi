// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
	"fmt"

	"github.com/jkaninda/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ConnectPostgres opens a GORM connection to PostgreSQL without running migrations.
func ConnectPostgres(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	logger.Info("database connected")
	return db, nil
}
