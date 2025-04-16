package repository

import (
	"context"
	"fmt"
	"log"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoVerseRepository implements VerseRepository interface using MongoDB
type MongoVerseRepository struct {
	collection *mongo.Collection
}

// BibleVerse represents a verse in the MongoDB collection
type BibleVerse struct {
	Book        string `bson:"book"`
	BookIndex   int    `bson:"book_index"`
	Chapter     int    `bson:"chapter"`
	Verse       int    `bson:"verse"`
	Text        string `bson:"text"`
	Translation string `bson:"translation"`
}

// NewMongoVerseRepository creates a new repository that uses MongoDB to fetch verse content
func NewMongoVerseRepository(db *mongo.Database) VerseRepository {
	// Ensure the collection exists
	col := db.Collection("bible_verses")

	// Create indexes for better performance
	indexModels := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "book", Value: 1}, {Key: "chapter", Value: 1}, {Key: "verse", Value: 1}, {Key: "translation", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "book_index", Value: 1}},
		},
	}

	// Create indexes in the background
	_, err := col.Indexes().CreateMany(context.Background(), indexModels)
	if err != nil {
		log.Printf("WARN: Failed to create indexes for bible_verses collection: %v", err)
	}

	log.Printf("INFO: Successfully initialized MongoDB Bible verse repository")
	return &MongoVerseRepository{collection: col}
}

// GetVerseByReference fetches the verse text from MongoDB based on the reference
func (r *MongoVerseRepository) GetVerseByReference(ctx context.Context, reference string) (string, error) {
	log.Printf("INFO: Getting verse content from MongoDB for reference: %s", reference)

	// Parse the reference, checking if it's a range or single verse
	if isVerseRange(reference) {
		// Handle verse range (e.g., "John 3:16-18")
		return r.getVerseRange(ctx, reference)
	}

	// Parse the reference (e.g., "John 3:16" -> book="John", chapter=3, verse=16)
	book, chapter, verse, err := parseReference(reference)
	if err != nil {
		return "", err
	}

	// Try to find the verse using various methods
	text, err := r.findSingleVerse(ctx, book, chapter, verse)
	if err != nil {
		return "", err
	}

	return text, nil
}

// parseReference parses a verse reference like "John 3:16" into components
func parseReference(reference string) (book string, chapter int, verse int, err error) {
	// Handle various reference formats
	parts := strings.Split(reference, " ")
	if len(parts) < 2 {
		return "", 0, 0, fmt.Errorf("invalid reference format: %s", reference)
	}

	// Extract book name (could be multiple words)
	bookParts := parts[:len(parts)-1]
	book = strings.Join(bookParts, " ")

	// Extract chapter and verse
	chapterVerse := parts[len(parts)-1]
	chapterVerseParts := strings.Split(chapterVerse, ":")
	if len(chapterVerseParts) != 2 {
		return "", 0, 0, fmt.Errorf("invalid chapter:verse format: %s", chapterVerse)
	}

	var chapterInt, verseInt int
	_, err = fmt.Sscanf(chapterVerseParts[0], "%d", &chapterInt)
	if err != nil {
		return "", 0, 0, fmt.Errorf("invalid chapter number: %s", chapterVerseParts[0])
	}

	// Handle potential dash indicating a range in the verse part
	versePart := chapterVerseParts[1]
	if strings.Contains(versePart, "-") {
		// For a range, just return the start verse - the range handling is in different function
		verseParts := strings.Split(versePart, "-")
		_, err = fmt.Sscanf(verseParts[0], "%d", &verseInt)
	} else {
		_, err = fmt.Sscanf(versePart, "%d", &verseInt)
	}

	if err != nil {
		return "", 0, 0, fmt.Errorf("invalid verse number: %s", versePart)
	}

	return book, chapterInt, verseInt, nil
}

// Helper function to check if a reference is a verse range
func isVerseRange(reference string) bool {
	// Look for patterns like "John 3:16-18"
	return strings.Contains(reference, "-")
}

// Helper function to parse verse range
func parseVerseRange(reference string) (book string, chapter int, startVerse int, endVerse int, err error) {
	// First get the basic components
	book, chapter, startVerse, err = parseReference(reference)
	if err != nil {
		return "", 0, 0, 0, err
	}

	// Extract the range part
	parts := strings.Split(reference, " ")
	chapterVerse := parts[len(parts)-1]
	chapterVerseParts := strings.Split(chapterVerse, ":")
	verseParts := strings.Split(chapterVerseParts[1], "-")
	if len(verseParts) != 2 {
		return "", 0, 0, 0, fmt.Errorf("invalid verse range format: %s", reference)
	}

	// Parse end verse
	_, err = fmt.Sscanf(verseParts[1], "%d", &endVerse)
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("invalid end verse number: %s", verseParts[1])
	}

	return book, chapter, startVerse, endVerse, nil
}

// mapVerseNumber converts from standard Bible verse numbers to the database verseid format
func mapVerseNumber(bookIndex int, chapter int, verse int) int {
	// According to the README:
	// Verseid field is a combination of Book + Chapter + Verse
	// First two digits: Book (0-65)
	// Next three digits: Chapter
	// Last three digits: Verse
	// All indices are 0-based in the original data

	// Adjusting for 0-based indexing in the original data
	adjustedChapter := chapter - 1
	adjustedVerse := verse - 1

	// Create the verseid according to the format
	return (bookIndex * 1000000) + (adjustedChapter * 1000) + adjustedVerse
}

// Helper function to find a single verse with multiple fallback approaches
func (r *MongoVerseRepository) findSingleVerse(ctx context.Context, book string, chapter int, verse int) (string, error) {
	// First get the book index
	bookIndex := getBookIndex(book)
	var err error // Declare the err variable once at the function level

	// Use the original database format approach - search by verseid
	if bookIndex >= 0 {
		// Map to the verseid format described in the README
		mappedVerse := mapVerseNumber(bookIndex, chapter, verse)

		// Create a filter based on our understanding of the database format
		filter := bson.M{
			"verse": mappedVerse,
		}

		// Try to find by verseid
		var result BibleVerse
		err = r.collection.FindOne(ctx, filter).Decode(&result)
		if err == nil {
			return result.Text, nil
		}
	}

	// Fallback to the standard approach if the above doesn't work
	// Create a filter for this verse
	filter := bson.M{
		"chapter": chapter,
		"verse":   verse,
	}

	// First try to query by book_index, which is most reliable with our imported data
	// Note: We already calculated bookIndex earlier, no need to call getBookIndex again
	if bookIndex >= 0 {
		filter["book_index"] = bookIndex

		// Try to find by book index first
		var result BibleVerse
		err = r.collection.FindOne(ctx, filter).Decode(&result) // Using = since err is declared at function level
		if err == nil {
			return result.Text, nil
		}
	}

	// If not found by book index, try by book name directly
	delete(filter, "book_index") // Remove book_index from filter if it was added
	filter["book"] = book

	var result BibleVerse
	err = r.collection.FindOne(ctx, filter).Decode(&result) // Using = instead of := since err is already declared
	if err == nil {
		return result.Text, nil
	}

	// If not found, try with alternative book name formats
	bookAlt := convertBookName(book)
	if bookAlt != book {
		filter["book"] = bookAlt
		err = r.collection.FindOne(ctx, filter).Decode(&result)
		if err == nil {
			return result.Text, nil
		}
	}

	// If still not found, try with 'Book N' format used in our imported data
	if bookIndex >= 0 {
		filter["book"] = fmt.Sprintf("Book %d", bookIndex+1) // Book indices are 0-based, but Book N is 1-based
		err = r.collection.FindOne(ctx, filter).Decode(&result)
		if err == nil {
			return result.Text, nil
		}
	}

	// If we still can't find it, return an error
	if err == mongo.ErrNoDocuments {
		return "", fmt.Errorf("verse %s %d:%d not found", book, chapter, verse)
	}
	return "", fmt.Errorf("failed to query verse: %w", err)
}

// getVerseRange gets a range of verses and concatenates them
func (r *MongoVerseRepository) getVerseRange(ctx context.Context, reference string) (string, error) {
	// Parse the range
	book, chapter, startVerse, endVerse, err := parseVerseRange(reference)
	if err != nil {
		return "", err
	}

	// Validate the range
	if endVerse < startVerse {
		return "", fmt.Errorf("invalid verse range: end verse must be greater than or equal to start verse")
	}

	if endVerse-startVerse > 30 {
		return "", fmt.Errorf("verse range too large: maximum is 30 verses")
	}

	// Build verses text
	var versesText strings.Builder
	for verse := startVerse; verse <= endVerse; verse++ {
		// Get this verse
		text, err := r.findSingleVerse(ctx, book, chapter, verse)
		if err != nil {
			// If this specific verse is not found, log and continue to next
			log.Printf("WARN: Verse %s %d:%d in range not found", book, chapter, verse)
			continue
		}

		// Add verse number and text
		if versesText.Len() > 0 {
			versesText.WriteString(" ")
		}
		versesText.WriteString(fmt.Sprintf("(%d) %s", verse, text))
	}

	// Check if we found any verses in the range
	if versesText.Len() == 0 {
		return "", fmt.Errorf("no verses found in range %s %d:%d-%d", book, chapter, startVerse, endVerse)
	}

	return versesText.String(), nil
}

// convertBookName handles alternative book name formats
func convertBookName(book string) string {
	// Map of common abbreviated forms to full names
	bookMap := map[string]string{
		"1 Cor":   "1 Corinthians",
		"1 Jn":    "1 John",
		"1 Pet":   "1 Peter",
		"1 Thess": "1 Thessalonians",
		"2 Cor":   "2 Corinthians",
		"2 Jn":    "2 John",
		"2 Pet":   "2 Peter",
		"2 Thess": "2 Thessalonians",
		"1 Tim":   "1 Timothy",
		"2 Tim":   "2 Timothy",
		"Rev":     "Revelation",
		"Prov":    "Proverbs",
		"Ps":      "Psalm",
		"Psa":     "Psalm",
		"Psalms":  "Psalm",
		"Eph":     "Ephesians",
		"Phil":    "Philippians",
		"Col":     "Colossians",
		"Jas":     "James",
		"Heb":     "Hebrews",
		"Matt":    "Matthew",
		"Rom":     "Romans",
		"Gal":     "Galatians",
		"Deut":    "Deuteronomy",
	}

	if fullName, ok := bookMap[book]; ok {
		return fullName
	}

	return book
}

// getBookIndex returns the numeric index for a given book name
func getBookIndex(book string) int {
	// Map of book names to their index numbers
	bookIndices := map[string]int{
		"Genesis": 0, "Exodus": 1, "Leviticus": 2, "Numbers": 3, "Deuteronomy": 4,
		"Joshua": 5, "Judges": 6, "Ruth": 7, "1 Samuel": 8, "2 Samuel": 9,
		"1 Kings": 10, "2 Kings": 11, "1 Chronicles": 12, "2 Chronicles": 13, "Ezra": 14,
		"Nehemiah": 15, "Esther": 16, "Job": 17, "Psalm": 18, "Proverbs": 19,
		"Ecclesiastes": 20, "Song of Solomon": 21, "Isaiah": 22, "Jeremiah": 23, "Lamentations": 24,
		"Ezekiel": 25, "Daniel": 26, "Hosea": 27, "Joel": 28, "Amos": 29,
		"Obadiah": 30, "Jonah": 31, "Micah": 32, "Nahum": 33, "Habakkuk": 34,
		"Zephaniah": 35, "Haggai": 36, "Zechariah": 37, "Malachi": 38, "Matthew": 39,
		"Mark": 40, "Luke": 41, "John": 42, "Acts": 43, "Romans": 44,
		"1 Corinthians": 45, "2 Corinthians": 46, "Galatians": 47, "Ephesians": 48, "Philippians": 49,
		"Colossians": 50, "1 Thessalonians": 51, "2 Thessalonians": 52, "1 Timothy": 53, "2 Timothy": 54,
		"Titus": 55, "Philemon": 56, "Hebrews": 57, "James": 58, "1 Peter": 59,
		"2 Peter": 60, "1 John": 61, "2 John": 62, "3 John": 63, "Jude": 64, "Revelation": 65,
	}

	// Normalize book name and check for index
	if index, ok := bookIndices[book]; ok {
		return index
	}

	// Try with alternative name
	altName := convertBookName(book)
	if index, ok := bookIndices[altName]; ok {
		return index
	}

	return -1 // Not found
}
