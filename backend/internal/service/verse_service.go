package service

import (
	"bibleapp/backend/internal/domain"
	"bibleapp/backend/internal/repository"
	"context"
	"fmt"
	"log"
)

// --- Service Layer ---

// VerseService provides access to verse content
type VerseService interface {
	// GetVerseContent fetches the full text of a specific verse reference on demand
	GetVerseContent(ctx context.Context, reference string) (string, error)

	// EnrichDailyVerse takes a daily verse with just a reference and fetches the full content
	EnrichDailyVerse(ctx context.Context, verse domain.DailyVerse) (domain.DailyVerse, error)
}

type verseService struct {
	repo repository.VerseRepository
}

// NewVerseService creates a new VerseService
func NewVerseService(repo repository.VerseRepository) VerseService {
	return &verseService{repo: repo}
}

// GetVerseContent fetches the full text of a specific verse reference
func (s *verseService) GetVerseContent(ctx context.Context, reference string) (string, error) {
	log.Printf("INFO: Getting verse content for reference: %s", reference)

	// Get the verse text from the repository
	verseText, err := s.repo.GetVerseByReference(ctx, reference)
	if err != nil {
		return "", fmt.Errorf("failed to get verse content for %s: %w", reference, err)
	}

	return verseText, nil
}

// EnrichDailyVerse takes a daily verse with just a reference and fetches the full content
func (s *verseService) EnrichDailyVerse(ctx context.Context, verse domain.DailyVerse) (domain.DailyVerse, error) {
	// If verse already has content, just return it
	if verse.Text != "" {
		return verse, nil
	}

	// Get the verse text
	text, err := s.GetVerseContent(ctx, verse.Reference)
	if err != nil {
		return verse, err
	}

	// Update the verse with the content
	verse.Text = text

	return verse, nil
}
