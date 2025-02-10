package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func verifyTable(table string) bool {
	for _, t := range TABLE_NAMES {
		if t == table {
			return true
		}
	}
	return false
}

type DbClient struct {
	*sql.DB
}

func QueryDb(
	w http.ResponseWriter,
	r *http.Request,
	db *sql.DB,
	log *logrus.Entry,
	query string,
	params ...interface{},
) error {
	rows, err := db.Query(query, params...)
	if err != nil {
		log.WithError(err).Warn("Failed to query database")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to query database"))
		return err
	}
	defer rows.Close()

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)

	columns, err := rows.Columns()
	if err != nil {
		log.WithError(err).Warn("Failed to get columns")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to get columns"))
		return err
	}
	log.Debugf("Columns: %v\n", columns)
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		result := make(map[string]interface{})

		if err := rows.Scan(valuePtrs...); err != nil {
			log.WithError(err).Warn("Failed to scan row")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to scan row"))
			return err
		}

		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				result[col] = string(b)
			} else {
				result[col] = val
			}
		}

		if err := enc.Encode(result); err != nil {
			log.WithError(err).Warn("Failed to encode result")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to encode result"))
			return err
		}

		log.Tracef("Wrote %s", result)
	}

	log.Info("Finished query")
	return nil
}

func newDbClient(ctx context.Context, cfg aws.Config, database string) (DbClient, error) {
	db, err := connectToDb(ctx, cfg, os.Getenv("DB_SECRET"), database)
	if err != nil {
		logrus.Fatalf("Failed to connect to db: %s\n", err)
	}
	dbClient := DbClient{db}

	if err = populate(db); err != nil {
		logrus.Errorf("Failed to populate database: %v", err)
		return dbClient, err
	}

	return dbClient, nil
}

func (dbc DbClient) apiHandler(w http.ResponseWriter, r *http.Request) {
	db := dbc.DB
	vars := mux.Vars(r)
	table := vars["table"]
	name := vars["name"]
	log := logrus.WithFields(logrus.Fields{
		"table":  table,
		"name":   name,
		"method": "api",
		"ip":     r.RemoteAddr,
	})
	log.Debugf("Received request for %s from %s\n", name, table)

	if !verifyTable(table) {
		log.Warnf("Invalid table %s\n", table)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid table"))
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE name = $1", table)
	if err := QueryDb(w, r, db, log, query, name); err != nil {
		log.WithError(err).Warn("Failed to get data")
	}
}

func (dbc DbClient) allHandler(w http.ResponseWriter, r *http.Request) {
	db := dbc.DB
	vars := mux.Vars(r)
	table := vars["table"]
	log := logrus.WithFields(logrus.Fields{
		"table":  table,
		"method": "all",
		"ip":     r.RemoteAddr,
	})
	log.Debugf("Received request for all from %s\n", table)

	query := fmt.Sprintf("SELECT * FROM %s", table)

	if err := QueryDb(w, r, db, log, query); err != nil {
		log.WithError(err).Warn("Failed to get all data")
	}
}

type APICapability struct {
	Path        string   `json:"path"`
	Methods     []string `json:"methods"`
	Description string   `json:"description,omitempty"`
}

func capabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	logrus.WithFields(logrus.Fields{
		"method": "capabilities",
		"ip":     r.RemoteAddr,
	}).Info("Received request for capabilities")
	capabilities := []APICapability{
		{
			Path:    "/api/{table}/{name}",
			Methods: []string{"POST"},
			Description: "Handles API requests for a specific table and name. " +
				"Used to insert or update data.",
		},
		{
			Path:        "/api/all/{table}",
			Methods:     []string{"GET"},
			Description: "Retrieves all records from a specified table.",
		},
		{
			Path:    "/api/capabilities",
			Methods: []string{"GET"},
			Description: "Returns a list of all available API endpoints and their " +
				"methods and descriptions.",
		},
		{
			Path:        "/api/capabilities/{table}",
			Methods:     []string{"GET"},
			Description: "Returns the feilds of a table",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(capabilities)
}

func tablesHandler(w http.ResponseWriter, r *http.Request) {
	logrus.WithFields(logrus.Fields{
		"method": "tables",
		"ip":     r.RemoteAddr,
	}).Info("Received request for tables")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TABLE_NAMES)
}

func describeTable(w http.ResponseWriter, r *http.Request) {
	log := logrus.WithFields(logrus.Fields{
		"method": "describe",
		"ip":     r.RemoteAddr,
	})
	vars := mux.Vars(r)
	t, ok := vars["table"]
	if !ok {
		log.Warn("No table specified")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("No table specified"))
		return
	}
	log = log.WithField("table", t)
	log.Debugf("Received request for table %s", t)

	w.Header().Set("Content-Type", "application/json")
	table, ok := TABLES[t]
	if !ok {
		log.Warnf("Table %s not found", t)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Table not found"))
		return
	}
	log.Debugf("Returning table %s", table)
	json.NewEncoder(w).Encode(table)
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
