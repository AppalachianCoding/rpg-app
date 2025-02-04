package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type RdsSecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

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

func connectToDb(ctx context.Context, cfg aws.Config, secretName string) (*sql.DB, error) {
	secret, err := getSecret(ctx, cfg, secretName)
	if err != nil {
		log.Warnf("failed to get secret, %v", err)
		return nil, err
	}
	endpoint := getEndpoint()

	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		secret.Username, secret.Password, endpoint, "5e-database")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open db, %v", err)
		return nil, err
	}

	return db, nil
}
