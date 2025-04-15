package domain

import (
	"time"
)

// User represents a user in the system.
// We store minimal info obtained from OAuth and our internal ID.
type User struct {
	ID        string    `bson:"_id,omitempty" json:"id"`         // Our internal MongoDB ID (_id)
	GoogleID  string    `bson:"google_id" json:"google_id"` // Google's unique user ID
	Email     string    `bson:"email" json:"email"`           // User's email
	Name      string    `bson:"name" json:"name"`             // User's display name
	Picture   string    `bson:"picture" json:"picture"`       // URL to profile picture
	CreatedAt time.Time `bson:"created_at" json:"created_at"` // Timestamp of user creation
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"` // Timestamp of last update
}
