package main

import (
	"encoding/json"
	"fmt"
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

	srv := startServer(dbClient, ":8082")

	for _, table := range TABLES {
		res, err := http.Post(
			fmt.Sprintf("http://localhost:8082/api/weapon_properties/%s", table.Name),
			"application/json",
			nil,
		)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", res.StatusCode)
		}

		dec := json.NewDecoder(res.Body)
		var body map[string]interface{}
		if err := dec.Decode(&body); err != nil {
			t.Fatalf("Failed to decode response body: %v", err)
		}

		var mapping = make(map[string]bool)
		for _, col := range table.Mapping {
			mapping[col] = false
		}

		for k, _ := range body {
			if _, ok := mapping[k]; !ok {
				t.Fatalf("Unexpected key in response: %s", k)
			}
			mapping[k] = true
		}

		for k, v := range mapping {
			if !v {
				t.Fatalf("Expected key not found in response: %s", k)
			}
		}

		logrus.Infof("Response: %s", body)
	}

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

	srv := startServer(dbClient, ":8081")

	res, err := http.Get("http://localhost:8081/api/all/weapon_properties")
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
