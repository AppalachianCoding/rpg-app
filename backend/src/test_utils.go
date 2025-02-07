package main

import (
	"database/sql"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/sirupsen/logrus"
)

func getTestDb() (*sql.DB, *embeddedpostgres.EmbeddedPostgres, error) {
	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Username("postgres").
		Password("postgres").
		Port(5432).
		DataPath("/tmp/postgres"),
	)

	err := postgres.Start()
	if err != nil {
		logrus.Errorf("Failed to start embedded postgres: %v", err)
		return nil, nil, err
	}

	dsn := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logrus.Errorf("Failed to open database: %v", err)
		postgres.Stop()
		return nil, nil, err
	}

	return db, postgres, nil
}
