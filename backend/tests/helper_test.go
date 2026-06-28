//go:build integration

// Package tests contains integration tests that require live MySQL and Redis.
// Run with: make test-integration
package tests

import (
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/glimjoe/sentinel-api-platform/internal/pkg/config"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN()), &gorm.Config{})
	if err != nil {
		testDB = nil
	} else {
		testDB = db
	}

	m.Run()
}
