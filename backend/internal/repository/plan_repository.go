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

// PlanRepository defines the interface for plan data storage operations.
type PlanRepository interface {
	Save(ctx context.Context, plan *domain.Plan) error
	FindByID(ctx context.Context, id string) (*domain.Plan, error)
	FindByUser(ctx context.Context, userID string) ([]*domain.Plan, error)
	// Add other methods like FindAll, Delete as needed
}

// MongoPlanRepository implements PlanRepository using MongoDB.
type MongoPlanRepository struct {
	collection *mongo.Collection
}

// NewMongoPlanRepository creates a new instance of MongoPlanRepository.
func NewMongoPlanRepository(db *mongo.Database) *MongoPlanRepository {
	collection := db.Collection("plans")

	// Consider creating indexes for common query fields, e.g., user_id
	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}}, // Index on user_id
		Options: options.Index().SetUnique(false), // user_id is likely not unique per plan
	}
	_, err := collection.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		log.Printf("WARN: Could not create 'user_id' index on plans collection: %v", err)
	}

	return &MongoPlanRepository{collection: collection}
}

// Save inserts or updates a plan in the database.
// It assumes plan.ID is empty for new plans and uses it for updates otherwise.
func (r *MongoPlanRepository) Save(ctx context.Context, plan *domain.Plan) error {
	now := time.Now()
	plan.UpdatedAt = now

	if plan.ID == "" { // New plan, insert
		plan.CreatedAt = now
		// Generate a new ObjectID for _id if it's not set (it shouldn't be for insert)
		// The driver handles this if we pass the struct directly, *but* we need the ID back.
		// Best practice is often to let the driver handle _id generation.
		
		// Clear potential empty string ID before insert if needed, though driver might handle it
		// if plan.ID == "" { ... }

		// We need to marshal without the ID field if we want mongo to generate it, 
		// or generate it ourselves. Let's generate it.
		objID := primitive.NewObjectID()
		plan.ID = objID.Hex() // Set the ID in the struct *before* inserting

		_, err := r.collection.InsertOne(ctx, plan) 
		if err != nil {
			log.Printf("ERROR: Failed to insert plan for user %s: %v", plan.UserID, err)
			// Reset ID if insert failed? Depends on desired behavior.
			// plan.ID = ""
			return err
		}
		log.Printf("INFO: Inserted new plan with ID: %s for user: %s", plan.ID, plan.UserID)
	} else { // Existing plan, update
		objID, err := primitive.ObjectIDFromHex(plan.ID)
		if err != nil {
			log.Printf("ERROR: Invalid plan ID format for update: %s", plan.ID)
			return errors.New("invalid plan ID format")
		}
		filter := bson.M{"_id": objID}

		// Use bson.M or bson.D for updates to avoid replacing the whole document unintentionally
		// $set is generally safer
		update := bson.M{
			"$set": bson.M{
				"user_id":     plan.UserID,
				"title":       plan.Title,
				"description": plan.Description,
				"duration":    plan.Duration,
				"verses":      plan.Verses,
				"status":      plan.Status,
				"updated_at":  plan.UpdatedAt, // Update the updated_at timestamp
			},
			"$setOnInsert": bson.M{
				"created_at": plan.CreatedAt, // Keep original created_at if upsert happens (though we separate insert/update logic here)
			},
		}

		// Or update the whole document if the structure is stable:
		// update := bson.M{"\$set": plan}

		// Use UpdateOne instead of ReplaceOne unless you intend to replace the whole doc
		result, err := r.collection.UpdateOne(ctx, filter, update)
		if err != nil {
			log.Printf("ERROR: Failed to update plan %s: %v", plan.ID, err)
			return err
		}
		if result.MatchedCount == 0 {
			log.Printf("WARN: Plan with ID %s not found for update", plan.ID)
			return mongo.ErrNoDocuments // Return specific error
		}
		log.Printf("INFO: Updated plan with ID: %s", plan.ID)
	}
	return nil
}

// FindByID retrieves a plan by its MongoDB ObjectID string.
func (r *MongoPlanRepository) FindByID(ctx context.Context, id string) (*domain.Plan, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Printf("ERROR: Invalid plan ID format for find: %s", id)
		return nil, errors.New("invalid plan ID format")
	}
	filter := bson.M{"_id": objID}
	var plan domain.Plan
	err = r.collection.FindOne(ctx, filter).Decode(&plan)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Standard practice: return nil, nil for not found
		}
		log.Printf("ERROR: Failed to find plan %s: %v", id, err)
		return nil, err
	}
	return &plan, nil
}

// FindByUser retrieves all plans associated with a specific user ID.
func (r *MongoPlanRepository) FindByUser(ctx context.Context, userID string) ([]*domain.Plan, error) {
	filter := bson.M{"user_id": userID}
	// Optionally add sorting, e.g., by creation date descending
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		log.Printf("ERROR: Failed to execute find query for user %s plans: %v", userID, err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var plans []*domain.Plan
	if err = cursor.All(ctx, &plans); err != nil {
		log.Printf("ERROR: Failed to decode plans for user %s: %v", userID, err)
		return nil, err
	}

	// If no documents are found, cursor.All returns an empty slice and nil error
	if plans == nil {
		plans = []*domain.Plan{} // Ensure non-nil slice is returned
	}

	return plans, nil
}
