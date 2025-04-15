package repository

import (
	"bibleapp/backend/internal/domain"
	"context"
	"errors"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserRepository defines the interface for user data storage operations.
// We might need more methods later (e.g., FindByID, Update).
type UserRepository interface {
	FindByGoogleID(ctx context.Context, googleID string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
	// Update potentially needed if user info from Google changes
	// Update(ctx context.Context, user *domain.User) error
}

// MongoUserRepository implements UserRepository using MongoDB.
type MongoUserRepository struct {
	collection *mongo.Collection
}

// NewMongoUserRepository creates a new instance of MongoUserRepository.
func NewMongoUserRepository(db *mongo.Database) *MongoUserRepository {
	collection := db.Collection("users")

	// Create indexes for faster lookups (important!)
	// Index on google_id (unique)
	googleIDIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "google_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	// Index on email (optional, depends on query needs)
	// emailIndex := mongo.IndexModel{
	// 	Keys:    bson.D{{Key: "email", Value: 1}},
	// 	Options: options.Index().SetUnique(false), // Email might not be unique if logins change
	// }

	_, err := collection.Indexes().CreateMany(context.Background(), []mongo.IndexModel{googleIDIndex})
	if err != nil {
		// Log the error but don't necessarily stop the application
		log.Printf("WARN: Could not create indexes on users collection: %v", err)
	}

	return &MongoUserRepository{collection: collection}
}

// FindByGoogleID finds a user by their Google ID.
func (r *MongoUserRepository) FindByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	filter := bson.M{"google_id": googleID}
	var user domain.User
	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // User not found, return nil error and nil user
		}
		log.Printf("ERROR: Failed to find user by Google ID %s: %v", googleID, err)
		return nil, err // Return other database errors
	}
	return &user, nil
}

// Create inserts a new user into the database.
func (r *MongoUserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	if user.ID != "" {
		return nil, errors.New("cannot create user with existing ID")
	}

	// Set timestamps
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Insert the user document
	// Let MongoDB generate the _id (ObjectID) automatically
	result, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		// Check for duplicate key error (e.g., unique index violation on google_id)
		if mongo.IsDuplicateKeyError(err) {
			log.Printf("WARN: Attempted to create user with duplicate Google ID: %s", user.GoogleID)
			// It might be better to fetch the existing user here instead of returning an error
			return nil, errors.New("user with this Google ID already exists") // Or a custom error type
		}
		log.Printf("ERROR: Failed to insert user: %v", err)
		return nil, err
	}

	// Get the generated ID and update the user struct
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		user.ID = oid.Hex() // Set the ID field in the returned user struct
		return user, nil
	} else {
		log.Printf("ERROR: Failed to get ObjectID from InsertOne result for user %s", user.GoogleID)
		// User was inserted, but we couldn't get the ID back easily.
		// We could try fetching it again, but returning an error might be safer.
		return nil, errors.New("failed to retrieve user ID after creation")
	}
}
