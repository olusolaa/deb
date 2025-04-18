package repository

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type VerseRepository interface {
	GetVerseByReference(ctx context.Context, reference string) (string, error)
	GetVersesByReferences(ctx context.Context, references []string) (map[string]string, error)
}

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

// verseRangeInfo stores information about a verse range for batch processing
type verseRangeInfo struct {
	book       string
	chapter    int
	startVerse int
	endVerse   int
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

// GetVersesByReferences fetches multiple verse texts in a single database operation
// Returns a map of reference -> verse text
func (r *MongoVerseRepository) GetVersesByReferences(ctx context.Context, references []string) (map[string]string, error) {
	result := make(map[string]string)

	// Create combined query for all references (singles and ranges)
	var allConditions []bson.M
	refToRangeInfo := make(map[string]verseRangeInfo)

	// Process all references and build query conditions
	for _, ref := range references {
		if isVerseRange(ref) {
			// Handle verse range
			conditions, rangeInfo, err := r.buildRangeQueryCondition(ref)
			if err != nil {
				log.Printf("WARN: Failed to process range reference '%s': %v", ref, err)
				continue
			}

			// Store range info for later processing
			refToRangeInfo[ref] = rangeInfo

			// Add range conditions to the query
			allConditions = append(allConditions, conditions)
		} else {
			// Handle single verse reference
			book, chapter, verse, err := parseReference(ref)
			if err != nil {
				log.Printf("WARN: Failed to parse reference '%s': %v", ref, err)
				continue
			}

			// Get book index and convert to database format
			bookIndex := getBookIndex(book)
			dbBook := fmt.Sprintf("Book %d", bookIndex+1) // Convert 0-based index to 1-based
			if bookIndex < 0 {
				// If book not found, try using the original name as fallback
				dbBook = book
			}

			// Create condition for this verse
			condition := bson.M{
				"book":    dbBook,
				"chapter": chapter,
				"verse":   verse,
			}

			allConditions = append(allConditions, condition)
		}
	}

	// If we have conditions, execute a single query for all verses
	if len(allConditions) > 0 {
		log.Printf("INFO: Executing batch query for %d references with %d conditions",
			len(references), len(allConditions))

		// Create a combined query with $or operator
		filter := bson.M{"$or": allConditions}

		// Execute the query
		cursor, err := r.collection.Find(ctx, filter)
		if err != nil {
			log.Printf("ERROR: Failed to execute batch query: %v", err)
			// Continue with empty result rather than returning error
		} else {
			defer cursor.Close(ctx)

			// Process results and build verse map (book -> dbVerseID -> BibleVerse struct)
			verseMap := make(map[string]map[int]BibleVerse)

			for cursor.Next(ctx) {
				var verse BibleVerse
				if err := cursor.Decode(&verse); err != nil {
					log.Printf("WARN: Failed to decode verse in batch: %v", err)
					continue
				}

				// Initialize book map if needed
				if _, exists := verseMap[verse.Book]; !exists {
					verseMap[verse.Book] = make(map[int]BibleVerse)
				}

				// Store the full verse struct, keyed by database verse ID
				verseMap[verse.Book][verse.Verse] = verse
			}

			// Process single references
			for _, ref := range references {
				if !isVerseRange(ref) {
					book, chapter, verse, err := parseReference(ref)
					if err != nil {
						continue
					}
					bookIndex := getBookIndex(book)
					dbBook := fmt.Sprintf("Book %d", bookIndex+1)
					if bookIndex < 0 {
						dbBook = book
					}

					// Calculate the expected database verse ID for the single verse
					dbVerseID := mapVerseNumber(bookIndex, chapter, verse)

					// Check if we have this verse in our results map
					if bookData, ok := verseMap[dbBook]; ok {
						if verseData, ok := bookData[dbVerseID]; ok {
							result[ref] = verseData.Text // Get text from stored struct
						}
					}
				}
			}

			// Process range references
			for ref, rangeInfo := range refToRangeInfo {
				var versesInRange []BibleVerse
				log.Printf("DEBUG: Processing batch range ref %s: %s %d:%d-%d",
					ref, rangeInfo.book, rangeInfo.chapter, rangeInfo.startVerse, rangeInfo.endVerse)

				// Get all verses fetched for this book
				if bookDataMap, bookFound := verseMap[rangeInfo.book]; bookFound {
					bookIndex := getBookIndex(rangeInfo.book) // Need index for simple verse extraction
					if bookIndex < 0 {
						// Attempt to get index from parsed book name if rangeInfo.book was a fallback
						parsedBook, _, _, _ := parseReference(ref) // Reparse might be inefficient but gets original book name
						bookIndex = getBookIndex(parsedBook)
					}

					// Iterate through all verses found for this book
					for dbVerseID, verseData := range bookDataMap {
						// Extract the simple verse number from the database ID
						simpleVerse := extractSimpleVerse(bookIndex, dbVerseID)
						if simpleVerse >= rangeInfo.startVerse && simpleVerse <= rangeInfo.endVerse {
							// If it's within the requested range, add it to our list
							versesInRange = append(versesInRange, verseData)
						}
					}

					// Sort the collected verses by simple verse number
					sort.Slice(versesInRange, func(i, j int) bool {
						// Need bookIndex to extract simple verse for comparison
						simpleI := extractSimpleVerse(bookIndex, versesInRange[i].Verse)
						simpleJ := extractSimpleVerse(bookIndex, versesInRange[j].Verse)
						// Handle potential extraction errors defensively
						if simpleI == -1 {
							return false
						} // Put errors at the end
						if simpleJ == -1 {
							return true
						}
						return simpleI < simpleJ
					})

					// Build the final string from sorted verses
					var versesText strings.Builder
					for _, verseData := range versesInRange {
						simpleVerse := extractSimpleVerse(bookIndex, verseData.Verse)
						if simpleVerse != -1 { // Only include verses where extraction worked
							if versesText.Len() > 0 {
								versesText.WriteString(" ")
							}
							versesText.WriteString(fmt.Sprintf("[%d] %s", simpleVerse, verseData.Text))
						} else {
							log.Printf("WARN: Skipping verse with failed simple verse extraction: BookIndex=%d, DBVerseID=%d", bookIndex, verseData.Verse)
						}
					}

					// If we found any verses in the range, store the result
					if versesText.Len() > 0 {
						result[ref] = versesText.String()
					} else {
						log.Printf("DEBUG: No verses found within range %d-%d for book %s after filtering %d fetched verses",
							rangeInfo.startVerse, rangeInfo.endVerse, rangeInfo.book, len(bookDataMap))
					}
				} else {
					log.Printf("DEBUG: Book %s not found in results map", rangeInfo.book)
				}
			}
		}
	}

	// Log results summary
	log.Printf("INFO: Fetched %d/%d requested verse references in batch", len(result), len(references))

	return result, nil
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

// mapVerseNumber converts a book index, chapter, and verse into the database's numeric verse ID format.
// Format depends on the book:
// - Genesis (index 0): ID is just the verse number
// - Other books (index > 0): ID is bookIndex * 1,000,000 + verse number
func mapVerseNumber(bookIndex, chapter, verse int) int {
	// Input validation: Ensure non-negative index and positive verse
	if bookIndex < 0 || verse <= 0 { // Chapter is not used in ID calculation based on samples
		log.Printf("WARN: Invalid input to mapVerseNumber: bookIndex=%d, verse=%d", bookIndex, verse)
		return -1 // Indicate error
	}

	if bookIndex == 0 { // Genesis
		return verse
	}
	// Other books
	return (bookIndex * 1000000) + verse
}

// extractSimpleVerse attempts to recover the simple verse number from the database verse ID format.
func extractSimpleVerse(bookIndex, dbVerseID int) int {
	if bookIndex < 0 || dbVerseID <= 0 {
		log.Printf("WARN: Invalid input to extractSimpleVerse: bookIndex=%d, dbVerseID=%d", bookIndex, dbVerseID)
		return -1 // Indicate error
	}

	if bookIndex == 0 { // Genesis
		return dbVerseID // For Genesis, the ID is the simple verse number
	}
	// For other books, ID is bookIndex * 1,000,000 + simple_verse_number
	simpleVerse := dbVerseID - (bookIndex * 1000000)
	if simpleVerse <= 0 {
		log.Printf("WARN: Calculated negative/zero simple verse for bookIndex=%d, dbVerseID=%d -> %d", bookIndex, dbVerseID, simpleVerse)
		return -1 // Indicate calculation error
	}
	return simpleVerse
}

// findSingleVerse finds a single verse with multiple fallback approaches
func (r *MongoVerseRepository) findSingleVerse(ctx context.Context, book string, chapter int, verse int) (string, error) {
	// First get the book index
	bookIndex := getBookIndex(book)
	var err error // Declare the err variable once at the function level

	// Convert book name to database format (Book X)
	dbBook := fmt.Sprintf("Book %d", bookIndex+1) // Convert 0-based index to 1-based
	if bookIndex < 0 {
		// If book not found, try using the original name as fallback
		dbBook = book
	}

	// Primary approach: use standard book/chapter/verse fields
	filter := bson.M{
		"book":    dbBook,
		"chapter": chapter,
		"verse":   verse,
	}

	// Try to find by primary fields
	var result BibleVerse
	err = r.collection.FindOne(ctx, filter).Decode(&result)
	if err == nil {
		return result.Text, nil
	}
	if bookIndex >= 0 {
		filter = bson.M{
			"book_index": bookIndex,
			"chapter":    chapter,
			"verse":      verse,
		}

		// Try to find by book_index/chapter/verse
		err = r.collection.FindOne(ctx, filter).Decode(&result)
		if err == nil {
			return result.Text, nil
		}
	}

	// If not found by book index, try by book name directly
	delete(filter, "book_index") // Remove book_index from filter if it was added
	filter["book"] = book

	err = r.collection.FindOne(ctx, filter).Decode(&result)
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

// buildRangeQueryCondition creates MongoDB query conditions for a verse range
func (r *MongoVerseRepository) buildRangeQueryCondition(reference string) (bson.M, verseRangeInfo, error) {
	// Parse the range
	book, chapter, startVerse, endVerse, err := parseVerseRange(reference)
	if err != nil {
		return nil, verseRangeInfo{}, err
	}

	// Validate the range
	if endVerse < startVerse {
		return nil, verseRangeInfo{}, fmt.Errorf("invalid verse range: end verse must be greater than or equal to start verse")
	}

	// Get book index and convert to database format
	bookIndex := getBookIndex(book)
	dbBook := fmt.Sprintf("Book %d", bookIndex+1) // Convert 0-based index to 1-based
	if bookIndex < 0 {
		// If book not found, try using the original name as fallback
		dbBook = book
	}

	// Create a range query: verses within this chapter between startVerse and endVerse
	// Map verse numbers to the database format (e.g., 1 -> 1002001 for Exodus 3:1)
	dbStartVerse := mapVerseNumber(bookIndex, chapter, startVerse)
	dbEndVerse := mapVerseNumber(bookIndex, chapter, endVerse)

	log.Printf("DEBUG: Mapping verse range %d-%d to database format: %d-%d", startVerse, endVerse, dbStartVerse, dbEndVerse)

	condition := bson.M{
		"book": dbBook,
		"verse": bson.M{
			"$gte": dbStartVerse,
			"$lte": dbEndVerse,
		},
	}

	// Return the condition and range info for later processing
	return condition, verseRangeInfo{
		book:       dbBook, // Store the database book name
		chapter:    chapter,
		startVerse: startVerse,
		endVerse:   endVerse,
	}, nil
}

// getVerseRange gets a range of verses and concatenates them
func (r *MongoVerseRepository) getVerseRange(ctx context.Context, reference string) (string, error) {
	// Parse the range
	condition, rangeInfo, err := r.buildRangeQueryCondition(reference)
	if err != nil {
		return "", err
	}

	// Execute query to get all verses in the range at once
	cursor, err := r.collection.Find(ctx, condition)
	if err != nil {
		return "", fmt.Errorf("database query failed: %w", err)
	}
	defer cursor.Close(ctx)

	// Add debug logging for the query condition
	log.Printf("DEBUG: Verse range query condition: %+v", condition)

	// Create a map to store verses by verse number
	verseMap := make(map[int]string)
	for cursor.Next(ctx) {
		var verse BibleVerse
		if err := cursor.Decode(&verse); err != nil {
			continue
		}
		// Log each verse we retrieve
		log.Printf("DEBUG: Retrieved verse %d: %s", verse.Verse, verse.Text[:20])
		verseMap[verse.Verse] = verse.Text
	}

	// Log the verse numbers found
	var foundVerses []int
	for v := range verseMap {
		foundVerses = append(foundVerses, v)
	}
	log.Printf("DEBUG: Found verses: %v for range %s, expected verses %d-%d",
		foundVerses, reference, rangeInfo.startVerse, rangeInfo.endVerse)

	// Build the formatted result in verse order
	var versesText strings.Builder
	bookIndex := getBookIndex(rangeInfo.book)
	if strings.HasPrefix(rangeInfo.book, "Book ") {
		// Extract book index from format "Book X"
		_, err := fmt.Sscanf(rangeInfo.book, "Book %d", &bookIndex)
		if err != nil {
			log.Printf("WARN: Could not parse book index from %s: %v", rangeInfo.book, err)
		}
		// Adjust to 0-based for internal use
		bookIndex--
	}

	for verse := rangeInfo.startVerse; verse <= rangeInfo.endVerse; verse++ {
		// Map the simple verse number to database format
		dbVerse := mapVerseNumber(bookIndex, rangeInfo.chapter, verse)
		log.Printf("DEBUG: Looking for verse %d (db format: %d)", verse, dbVerse)

		if text, ok := verseMap[dbVerse]; ok {
			if versesText.Len() > 0 {
				versesText.WriteString(" ")
			}
			log.Printf("DEBUG: Adding verse %d text to output", verse)  // Changed log message
			versesText.WriteString(fmt.Sprintf("[%d] %s", verse, text)) // Add verse number for clarity
		} else {
			log.Printf("DEBUG: Verse %d (db format %d) missing in retrieved map", verse, dbVerse) // Adjusted log message
		}
	}

	// Check if we found any verses in the range
	if versesText.Len() == 0 {
		return "", fmt.Errorf("no verses found in range %s %d:%d-%d",
			rangeInfo.book, rangeInfo.chapter, rangeInfo.startVerse, rangeInfo.endVerse)
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
