package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/gorilla/mux"
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

	srv := startServer(ctx, dbClient, ":80")
	defer srv.Shutdown(ctx)
	logrus.Info("Server started")
	select {}
}

func startServer(ctx context.Context, dbClient DbClient, port string) *http.Server {
	logrus.Info("Starting server...")

	r := mux.NewRouter()
	r.UseEncodedPath()

	r.HandleFunc("/health", healthCheckHandler).Methods("GET")
	r.HandleFunc("/tables", tablesHandler).Methods("GET")
	r.HandleFunc("/capabilities", capabilitiesHandler).Methods("GET")

	r.HandleFunc("/all/{table}", dbClient.allHandler).Methods("GET")
	r.HandleFunc("/capabilities/{table}", describeTable).Methods("GET")

	r.HandleFunc("/", healthCheckHandler).Methods("GET")
	r.HandleFunc("/{table}/{name}", dbClient.apiHandler).Methods("GET")
	r.HandleFunc("/{table}", dbClient.getAllNamesHandler).Methods("GET")

	srv := &http.Server{
		Addr:         port,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Println("Starting server on", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Server failed: %s\n", err)
		}
	}()

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit

		log.Println("Shutting down server...")
		srv.Shutdown(ctx)
	}()

	return srv
}
