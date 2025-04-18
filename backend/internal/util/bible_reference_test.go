package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitMultiChapterReference(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedRefs []string
		// expectError // Function doesn't return error, handles via normalization
	}{
		{
			name:         "Same chapter range",
			input:        "Matthew 5:1-7",
			expectedRefs: []string{"Matthew 5:1-7"},
		},
		{
			name:         "Multi-chapter range",
			input:        "Matthew 5:1-7:29",
			expectedRefs: []string{"Matthew 5:1-176", "Matthew 6:1-176", "Matthew 7:1-29"},
		},
		{
			name:         "Single verse",
			input:        "John 3:16",
			expectedRefs: []string{"John 3:16"}, // Normalized
		},
		{
			name:         "Book with number and space",
			input:        "1 John 1:1-5",
			expectedRefs: []string{"1 John 1:1-5"},
		},
		{
			name:         "Book with number and space multi-chapter",
			input:        "2 Timothy 1:1-2:2",
			expectedRefs: []string{"2 Timothy 1:1-176", "2 Timothy 2:1-2"},
		},
		{
			name:         "Invalid format - missing verse",
			input:        "Genesis 1-",
			expectedRefs: []string{"Genesis 1-"}, // Normalized by NormalizeBibleReference
		},
		{
			name:         "Invalid format - just book",
			input:        "Genesis",
			expectedRefs: []string{"Genesis"}, // Normalized by NormalizeBibleReference
		},
		{
			name:         "Philemon single chapter range", // Book with only one chapter
			input:        "Philemon 1:5-10",
			expectedRefs: []string{"Philemon 1:5-10"},
		},
		{
			name:         "Chapter only reference",
			input:        "John 3",
			expectedRefs: []string{"John 3:1-176"},
		},
		{
			name:         "Multi-book reference (example)", // Assuming SplitMultiChapterReference handles recursive calls correctly
			input:        "1 John 5:18-2 John 1:3",
			expectedRefs: []string{"1 John 5:18-176", "2 John 1:1-3"}, // Depends on NormalizeBibleReference and the recursive split
		},
		// Add more test cases as needed
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualRefs := SplitMultiChapterReference(tc.input)
			assert.Equal(t, tc.expectedRefs, actualRefs)

			// No error checking needed here as the function doesn't return one
			// if tc.expectError {
			// 	assert.Error(t, err)
			// } else {
			// 	assert.NoError(t, err)
			// 	assert.Equal(t, tc.expectedStart, startRef)
			// 	assert.Equal(t, tc.expectedEnd, endRef)
			// }
		})
	}
}

func TestIsValidReference(t *testing.T) {
	tests := []struct {
		name                string
		input               string
		expectValid         bool
		expectErrorContains string // Substring of the expected error message if expectValid is false
	}{
		{
			name:        "Valid single verse",
			input:       "John 3:16",
			expectValid: true,
		},
		{
			name:        "Valid verse range",
			input:       "Romans 8:38-39",
			expectValid: true,
		},
		{
			name:        "Valid numbered book",
			input:       "1 Corinthians 13:4-8",
			expectValid: true,
		},
		{
			name:        "Valid whole chapter reference (normalized)",
			input:       "Psalm 23", // Normalizes to Psalm 23:1-176
			expectValid: true,
		},
		{
			name:        "Valid multi-chapter reference (normalized)",
			input:       "Matthew 5:1-7:29", // Splits into Matt 5:1-176, Matt 6:1-176, Matt 7:1-29
			expectValid: true,
		},
		{
			name:        "Valid Philemon single chapter range",
			input:       "Philemon 1:5-10",
			expectValid: true,
		},
		{
			name:                "Invalid - incomplete verse",
			input:               "Genesis 1:",
			expectValid:         false,
			expectErrorContains: "incomplete (missing verse number)",
		},
		{
			name:                "Invalid - incomplete range",
			input:               "Exodus 20:1-",
			expectValid:         false,
			expectErrorContains: "incomplete (missing verse number)",
		},
		{
			name:                "Invalid - range start > end",
			input:               "John 11:35-30",
			expectValid:         false,
			expectErrorContains: "start verse (35) greater than end verse (30)",
		},
		{
			name:                "Invalid - non-numeric verse",
			input:               "John 3:abc",
			expectValid:         false,
			expectErrorContains: "does not match expected format",
		},
		{
			name:                "Invalid - wrong structure",
			input:               "John chapter 3 verse 16",
			expectValid:         false,
			expectErrorContains: "does not match expected format",
		},
		{
			name:                "Invalid - book name only",
			input:               "Revelation", // Normalizes to self, IsValidReference checks format
			expectValid:         false,
			expectErrorContains: "incomplete (missing chapter/verse)",
		},
		{
			name:                "Invalid - just number",
			input:               "123",
			expectValid:         false,
			expectErrorContains: "incomplete (missing chapter/verse)",
		},
		{
			name:                "Invalid - empty string",
			input:               "",
			expectValid:         false,
			expectErrorContains: "does not match expected format",
		},
		{
			name:                "Invalid - whitespace string",
			input:               "   ",
			expectValid:         false,
			expectErrorContains: "does not match expected format",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isValid, err := IsValidReference(tc.input)

			if tc.expectValid {
				assert.True(t, isValid, "Expected reference to be valid")
				assert.NoError(t, err, "Expected no error for valid reference")
			} else {
				assert.False(t, isValid, "Expected reference to be invalid")
				assert.Error(t, err, "Expected error for invalid reference")
				if err != nil && tc.expectErrorContains != "" {
					assert.Contains(t, err.Error(), tc.expectErrorContains, "Error message should contain specific text")
				}
			}
		})
	}
}
