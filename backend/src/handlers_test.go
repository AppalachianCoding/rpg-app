package main

import (
	"io"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestApiHandler(t *testing.T) {
	logrus.SetLevel(logrus.InfoLevel)

	db, pg, err := getTestDb()
	if err != nil {
		t.Fatalf("Failed to get test database: %v", err)
	}
	defer pg.Stop()
	defer db.Close()

	populate(db)

	dbClient := DbClient{db}

	srv := startServer(dbClient, ":8080")

	res, err := http.Post("http://localhost:8080/api/weapon_properties/Versatile", "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	logrus.Infof("Response: %s", body)

	srv.Shutdown(nil)
}

func TestAllHandler(t *testing.T) {
	logrus.SetLevel(logrus.InfoLevel)

	db, pg, err := getTestDb()
	if err != nil {
		t.Fatalf("Failed to get test database: %v", err)
	}
	defer pg.Stop()
	defer db.Close()

	populate(db)

	dbClient := DbClient{db}

	srv := startServer(dbClient, ":8080")

	res, err := http.Get("http://localhost:8080/api/all/weapon_properties")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	logrus.Infof("Response: %s", body)

	srv.Shutdown(nil)
}
