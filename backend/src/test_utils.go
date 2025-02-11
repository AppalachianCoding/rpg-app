package main

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"os"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/sirupsen/logrus"
)

func getTestDb() (*sql.DB, *embeddedpostgres.EmbeddedPostgres, error) {
	tmpPath, err := os.CreateTemp("", "postgres")
	rng := rand.Reader
	rngInt, _ := rand.Int(rng, big.NewInt(4000))
	randomPort := rngInt.Int64() + 1000
	if err != nil {
		logrus.Errorf("Failed to create temporary directory: %v", err)
		return nil, nil, err
	}
	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Username("postgres").
		Password("postgres").
		Port(uint32(randomPort)).
		DataPath(tmpPath.Name()),
	)

	err = postgres.Start()
	if err != nil {
		logrus.Errorf("Failed to start embedded postgres: %v", err)
		return nil, nil, err
	}

	dsn := fmt.Sprintf("postgres://postgres:postgres@localhost:%d/postgres?sslmode=disable", randomPort)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logrus.Errorf("Failed to open database: %v", err)
		postgres.Stop()
		return nil, nil, err
	}

	return db, postgres, nil
}
