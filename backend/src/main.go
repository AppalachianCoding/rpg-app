package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.TraceLevel)

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Failed to load config: %s\n", err)
		return
	}
	logrus.Info("AWS config loaded")

	dbClient, err := newDbClient(ctx, cfg, DND_DATABASE)
	if err != nil {
		log.Fatalf("Failed to create db client: %s\n", err)
		return
	}
	defer dbClient.DB.Close()
	logrus.Info("DB client created")

	startServer(dbClient, "80")
}

func startServer(dbClient DbClient, port string) {
	logrus.WithFields(logrus.Fields{
		"port": port,
		"host": "0.0.0.0",
	}).Info("Starting server")

	r := gin.Default()
	r.UseRawPath = true

	r.GET("/health", healthCheckHandler)
	r.GET("/tables", tablesHandler)
	r.GET("/capabilities", capabilitiesHandler)

	r.GET("/all/:table", dbClient.allHandler)
	r.GET("/capabilities/:table", describeTable)

	r.GET("/", healthCheckHandler)
	r.GET("/:table/:name", dbClient.apiHandler)
	r.GET("/:table", dbClient.getAllNamesHandler)

	r.Run("0.0.0.0:" + port)
}
