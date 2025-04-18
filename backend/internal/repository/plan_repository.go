package repository

import (
	"bibleapp/backend/internal/domain"
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Error definitions
var (
	ErrDayOutOfRange = errors.New("day number is out of plan range")
)

// PlanRepository defines the interface for plan data storage operations.
type PlanRepository interface {
	Save(ctx context.Context, plan *domain.ReadingPlan) error
	FindByID(ctx context.Context, id string) (*domain.ReadingPlan, error)
	FindByUser(ctx context.Context, userID string) ([]*domain.ReadingPlan, error)
	Delete(ctx context.Context, id string) error
	// Add other methods like FindAll as needed
}

// MongoPlanRepository implements PlanRepository using MongoDB.
type MongoPlanRepository struct {
	collection *mongo.Collection
}

// FindByUser retrieves all plans associated with a specific user ID.
func (r *MongoPlanRepository) FindByUser(ctx context.Context, userID string) ([]*domain.ReadingPlan, error) {
	// Filter by userID - using case-insensitive regex to handle potential case variations
	// For "default" specifically, we'll use an exact match
	var filter bson.M
	if userID == "default" {
		filter = bson.M{"user_id": "default"}
		log.Printf("INFO: Fetching default plans with exact match")
	} else {
		filter = bson.M{"user_id": userID}
		log.Printf("INFO: Fetching plans for userID: %s", userID)
	}

	// Count total matching documents for debugging
	count, countErr := r.collection.CountDocuments(ctx, filter)
	if countErr != nil {
		log.Printf("WARN: Failed to count documents for user %s: %v", userID, countErr)
	} else {
		log.Printf("DEBUG: Found %d plan documents in database for user %s", count, userID)
	}

	// Debug - Show all plans in the collection
	debugCursor, debugErr := r.collection.Find(ctx, bson.M{})
	if debugErr == nil {
		defer debugCursor.Close(ctx)
		var allPlans []*domain.ReadingPlan
		if debugCursor.All(ctx, &allPlans) == nil {
			log.Printf("DEBUG: Total plans in database: %d", len(allPlans))
			for i, plan := range allPlans {
				log.Printf("DEBUG: Plan #%d - ID: %s, UserID: '%s', Topic: '%s'",
					i+1, plan.ID, plan.UserID, plan.Topic)
			}
		}
	}

	// Sort by creation date descending (newest first)
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		log.Printf("ERROR: Failed to execute find query for user %s plans: %v", userID, err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var plans []*domain.ReadingPlan
	if err = cursor.All(ctx, &plans); err != nil {
		log.Printf("ERROR: Failed to decode plans for user %s: %v", userID, err)
		return nil, err
	}

	// If no documents are found, cursor.All returns an empty slice and nil error
	if plans == nil {
		plans = []*domain.ReadingPlan{} // Ensure non-nil slice is returned
	}

	// Log result details
	log.Printf("INFO: Found %d plans for user %s", len(plans), userID)

	return plans, nil
}

// NewMongoPlanRepository creates a new instance of MongoPlanRepository.
func NewMongoPlanRepository(db *mongo.Database) *MongoPlanRepository {
	collection := db.Collection("plans")

	// Consider creating indexes for common query fields, e.g., user_id (if applicable)
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "target_audience", Value: 1}}, // Index on target_audience as an example
		Options: options.Index().SetUnique(false),
	}
	_, err := collection.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		log.Printf("WARN: Could not create 'user_id' index on plans collection: %v", err)
	}

	return &MongoPlanRepository{collection: collection}
}

// Save inserts or updates a plan in the database.
// It assumes plan.ID is empty for new plans and uses it for updates otherwise.
func (r *MongoPlanRepository) Save(ctx context.Context, plan *domain.ReadingPlan) error {
	now := time.Now()

	if plan.ID == uuid.Nil { // New plan, insert
		plan.CreatedAt = now
		// Generate a new UUID for the plan
		plan.ID = uuid.New()

		_, err := r.collection.InsertOne(ctx, plan)
		if err != nil {
			log.Printf("ERROR: Failed to insert plan: %v", err)
			return err
		}
		log.Printf("INFO: Inserted new plan with ID: %s", plan.ID.String())
	} else { // Existing plan, update
		filter := bson.M{"_id": plan.ID}

		// Use bson.M or bson.D for updates to avoid replacing the whole document unintentionally
		// $set is generally safer
		update := bson.M{
			"$set": bson.M{
				"topic":           plan.Topic,
				"duration_days":   plan.DurationDays,
				"target_audience": plan.TargetAudience,
				"daily_verses":    plan.DailyVerses,
			},
			"$setOnInsert": bson.M{
				"created_at": plan.CreatedAt, // Keep original created_at if upsert happens
			},
		}

		// Use UpdateOne instead of ReplaceOne unless you intend to replace the whole doc
		result, err := r.collection.UpdateOne(ctx, filter, update)
		if err != nil {
			log.Printf("ERROR: Failed to update plan %s: %v", plan.ID.String(), err)
			return err
		}
		if result.MatchedCount == 0 {
			log.Printf("WARN: Plan with ID %s not found for update", plan.ID.String())
			return mongo.ErrNoDocuments // Return specific error
		}
		log.Printf("INFO: Updated plan with ID: %s", plan.ID.String())
	}
	return nil
}

// FindByID retrieves a plan by its MongoDB ObjectID string.
func (r *MongoPlanRepository) FindByID(ctx context.Context, id string) (*domain.ReadingPlan, error) {
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		log.Printf("ERROR: Invalid plan UUID format for find: %s", id)
		return nil, errors.New("invalid plan UUID format")
	}
	filter := bson.M{"_id": parsedUUID}
	var plan domain.ReadingPlan
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

// Delete removes a plan by its ID
func (r *MongoPlanRepository) Delete(ctx context.Context, id string) error {
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		log.Printf("ERROR: Invalid plan UUID format for delete: %s", id)
		return errors.New("invalid plan UUID format")
	}
	filter := bson.M{"_id": parsedUUID}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		log.Printf("ERROR: Failed to delete plan %s: %v", id, err)
		return err
	}
	if result.DeletedCount == 0 {
		log.Printf("WARN: Plan with ID %s not found for deletion", id)
		return mongo.ErrNoDocuments
	}
	log.Printf("INFO: Successfully deleted plan with ID: %s", id)
	return nil
}
