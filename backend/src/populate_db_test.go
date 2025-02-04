package main

import (
	"database/sql"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

func TestPopulateDb(t *testing.T) {
	log.SetLevel(log.InfoLevel)

	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Username("postgres").
		Password("postgres").
		Port(5432),
	)

	err := postgres.Start()
	if err != nil {
		log.Fatalf("Failed to start embedded Postgres: %v", err)
	}
	defer postgres.Stop()

	dsn := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := populate(db); err != nil {
		t.Errorf("Failed to populate database: %v", err)
	}
}
