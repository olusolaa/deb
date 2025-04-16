package repository

import (
	"context"
	"sync"
	"time"
)

// ChatUsageRepository defines methods for tracking chat usage
type ChatUsageRepository interface {
	// IncrementUsage increments the usage count for a user and returns the new count
	IncrementUsage(ctx context.Context, userID string) (int, error)

	// GetTodayUsage gets the current usage count for a user today
	GetTodayUsage(ctx context.Context, userID string) (int, error)
}

// Key structure for in-memory chat usage map
type chatUsageKey struct {
	UserID string
	Date   string // Date in YYYY-MM-DD format
}

// MemoryChatUsageRepository implements ChatUsageRepository with in-memory storage
type MemoryChatUsageRepository struct {
	usageMutex sync.RWMutex
	usageMap   map[chatUsageKey]int
}

// NewMemoryChatUsageRepository creates a new in-memory repository for tracking chat usage
func NewMemoryChatUsageRepository() ChatUsageRepository {
	return &MemoryChatUsageRepository{
		usageMap: make(map[chatUsageKey]int),
	}
}

// formatDate returns the date in YYYY-MM-DD format which we use as keys
func formatDate(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}

// GetTodayUsage gets the current usage count for a user today
func (r *MemoryChatUsageRepository) GetTodayUsage(ctx context.Context, userID string) (int, error) {
	r.usageMutex.RLock()
	defer r.usageMutex.RUnlock()

	todayKey := chatUsageKey{
		UserID: userID,
		Date:   formatDate(time.Now()),
	}

	count, exists := r.usageMap[todayKey]
	if !exists {
		return 0, nil
	}

	return count, nil
}

// IncrementUsage increments the usage count for a user and returns the new count
func (r *MemoryChatUsageRepository) IncrementUsage(ctx context.Context, userID string) (int, error) {
	r.usageMutex.Lock()
	defer r.usageMutex.Unlock()

	todayKey := chatUsageKey{
		UserID: userID,
		Date:   formatDate(time.Now()),
	}

	// Increment and return the new count
	r.usageMap[todayKey]++
	return r.usageMap[todayKey], nil
}
