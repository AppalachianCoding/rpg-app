package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type FiveETable struct {
	name    string
	mapping []string
	file    string
}

func convert_key(key string) string {
	switch key {
	case "index":
		return "_index"
	case "desc":
		return "_desc"
	default:
		return key
	}
}

func safeSQLValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		// Escape single quotes
		return "'" + strings.ReplaceAll(v, "'", "''") + "'"
	default:
		// Convert non-string values (e.g., maps, slices) to JSON
		jsonValue, err := json.Marshal(v)
		if err != nil {
			log.Printf("Error converting value to JSON: %v", err)
			return "NULL" // Fail gracefully
		}
		return "'" + strings.ReplaceAll(string(jsonValue), "'", "''") + "'" // Store as JSON string
	}
}

func createTable(table *FiveETable, db *sql.DB) error {
	log := log.WithField("table", table.name)

	query := "CREATE TABLE " + table.name + " ("
	for i, key := range table.mapping {
		key = convert_key(key)
		if i != 0 {
			query += ", "
		}
		query += key + " TEXT"
	}
	query += ");"

	log.WithField("query", query).Debug("Executing query")
	_, err := db.Exec(query)
	if err != nil {
		log.WithError(err).Error("Failed to create table")
		return err
	}

	log.Info("Created table")

	return nil
}

func insert(table *FiveETable, db *sql.DB, data []map[string]interface{}) error {
	var err error

	for _, row := range data {
		log := log.WithFields(log.Fields{
			"table": table,
			"row":   row,
		})

		for key := range row {
			found := false
			for _, k := range table.mapping {
				if key == k {
					found = true
				}
			}
			if !found {
				log.WithFields(logrus.Fields{
					"key": key,
				}).Fatalf("Extra key in row")
			}
		}

		query := "INSERT INTO " + table.name + " ("
		values := "VALUES ("
		for i, key := range table.mapping {
			if i != 0 {
				query += ", "
				values += ", "
			}
			query += convert_key(key)
			if row[key] == nil {
				log.WithFields(logrus.Fields{
					"key":   key,
					"value": row[key],
				}).Debug("NULL value")
				values += "NULL"
			} else {
				values += safeSQLValue(row[key])
			}
		}
		query += ") " + values + ");"

		_, err := db.Exec(query)
		if err != nil {
			log.WithError(err).Error("Failed to insert row")
			return err
		}

		log.Debug("Inserted row")
	}

	return err
}

func populate(db *sql.DB) error {
	log.Info("Populating database")
	dir := "5e_data"

	populated_rows, err := db.Query("SELECT tablename FROM pg_tables WHERE schemaname = 'public';")
	if err != nil {
		log.WithError(err).Error("Failed to query database for populated check")
		return err
	}
	defer populated_rows.Close()

	var already_populated = make(map[string]bool)
	for populated_rows.Next() {
		var name string
		if err := populated_rows.Scan(&name); err != nil {
			log.WithError(err).Error("Failed to scan row")
			continue
		}
		already_populated[name] = true
	}

	for _, table := range TABLES {
		if already_populated[table.name] {
			log.WithField("table", table.name).Debugf("Table already populated")
			continue
		}

		log := log.WithField("table", table.name)
		file := filepath.Join(dir, table.file)

		jsonFile, err := os.Open(file)
		defer jsonFile.Close()

		jsonData, err := io.ReadAll(jsonFile)
		if err != nil {
			log.WithError(err).Error("Failed to read JSON file")
			return err
		}

		var data []map[string]interface{}
		if err := json.Unmarshal(jsonData, &data); err != nil {
			log.WithError(err).Error("Failed to unmarshal JSON data")
			return err
		}

		if err := createTable(&table, db); err != nil {
			log.WithError(err).Error("Failed to create table")
			return err
		}

		if err := insert(&table, db, data); err != nil {
			log.WithError(err).Error("Failed to insert data")
			return err
		}
	}

	return nil
}
