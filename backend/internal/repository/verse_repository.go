package repository

import (
	"bibleapp/backend/internal/domain"
	"context"
	"errors"
)

// VerseRepository defines the interface for accessing verse data.
type VerseRepository interface {
	GetVerseForDay(ctx context.Context, dayOfYear int) (domain.BibleVerse, error)
}

// staticVerseRepository provides verses from a hardcoded list.
type staticVerseRepository struct {
	verses []domain.BibleVerse
}

// NewStaticVerseRepository creates a new repository with predefined verses.
func NewStaticVerseRepository() VerseRepository {
	// Initialize with the same verse list (add more!)
	verses := []domain.BibleVerse{
		{Book: "John", Chapter: 3, VerseNumber: 16, Text: "For God so loved the world that he gave his one and only Son, that whoever believes in him shall not perish but have eternal life."},
		{Book: "Philippians", Chapter: 4, VerseNumber: 13, Text: "I can do all this through him who gives me strength."},
		{Book: "Jeremiah", Chapter: 29, VerseNumber: 11, Text: "For I know the plans I have for you,” declares the LORD, “plans to prosper you and not to harm you, plans to give you hope and a future."},
		{Book: "Romans", Chapter: 8, VerseNumber: 28, Text: "And we know that in all things God works for the good of those who love him, who have been called according to his purpose."},
		{Book: "Proverbs", Chapter: 3, VerseNumber: 5, Text: "Trust in the LORD with all your heart and lean not on your own understanding;"},
		{Book: "Psalm", Chapter: 23, VerseNumber: 1, Text: "The LORD is my shepherd, I lack nothing."},
		{Book: "Joshua", Chapter: 1, VerseNumber: 9, Text: "Have I not commanded you? Be strong and courageous. Do not be afraid; do not be discouraged, for the LORD your God will be with you wherever you go."},
		{Book: "Matthew", Chapter: 6, VerseNumber: 33, Text: "But seek first his kingdom and his righteousness, and all these things will be given to you as well."},
		{Book: "Isaiah", Chapter: 40, VerseNumber: 31, Text: "but those who hope in the LORD will renew their strength. They will soar on wings like eagles; they will run and not grow weary, they will walk and not be faint."},
		{Book: "1 Corinthians", Chapter: 13, VerseNumber: 4, Text: "Love is patient, love is kind. It does not envy, it does not boast, it is not proud."},
		// Add many more verses here for better daily variety
	}

	// Pre-generate references
	for i := range verses {
		verses[i].GenerateReference()
	}

	return &staticVerseRepository{verses: verses}
}

// GetVerseForDay returns a verse based on the day of the year.
func (r *staticVerseRepository) GetVerseForDay(ctx context.Context, dayOfYear int) (domain.BibleVerse, error) {
	if len(r.verses) == 0 {
		return domain.BibleVerse{}, errors.New("no verses available in the repository")
	}
	// Use modulo to wrap around if dayOfYear exceeds the number of verses
	index := (dayOfYear - 1) % len(r.verses)
	if index < 0 { // Should not happen with YearDay() but good practice
		index = 0
	}
	return r.verses[index], nil
}
