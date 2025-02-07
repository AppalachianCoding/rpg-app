package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

func apiHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	table := vars["table"]
	name := vars["name"]

	collection := client.Database(DND_DATABASE).Collection(table)
	filter := bson.D{{"name", name}}
	var result bson.M
	err := collection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)

	fmt.Printf("Echoed %s from %s\n", name, table)
}

func allHander(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	table := vars["table"]

	collection := client.Database(DND_DATABASE).Collection(table)

	cursor, err := collection.Find(context.Background(),
		bson.D{},
		options.Find().SetProjection(bson.D{{"name", 1}}),
	)
	if err != nil {
		log.Fatalf("Failed to find: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var results []bson.M
	if err = cursor.All(context.Background(), &results); err != nil {
		log.Fatalf("Failed to decode: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)

	fmt.Printf("Echoed all from %s\n", table)
}

func main() {
	logrus.SetLevel(logrus.TraceLevel)
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Failed to load config: %s\n", err)
		return
	}
	dbClient, err := connectToDb(ctx, cfg, os.Getenv("DB_SECRET"), DND_DATABASE)
	if err != nil {
		log.Fatalf("Failed to connect to db: %s\n", err)
	}
	if err = populate(dbClient); err != nil {
		log.Fatalf("Failed to populate db: %s\n", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/api/all/{table}", allHander).Methods("GET")
	r.HandleFunc("/api/{table}/{name}", apiHandler).Methods("POST")

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Println("Starting server on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %s\n", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	client.Disconnect(context.TODO())
	srv.Close()
}
