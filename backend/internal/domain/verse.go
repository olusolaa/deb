package domain

import "fmt"

type BibleVerse struct {
	Book        string `json:"book"`
	Chapter     int    `json:"chapter"`
	VerseNumber int    `json:"verse"`
	Text        string `json:"text"`
	Reference   string `json:"reference"` // Combined reference like "John 3:16"
}

func (v *BibleVerse) GenerateReference() {
	v.Reference = fmt.Sprintf("%s %d:%d", v.Book, v.Chapter, v.VerseNumber)
}
