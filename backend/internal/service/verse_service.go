package service

import (
	"bibleapp/backend/internal/domain"
	"bibleapp/backend/internal/repository"
	"bibleapp/backend/internal/util"
	"context"
	"fmt"
	"log"
	"strings"
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

	// Split and normalize the reference(s)
	references := util.SplitReferences(reference)
	log.Printf("INFO: Split reference into %d parts", len(references))

	// If it's a single reference with no splitting, use the simpler method
	if len(references) == 1 && references[0] == reference {
		verseText, err := s.repo.GetVerseByReference(ctx, reference)
		if err != nil {
			return "", fmt.Errorf("failed to get verse content for %s: %w", reference, err)
		}
		return verseText, nil
	}

	// For multiple references or normalized references, use the batch method
	log.Printf("INFO: Using batch processing for %d references", len(references))

	// Get all verses in a single database call
	verseMap, err := s.repo.GetVersesByReferences(ctx, references)
	if err != nil {
		log.Printf("WARN: Batch retrieval encountered an error: %v", err)
		// Fall back to individual retrieval if batch fails
		return s.fallbackIndividualRetrieval(ctx, references, reference)
	}

	// Check if we got any results
	if len(verseMap) == 0 {
		return "", fmt.Errorf("failed to get any verse content for reference %s", reference)
	}

	// For multiple references, combine them
	if len(references) > 1 {
		var allTexts []string
		for _, ref := range references {
			if text, ok := verseMap[ref]; ok {
				allTexts = append(allTexts, text)
			}
		}

		if len(allTexts) == 0 {
			return "", fmt.Errorf("failed to get any verse content for reference %s", reference)
		}

		// Join the texts with newlines
		return strings.Join(allTexts, "\n\n"), nil
	}

	// For a normalized single reference
	normalizedRef := references[0]
	if text, ok := verseMap[normalizedRef]; ok {
		return text, nil
	}

	// Shouldn't reach here, but just in case
	return "", fmt.Errorf("failed to get verse content for %s", reference)
}

// fallbackIndividualRetrieval handles individual verse retrieval as a fallback
func (s *verseService) fallbackIndividualRetrieval(ctx context.Context, references []string, originalRef string) (string, error) {
	log.Printf("INFO: Falling back to individual retrieval for %d references", len(references))

	var allTexts []string
	for _, ref := range references {
		text, err := s.repo.GetVerseByReference(ctx, ref)
		if err != nil {
			log.Printf("WARN: Failed to get content for reference '%s': %v", ref, err)
			continue
		}
		allTexts = append(allTexts, text)
	}

	if len(allTexts) == 0 {
		return "", fmt.Errorf("failed to get any verse content for reference %s", originalRef)
	}

	// Join the texts with newlines
	return strings.Join(allTexts, "\n\n"), nil
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
