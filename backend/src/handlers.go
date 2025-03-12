package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gin-gonic/gin"
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
	c *gin.Context,
	db *sql.DB,
	log *logrus.Entry,
	query string,
	params ...interface{},
) error {
	log = log.WithFields(logrus.Fields{
		"method": c.Request.Method,
		"ip":     c.RemoteIP(),
		"query":  query,
		"params": params,
	})
	rows, err := db.Query(query, params...)
	if err != nil {
		log.WithError(err).Warn("Failed to query database")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query database"})
		return err
	}
	defer rows.Close()

	c.Header("Content-Type", "application/json")
	enc := json.NewEncoder(c.Writer)
	enc.SetIndent("", "")
	enc.SetEscapeHTML(false)

	columns, err := rows.Columns()
	if err != nil {
		log.WithError(err).Warn("Failed to get columns")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get columns"})
		return err
	}
	log.Tracef("Columns: %v\n", columns)
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	anyResults := false
	for rows.Next() {
		result := make(map[string]interface{})

		if err := rows.Scan(valuePtrs...); err != nil {
			log.WithError(err).Warn("Failed to scan row")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan row"})
			return err
		}

		for i, col := range columns {
			col = convertToKey(col)
			val := values[i]
			if b, ok := val.([]byte); ok {
				result[col] = string(b)
			} else {
				result[col] = val
			}
		}

		if len(result) == 0 {
			log.Warn("No results in row")
			c.JSON(http.StatusNotFound, gin.H{"error": "No results"})
			continue
		}

		if err := enc.Encode(result); err != nil {
			log.WithError(err).Warn("Failed to encode result")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode result"})
			return err
		}

		log.Tracef("Wrote %s", result)
		anyResults = true
	}

	if !anyResults {
		log.Error("No results")
		c.JSON(http.StatusNoContent, gin.H{"error": "No results"})
	}

	c.Writer.Flush()
	log.Debugf("Finished query")
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

func (dbc DbClient) apiHandler(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	db := dbc.DB

	table := c.Param("table")
	table, _ = url.PathUnescape(table)
	name := c.Param("name")
	name, _ = url.PathUnescape(name)

	log := logrus.WithFields(logrus.Fields{
		"table":  table,
		"name":   name,
		"method": "api",
		"ip":     c.RemoteIP(),
	})
	log.Debugf("Received request for %s from %s\n", name, table)

	if !verifyTable(table) {
		log.Warnf("Invalid table %s\n", table)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid table"})
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE name = $1", table)
	log = log.WithField("query", query)
	if err := QueryDb(c, db, log, query, name); err != nil {
		log.WithError(err).Warn("Failed to get data")
	}
}

func (dbc DbClient) allHandler(c *gin.Context) {
	db := dbc.DB
	table := c.Param("table")
	table, _ = url.PathUnescape(table)
	log := logrus.WithFields(logrus.Fields{
		"table":  table,
		"method": "all",
		"ip":     c.RemoteIP(),
	})
	log.Debugf("Received request for all from %s\n", table)

	if !verifyTable(table) {
		log.Warnf("Invalid table %s\n", table)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid table"})
	}

	query := fmt.Sprintf("SELECT * FROM %s", table)

	if err := QueryDb(c, db, log, query); err != nil {
		log.WithError(err).Warn("Failed to get all data")
	}
}

func (dbc DbClient) getAllNamesHandler(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	db := dbc.DB
	table := c.Param("table")
	table, _ = url.PathUnescape(table)
	log := logrus.WithFields(logrus.Fields{
		"table":  table,
		"method": "allNames",
		"ip":     c.RemoteIP(),
	})
	log.Debugf("Received request for all names from %s\n", table)

	if !verifyTable(table) {
		log.Warnf("Invalid table %s\n", table)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid table"})
	}

	query := fmt.Sprintf("SELECT name FROM %s", table)

	if err := QueryDb(c, db, log, query); err != nil {
		log.WithError(err).Warn("Failed to get all names")
	}
}

type APICapability struct {
	Path        string   `json:"path"`
	Methods     []string `json:"methods"`
	Description string   `json:"description,omitempty"`
}

func capabilitiesHandler(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	logrus.WithFields(logrus.Fields{
		"method": "capabilities",
		"ip":     c.RemoteIP(),
	}).Debugf("Received request for capabilities")
	capabilities := []APICapability{
		{
			Path:    "/{table}/{name}",
			Methods: []string{"GET"},
			Description: "Handles API requests for a specific table and name. " +
				"Used to insert or update data.",
		},
		{
			Path:        "/{table}",
			Methods:     []string{"GET"},
			Description: "Retrieves all names for all records in a specified table.",
		},
		{
			Path:        "/all/{table}",
			Methods:     []string{"GET"},
			Description: "Retrieves all records from a specified table.",
		},
		{
			Path:    "/capabilities",
			Methods: []string{"GET"},
			Description: "Returns a list of all available API endpoints and their " +
				"methods and descriptions.",
		},
		{
			Path:        "/capabilities/{table}",
			Methods:     []string{"GET"},
			Description: "Returns the feilds of a table",
		},
	}

	c.JSON(http.StatusOK, capabilities)
}

func tablesHandler(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	logrus.WithFields(logrus.Fields{
		"method": "tables",
		"ip":     c.RemoteIP(),
	}).Debugf("Received request for tables")

	c.JSON(http.StatusOK, TABLE_NAMES)
}

func describeTable(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	log := logrus.WithFields(logrus.Fields{
		"method": "describe",
		"ip":     c.RemoteIP(),
	})

	tablereq := c.Param("table")
	log = log.WithField("table", tablereq)
	log.Debugf("Received request for table %s", tablereq)

	table, ok := TABLES[tablereq]
	if !ok {
		log.Warnf("Table %s not found", tablereq)
		c.JSON(http.StatusNotFound, gin.H{"error": "Table not found"})
		return
	}
	log.Tracef("Returning table %s", table)

	c.JSON(http.StatusOK, table)
}

func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
