package main

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

func TestPopulateDb(t *testing.T) {
	log.SetLevel(log.InfoLevel)

	db, pg, err := getTestDb()
	if err != nil {
		t.Fatalf("Failed to get test database: %v", err)
	}
	defer pg.Stop()
	defer db.Close()

	if err := populate(db); err != nil {
		t.Fatalf("Failed to populate database: %v", err)
	}
}
