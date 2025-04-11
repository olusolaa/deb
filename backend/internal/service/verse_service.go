package service

import (
	"bibleapp/backend/internal/domain"
	"bibleapp/backend/internal/repository"
	"context"
	"fmt"
	"time"
)

// --- Service Layer ---

// VerseService provides access to verse logic.
type VerseService interface {
	GetDailyVerse(ctx context.Context) (domain.BibleVerse, error)
}

type verseService struct {
	repo repository.VerseRepository
}

// NewVerseService creates a new VerseService.
func NewVerseService(repo repository.VerseRepository) VerseService {
	return &verseService{repo: repo}
}

// GetDailyVerse gets the verse for the current day.
func (s *verseService) GetDailyVerse(ctx context.Context) (domain.BibleVerse, error) {
	dayOfYear := time.Now().YearDay()
	verse, err := s.repo.GetVerseForDay(ctx, dayOfYear)
	if err != nil {
		return domain.BibleVerse{}, fmt.Errorf("failed to get verse for day %d: %w", dayOfYear, err)
	}
	return verse, nil
}
