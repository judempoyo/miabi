// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package database

import (
	"testing"
	"time"

	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/storage/repositories"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// sqlite-friendly stand-ins for the real models: the full models embed UIDModel,
// whose column carries a Postgres `gen_random_uuid()` default that sqlite cannot
// parse at DDL time. These mirror only the columns the resolver reads/writes,
// under the same table names, so the repository (which uses the real models with
// SELECT *) reads them back fine.
type dbInstRow struct {
	ID          uint `gorm:"primaryKey"`
	UID         string
	WorkspaceID uint
	Name        string
	DisplayName string
	Engine      string
	Version     string
	Status      string
}

func (dbInstRow) TableName() string { return "database_instances" }

type logicalDBRow struct {
	ID          uint `gorm:"primaryKey"`
	UID         string
	WorkspaceID uint
	InstanceID  uint
	Name        string
	Username    string
	Status      string
	Metadata    models.Metadata `gorm:"serializer:json"`
	CreatedAt   time.Time
}

func (logicalDBRow) TableName() string { return "databases" }

// instNetRow is the empty many2many join FindInWorkspace's Preload("Networks")
// reads; with no rows the preload short-circuits without touching the networks
// table.
type instNetRow struct {
	DatabaseInstanceID uint `gorm:"primaryKey"`
	NetworkID          uint `gorm:"primaryKey"`
}

func (instNetRow) TableName() string { return "database_instance_networks" }

func newDeclNameSvc(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&dbInstRow{}, &logicalDBRow{}, &instNetRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return &Service{repo: repositories.NewDatabaseRepository(db)}, db
}

// TestFindDatabaseByDeclName is the heart of the GitOps resolver fix: a manifest
// database name resolves to the *specific* logical database stamped with that
// name — not "the first/most-recent database on the instance". Here one shared
// Postgres instance hosts authentik_db (untagged, created first) and posta_db
// (tagged "posta-db", created later); resolving "posta-db" must return posta_db,
// never authentik_db.
func TestFindDatabaseByDeclName(t *testing.T) {
	s, db := newDeclNameSvc(t)
	const ws = uint(1)

	if err := db.Create(&dbInstRow{
		ID: 1, UID: "inst-1", WorkspaceID: ws, Name: "shared-pg", DisplayName: "shared-pg",
		Engine: string(models.DBEnginePostgres), Version: "17", Status: string(models.DBStatusRunning),
	}).Error; err != nil {
		t.Fatalf("seed instance: %v", err)
	}

	rows := []logicalDBRow{
		{
			ID: 1, UID: "db-authentik", WorkspaceID: ws, InstanceID: 1, Name: "authentik_db",
			Username: "u_authentik", Status: string(models.DBStatusRunning), CreatedAt: time.Unix(1000, 0),
		},
		{
			ID: 2, UID: "db-posta", WorkspaceID: ws, InstanceID: 1, Name: "posta_db",
			Username: "u_posta", Status: string(models.DBStatusRunning),
			CreatedAt: time.Unix(2000, 0), // newer → would be ListDatabases dbs[0] (DESC)
			Metadata:  models.SetBuiltin(models.Metadata{}, models.MetaDeclarativeName, "posta-db"),
		},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed databases: %v", err)
	}

	got, gotInst, ok := s.FindDatabaseByDeclName(ws, "posta-db")
	if !ok {
		t.Fatal("FindDatabaseByDeclName(posta-db): not found")
	}
	if got.Name != "posta_db" {
		t.Errorf("resolved logical db = %q, want posta_db (must not be authentik_db)", got.Name)
	}
	if gotInst == nil || gotInst.ID != 1 {
		t.Errorf("resolved instance = %v, want id 1", gotInst)
	}

	// An untagged database is never resolved by a declarative name.
	if _, _, ok := s.FindDatabaseByDeclName(ws, "authentik-db"); ok {
		t.Error("FindDatabaseByDeclName(authentik-db): unexpectedly matched an untagged database")
	}
	if _, _, ok := s.FindDatabaseByDeclName(ws, ""); ok {
		t.Error("FindDatabaseByDeclName(\"\"): empty name must never match")
	}
}
