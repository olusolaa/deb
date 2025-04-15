package db

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
)

// ConnectMongoDB establishes a connection to MongoDB using the provided URI.
func ConnectMongoDB(ctx context.Context, uri string) (*mongo.Client, error) {
	log.Println("INFO: Attempting to connect to MongoDB...")
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Printf("ERROR: Failed to create MongoDB client: %v", err)
		return nil, err
	}

	// Ping the primary node to verify connection within the given context timeout
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Printf("ERROR: Failed to ping MongoDB primary: %v", err)
		// Attempt to disconnect if ping fails after connection attempt
		_ = client.Disconnect(context.Background()) // Use background context for cleanup
		return nil, err
	}

	log.Println("INFO: Successfully connected and pinged MongoDB.")
	return client, nil
}
