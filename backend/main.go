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

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

// I have no idea what this is. Change this
const DATABASE = "5e-collection"

func apiHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	table := vars["table"]
	name := vars["name"]

	collection := client.Database(DATABASE).Collection(table)
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

func connectMongo() {
	var err error
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %s\n", err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatalf("MongoDB ping failed: %s\n", err)
	}
	log.Println("Connected to MongoDB")
}

func main() {
	connectMongo()

	r := mux.NewRouter()
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
