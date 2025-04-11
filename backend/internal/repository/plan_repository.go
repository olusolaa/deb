package repository

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"bibleapp/backend/internal/domain" // Adjust import path if needed
	"github.com/google/uuid"
)

var (
	ErrPlanNotFound  = errors.New("reading plan not found")
	ErrNoActivePlan  = errors.New("no active reading plan found")
	ErrDayOutOfRange = errors.New("day number is out of range for the plan duration")
)

// PlanRepository defines the interface for storing and retrieving reading plans.
type PlanRepository interface {
	SavePlan(ctx context.Context, plan *domain.ReadingPlan) error
	GetPlanByID(ctx context.Context, id uuid.UUID) (domain.ReadingPlan, error)
	GetLatestPlan(ctx context.Context) (domain.ReadingPlan, error)
	ListAllPlans(ctx context.Context) ([]domain.ReadingPlan, error)
}

// inMemoryPlanRepository implements PlanRepository using a map.
// NOTE: Data is lost on application restart.
type inMemoryPlanRepository struct {
	mu    sync.RWMutex
	plans map[uuid.UUID]domain.ReadingPlan
	// Store IDs ordered by creation time to easily find the latest
	orderedIDs []uuid.UUID
}

// NewInMemoryPlanRepository creates a new in-memory repository.
func NewInMemoryPlanRepository() PlanRepository {
	return &inMemoryPlanRepository{
		plans:      make(map[uuid.UUID]domain.ReadingPlan),
		orderedIDs: make([]uuid.UUID, 0),
	}
}

func (r *inMemoryPlanRepository) SavePlan(ctx context.Context, plan *domain.ReadingPlan) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if plan.ID == uuid.Nil {
		plan.ID = uuid.New() // Assign ID if not present
	}
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = time.Now().UTC()
	}

	// Ensure DailyVerses are sorted by DayNumber (LLM might not guarantee order)
	sort.SliceStable(plan.DailyVerses, func(i, j int) bool {
		return plan.DailyVerses[i].DayNumber < plan.DailyVerses[j].DayNumber
	})

	_, exists := r.plans[plan.ID]
	r.plans[plan.ID] = *plan

	// Add to ordered list only if it's a new plan
	if !exists {
		r.orderedIDs = append(r.orderedIDs, plan.ID)
	} else {
		// If updating, ensure CreatedAt doesn't change unnecessarily affecting "latest"
		// For simplicity here, we just overwrite. A real DB handles this better.
		// If needed, re-sort orderedIDs by CreatedAt if updates could change order.
	}

	return nil
}

func (r *inMemoryPlanRepository) GetPlanByID(ctx context.Context, id uuid.UUID) (domain.ReadingPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plan, found := r.plans[id]
	if !found {
		return domain.ReadingPlan{}, ErrPlanNotFound
	}
	return plan, nil
}

// GetLatestPlan returns the most recently created plan.
func (r *inMemoryPlanRepository) GetLatestPlan(ctx context.Context) (domain.ReadingPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.orderedIDs) == 0 {
		return domain.ReadingPlan{}, ErrNoActivePlan
	}

	latestID := r.orderedIDs[len(r.orderedIDs)-1]
	plan, found := r.plans[latestID]
	if !found {
		// This indicates an internal inconsistency (ID in list but not in map)
		return domain.ReadingPlan{}, errors.New("internal inconsistency: latest plan ID not found in map")
	}
	return plan, nil
}

func (r *inMemoryPlanRepository) ListAllPlans(ctx context.Context) ([]domain.ReadingPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]domain.ReadingPlan, 0, len(r.plans))
	// Return in reverse chronological order (newest first)
	for i := len(r.orderedIDs) - 1; i >= 0; i-- {
		id := r.orderedIDs[i]
		if plan, ok := r.plans[id]; ok {
			list = append(list, plan)
		}
	}
	return list, nil
}
