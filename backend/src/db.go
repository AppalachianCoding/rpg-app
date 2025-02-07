package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type RdsSecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

const DND_DATABASE = "5e"

func getSecret(ctx context.Context, cfg aws.Config, secretName string) (*RdsSecret, error) {
	client := secretsmanager.NewFromConfig(cfg)

	input := &secretsmanager.GetSecretValueInput{
		SecretId: &secretName,
	}

	result, err := client.GetSecretValue(ctx, input)
	if err != nil {
		log.Warnf("failed to get secret, %v", err)
		return nil, err
	}

	var secret RdsSecret
	err = json.Unmarshal([]byte(*result.SecretString), &secret)
	if err != nil {
		log.Warnf("failed to unmarshal secret, %v", err)
		return nil, err
	}

	return &secret, nil
}

func getEndpoint() string {
	return fmt.Sprintf("%s:%s", os.Getenv("DB_ENDPOINT"), os.Getenv("DB_PORT"))
}

func createDatabase(db *sql.DB, databaseName string) error {
	log.Info("Creating database, if exists")

	query := fmt.Sprintf("CREATE DATABASE \"%s\"", databaseName)
	rows, err := db.Query(query)
	if err != nil {
		log.WithError(err).Error("Failed to create database")
		return err
	}
	defer rows.Close()
	for row := rows.Next(); row; row = rows.Next() {
		log.Debugf("Row: %v", row)
	}

	log.Info("Created database")

	return nil
}

func connectToDb(ctx context.Context, cfg aws.Config, secretName string, databaseName string) (*sql.DB, error) {
	secret, err := getSecret(ctx, cfg, secretName)
	if err != nil {
		log.Warnf("failed to get secret, %v", err)
		return nil, err
	}
	logrus.Info("Successfully retrieved secret")
	logrus.Debugf("Password: %s Username: %s", secret.Password, secret.Username)

	endpoint := getEndpoint()
	logrus.Debugf("Endpoint: %s", endpoint)

	var dsn string
	if databaseName != "" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=require",
			url.QueryEscape(secret.Username),
			url.QueryEscape(secret.Password),
			endpoint, databaseName)
	} else {
		dsn = fmt.Sprintf("postgres://%s:%s@%s?sslmode=require",
			url.QueryEscape(secret.Username),
			url.QueryEscape(secret.Password),
			endpoint)
	}
	logrus.Debugf("DSN: %s", dsn)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.WithError(err).Error("Failed to connect to db")
		return nil, err
	}
	if err = db.Ping(); err != nil {
		does_not_exist_err := fmt.Sprintf("pq: database \"%s\" does not exist", databaseName)
		if err.Error() == does_not_exist_err {
			logrus.Infof("Database does not exist, creating database %s", databaseName)
			dbClient, err := connectToDb(ctx, cfg, secretName, "postgres")
			if err != nil {
				logrus.Errorf("failed to connect to postgres via default \"database\" postgres, %v", err)
				return nil, err
			}
			if err = createDatabase(dbClient, databaseName); err != nil {
				logrus.Errorf("failed to create database, %v", err)
				return nil, err
			}
			connectToDb(ctx, cfg, secretName, databaseName)
		} else {
			log.WithError(err).Error("Failed to ping db")
			return nil, err
		}
	}
	logrus.Info("Successfully connected to db")

	return db, nil
}
