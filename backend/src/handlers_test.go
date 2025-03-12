package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestApiHandler(t *testing.T) {
	logrus.SetLevel(logrus.InfoLevel)
	log := logrus.WithField("test", "TestApiHandler")

	db, pg, err := getTestDb()
	if err != nil {
		t.Fatalf("Failed to get test database: %v", err)
	}
	defer pg.Stop()
	defer db.Close()

	populate(db)

	dbClient := DbClient{db}
	defer dbClient.Close()

	t.Logf("Starting server")
	go func() {
		startServer(dbClient, "8082")
	}()
	t.Logf("Started server")

	for _, table := range TABLES {
		log = log.WithField("table", table.Name)
		namesRes, err := http.Get(fmt.Sprintf("http://localhost:8082/%s",
			url.PathEscape(table.Name)))
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		if namesRes.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", namesRes.StatusCode)
		}

		dec := json.NewDecoder(namesRes.Body)

		for dec.More() {
			var line map[string]string
			if err := dec.Decode(&line); err != nil {
				t.Logf("Line: %s", line)
				t.Fatalf("Failed to decode response body: %v", err)
			}
			name := line["name"]
			log = log.WithField("name", name)
			name = convertKey(name)
			log.Debugf("Getting data for %s/%s", url.PathEscape(table.Name), url.PathEscape(name))
			res, err := http.Get(fmt.Sprintf("http://localhost:8082/%s/%s",
				url.PathEscape(table.Name),
				url.PathEscape(name),
			))
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			if res.StatusCode == http.StatusNoContent {
				t.Logf("No content for %s/%s", table.Name, name)
				continue
			}
			if res.StatusCode != http.StatusOK {
				t.Fatalf("Expected status 200, got %d", res.StatusCode)
			}

			dec := json.NewDecoder(res.Body)
			var body map[string]interface{}
			if err := dec.Decode(&body); err != nil {
				t.Logf("Body: %s", body)
				t.Fatalf("Failed to decode response body: %v", err)
			}

			var mapping = make(map[string]bool)
			for _, col := range table.Mapping {
				mapping[col] = false
			}

			for k := range body {
				if _, ok := mapping[k]; !ok {
					log.Errorf("Unexpected key in response: %s", k)
					t.Fatalf("Unexpected key in response: %s", k)
				}
				mapping[k] = true
			}

			for k, v := range mapping {
				if !v {
					log.Errorf("Expected key not found in response: %s", k)
					t.Fatalf("Expected key not found in response: %s", k)
				}
			}

			log.Debugf("Response: %s", body)
		}
	}
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
	defer dbClient.Close()

	t.Logf("Starting server")
	go func() {
		startServer(dbClient, "8081")
	}()
	t.Logf("Started server")

	for table := range TABLES {
		url := fmt.Sprintf("http://localhost:8081/all/%s", table)
		res, err := http.Get(url)
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
		_, err = json.MarshalIndent(body, "", "  ")
	}
}
