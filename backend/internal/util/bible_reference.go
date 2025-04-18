package util

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

// NormalizeBibleReference standardizes Bible reference formats
// to a consistent pattern: "BookName Chapter:Verse" or "BookName Chapter:StartVerse-EndVerse"
func NormalizeBibleReference(reference string) string {
	// Trim any whitespace
	reference = strings.TrimSpace(reference)

	// Check if it's already in the standard format
	standardFormatRegex := regexp.MustCompile(`^([1-3]?\s*[A-Za-z]+)\s+(\d+):(\d+)(?:-(\d+))?$`)
	if standardFormatRegex.MatchString(reference) {
		// Already in standard format, ensure spaces are correct
		parts := standardFormatRegex.FindStringSubmatch(reference)
		book := strings.TrimSpace(parts[1])
		chapter := parts[2]
		verse := parts[3]

		if parts[4] != "" { // It's a range
			return fmt.Sprintf("%s %s:%s-%s", book, chapter, verse, parts[4])
		}
		return fmt.Sprintf("%s %s:%s", book, chapter, verse)
	}

	// Handle more complex cases

	// Pattern: "BookName Chapter:Verse-Chapter:Verse" (spanning multiple chapters)
	// Example: "John 1:1-2:5"
	multiChapterRangeRegex := regexp.MustCompile(`^([1-3]?\s*[A-Za-z]+)\s+(\d+):(\d+)-(\d+):(\d+)$`)
	if multiChapterRangeRegex.MatchString(reference) {
		parts := multiChapterRangeRegex.FindStringSubmatch(reference)
		book := strings.TrimSpace(parts[1])
		startChapter := parts[2]
		startVerse := parts[3]
		endChapter := parts[4]
		endVerse := parts[5]

		// Convert to multiple single-chapter references
		if startChapter == endChapter {
			return fmt.Sprintf("%s %s:%s-%s", book, startChapter, startVerse, endVerse)
		}

		// For multi-chapter spans, return in proper format
		return fmt.Sprintf("%s %s:%s-%s:%s", book, startChapter, startVerse, endChapter, endVerse)
	}

	// Pattern: "BookName Chapter" (whole chapter)
	// Example: "John 3"
	wholeChapterRegex := regexp.MustCompile(`^([1-3]?\s*[A-Za-z]+)\s+(\d+)$`)
	if wholeChapterRegex.MatchString(reference) {
		parts := wholeChapterRegex.FindStringSubmatch(reference)
		book := strings.TrimSpace(parts[1])
		chapter := parts[2]
		// When just a chapter is provided, fetch the entire chapter (1-176)
		// The upper verse number is the actual maximum verse count (Psalm 119)
		return fmt.Sprintf("%s %s:1-176", book, chapter)
	}

	// If we can't normalize it, return the original
	return reference
}

// SplitReferences splits a reference string that may contain multiple references
// separated by commas or other delimiters
func SplitReferences(referenceString string) []string {
	// First check for comma-separated references
	if strings.Contains(referenceString, ",") {
		parts := strings.Split(referenceString, ",")
		var result []string
		for _, part := range parts {
			// Each part could be a single reference or a complex one
			subRefs := SplitMultiChapterReference(strings.TrimSpace(part))
			result = append(result, subRefs...)
		}
		return result
	}

	// If no commas, process as a potentially complex reference
	return SplitMultiChapterReference(referenceString)
}

// SplitMultiChapterReference splits a reference that spans multiple chapters
// into individual chapter references
func SplitMultiChapterReference(reference string) []string {
	// Handle chapter-only references like "John 3"
	wholeChapterRegex := regexp.MustCompile(`^([1-3]?\s*[A-Za-z]+)\s+(\d+)$`)
	if wholeChapterRegex.MatchString(reference) {
		parts := wholeChapterRegex.FindStringSubmatch(reference)
		book := strings.TrimSpace(parts[1])
		chapter, _ := strconv.Atoi(parts[2])
		// Use full chapter range (1-176, the max verses in any chapter)
		return []string{fmt.Sprintf("%s %d:1-176", book, chapter)}
	}

	// Check for potentially ambiguous formats first
	// This regex handles both formats like "Matthew 5:1-7" (verse range) and "Matthew 5:1-7:29" (chapter range)
	genericRangeRegex := regexp.MustCompile(`^([1-3]?\s*[A-Za-z]+)\s+(\d+):(\d+)-(\d+):?(\d*)$`)
	if genericRangeRegex.MatchString(reference) {
		parts := genericRangeRegex.FindStringSubmatch(reference)
		book := strings.TrimSpace(parts[1])
		startChapter, _ := strconv.Atoi(parts[2])
		startVerse, _ := strconv.Atoi(parts[3])
		endChapterOrVerse, _ := strconv.Atoi(parts[4])
		endVerse := 0

		// If we have a 5th capture group with digits, this is "Chapter:Verse-Chapter:Verse" format
		if parts[5] != "" {
			endVerse, _ = strconv.Atoi(parts[5])
			endChapter := endChapterOrVerse

			log.Printf("INFO: Parsing multi-chapter reference: %s %d:%d-%d:%d", book, startChapter, startVerse, endChapter, endVerse)

			if startChapter == endChapter {
				// Same chapter, just normalize
				return []string{fmt.Sprintf("%s %d:%d-%d", book, startChapter, startVerse, endVerse)}
			}

			// Create references for each chapter in the range
			var references []string

			// First chapter (from startVerse to end of chapter)
			references = append(references, fmt.Sprintf("%s %d:%d-176", book, startChapter, startVerse))

			// Middle chapters (whole chapters)
			for chapter := startChapter + 1; chapter < endChapter; chapter++ {
				references = append(references, fmt.Sprintf("%s %d:1-176", book, chapter))
			}

			// Last chapter (from beginning to endVerse)
			references = append(references, fmt.Sprintf("%s %d:1-%d", book, endChapter, endVerse))

			return references
		} else {
			// This is "Chapter:Verse-Verse" format (same chapter, verse range)
			log.Printf("INFO: Parsing same-chapter verse range: %s %d:%d-%d", book, startChapter, startVerse, endChapterOrVerse)
			return []string{fmt.Sprintf("%s %d:%d-%d", book, startChapter, startVerse, endChapterOrVerse)}
		}
	}

	// Handle multi-book references (e.g., "1 John 5:18-2 John 1:3")
	// This is complex and rarely standardized, so we'd need custom logic
	multiBookRegex := regexp.MustCompile(`([1-3]?\s*[A-Za-z]+[^\d]+\d+:\d+(?:-\d+)?)\s*[-—–]\s*([1-3]?\s*[A-Za-z]+\s+\d+:\d+(?:-\d+)?)`)
	simpleRefRegex := regexp.MustCompile(`^([1-3]?\s*[A-Za-z]+)\s+(\d+):(\d+)$`)

	if multiBookRegex.MatchString(reference) {
		matches := multiBookRegex.FindStringSubmatch(reference)
		if len(matches) >= 3 { // Full match + 2 capture groups
			firstRefStr := strings.TrimSpace(matches[1])
			secondRefStr := strings.TrimSpace(matches[2])
			log.Printf("INFO: Splitting multi-book reference '%s' into '%s' and '%s'", reference, firstRefStr, secondRefStr)

			result := []string{}

			// Parse the first reference part (e.g., "1 John 5:18")
			firstParts := simpleRefRegex.FindStringSubmatch(firstRefStr)
			if len(firstParts) == 4 {
				book := strings.TrimSpace(firstParts[1])
				chapter, _ := strconv.Atoi(firstParts[2])
				verse, _ := strconv.Atoi(firstParts[3])
				// Create range from start verse to end of chapter
				result = append(result, fmt.Sprintf("%s %d:%d-176", book, chapter, verse))
			} else {
				// If the first part is more complex (e.g., already a range), handle it recursively
				// This might need refinement depending on desired behavior for ranges like "Gen 1:1-5 - Ex 2:3"
				log.Printf("WARN: Multi-book start reference '%s' is not simple C:V, attempting recursive split", firstRefStr)
				result = append(result, SplitMultiChapterReference(firstRefStr)...)
			}

			// TODO: Handle intermediate books/chapters if necessary. Currently assumes direct adjacency or ignores gaps.

			// Parse the second reference part (e.g., "2 John 1:3")
			secondParts := simpleRefRegex.FindStringSubmatch(secondRefStr)
			if len(secondParts) == 4 {
				book := strings.TrimSpace(secondParts[1])
				chapter, _ := strconv.Atoi(secondParts[2])
				verse, _ := strconv.Atoi(secondParts[3])
				// Create range from start of chapter to end verse
				result = append(result, fmt.Sprintf("%s %d:1-%d", book, chapter, verse))
			} else {
				// If the second part is more complex, handle it recursively
				log.Printf("WARN: Multi-book end reference '%s' is not simple C:V, attempting recursive split", secondRefStr)
				result = append(result, SplitMultiChapterReference(secondRefStr)...)
			}

			return result
		}
	}

	// If not a recognized multi-part reference, normalize and return as single item
	return []string{NormalizeBibleReference(reference)}
}

// --- Existing code in bible_reference.go above this line ---

// IsValidReference checks if a given reference string can be successfully parsed
// by SplitMultiChapterReference into one or more valid, normalized reference
// formats (Book Ch:V or Book Ch:V-V) that match structural expectations.
func IsValidReference(reference string) (bool, error) {
	// Define a regex for what a final, usable reference segment should look like.
	// Allows for books with numbers (e.g., "1 John"), spaces, and standard Ch:V or Ch:V-V formats.
	// Ensures chapters and verses are numeric.
	// Book name part is flexible but requires at least one letter.
	// Example matches: "John 3:16", "1 Corinthians 13:1-13", "Psalm 119:176"
	validRefSegmentRegex := regexp.MustCompile(`^([1-3]?\s*[A-Za-z]+(?:\s+[A-Za-z]+)*)\s+(\d+):(\d+)(?:-(\d+))?$`)

	// Use the existing SplitMultiChapterReference to break down complex refs
	splitRefs := SplitMultiChapterReference(reference)

	if len(splitRefs) == 0 {
		// This indicates an issue, possibly with the input reference itself before splitting.
		return false, fmt.Errorf("reference '%s' resulted in an empty split, likely invalid input", reference)
	}

	for _, refSegment := range splitRefs {
		trimmedRef := strings.TrimSpace(refSegment)

		// Check 1: Does the segment match the expected structure?
		if !validRefSegmentRegex.MatchString(trimmedRef) {
			// Provide a more specific error based on common issues
			if !strings.Contains(trimmedRef, ":") && len(strings.Fields(trimmedRef)) == 1 {
				// Likely just a book name, which isn't a full reference
				return false, fmt.Errorf("reference part '%s' is incomplete (missing chapter/verse)", trimmedRef)
			}
			if strings.HasSuffix(trimmedRef, ":") || strings.HasSuffix(trimmedRef, "-") {
				// Missing verse number or end verse number
				return false, fmt.Errorf("reference part '%s' is incomplete (missing verse number)", trimmedRef)
			}
			// Generic structural failure
			return false, fmt.Errorf("reference part '%s' does not match expected format 'Book Chapter:Verse' or 'Book Chapter:StartVerse-EndVerse'", trimmedRef)
		}

		// Check 2: If it's a range, is startVerse <= endVerse?
		matches := validRefSegmentRegex.FindStringSubmatch(trimmedRef)
		// Index 4 corresponds to the optional endVerse capture group [(?:-(\d+))?](cci:1://file:///Users/Shared/work/deb/backend/internal/util/bible_reference_test.go:8:0-83:1)
		if len(matches) > 4 && matches[4] != "" { // It's a range
			// We know these are digits from the regex match, so ignore errors
			startVerse, _ := strconv.Atoi(matches[3])
			endVerse, _ := strconv.Atoi(matches[4])
			if startVerse > endVerse {
				return false, fmt.Errorf("reference part '%s' has start verse (%d) greater than end verse (%d)", trimmedRef, startVerse, endVerse)
			}
		}
	}

	// If all segments passed the checks
	return true, nil
}
