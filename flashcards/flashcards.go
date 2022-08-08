// Copyright (c) 2022 Levi Gruspe
// License: GNU AGPLv3 or later

package flashcards

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"sync"

	"github.com/lggruspe/polycloze/database"
	"github.com/lggruspe/polycloze/sentences"
	"github.com/lggruspe/polycloze/text"
	"github.com/lggruspe/polycloze/translator"
	"github.com/lggruspe/polycloze/word_scheduler"
)

// Different from sentences.Sentence
type Sentence struct {
	ID        int      `json:"id"`    // id in database
	Parts     []string `json:"parts"` // Odd-numbered parts are blanks
	TatoebaID int64    `json:"tatoebaID,omitempty"`
}

type Item struct {
	Sentence    Sentence               `json:"sentence"`
	Translation translator.Translation `json:"translation"`
}

type ItemGenerator struct {
	db       *sql.DB
	courseDB string // to be attached
}

func (ig ItemGenerator) Session() (*database.Session, error) {
	return database.NewSession(ig.db, ig.courseDB)
}

// Creates an ItemGenerator.
func NewItemGenerator(db *sql.DB, courseDB string) ItemGenerator {
	return ItemGenerator{
		db:       db,
		courseDB: courseDB,
	}
}

func getParts(tokens []string, word string) []string {
	var indices []int
	for i, token := range tokens {
		if text.Casefold(token) == text.Casefold(word) {
			indices = append(indices, i)
		}
	}

	if len(indices) == 0 {
		message := fmt.Sprintf("Python casefold different from golang x case folder: %s, %v", word, tokens)
		panic(message)
	}

	index := indices[rand.Intn(len(indices))]
	return []string{
		strings.Join(tokens[:index], ""),
		tokens[index],
		strings.Join(tokens[index+1:], ""),
	}
}

func (ig ItemGenerator) GenerateWords(n int, pred func(word string) bool) ([]string, error) {
	// NOTE Can't goroutines share this object?
	session, err := ig.Session()
	if err != nil {
		return nil, err
	}
	defer session.Close()
	return word_scheduler.GetWordsWith(session, n, pred)
}

func (ig ItemGenerator) GenerateItem(word string) (Item, error) {
	var item Item

	// NOTE Can't goroutines share this object?
	session, err := ig.Session()
	if err != nil {
		return item, err
	}
	defer session.Close()

	sentence, err := sentences.PickSentence(session, word, word_scheduler.PreferredDifficulty(session))
	if err != nil {
		return item, err
	}

	translation, err := translator.Translate(session, *sentence)
	if err != nil {
		panic(fmt.Errorf("could not translate sentence (%v): %v", sentence, err.Error()))
	}
	return Item{
		Translation: translation,
		Sentence: Sentence{
			ID:        sentence.ID,
			Parts:     getParts(sentence.Tokens, word),
			TatoebaID: sentence.TatoebaID,
		},
	}, nil
}

// Creates a cloze item for each word.
func (ig ItemGenerator) GenerateItems(words []string) []Item {
	ch := make(chan Item, len(words))
	ig.GenerateItemsIntoChannel(ch, words)
	close(ch)

	items := make([]Item, 0)
	for item := range ch {
		items = append(items, item)
	}
	return items
}

func (ig ItemGenerator) GenerateItemsIntoChannel(ch chan Item, words []string) {
	var wg sync.WaitGroup
	wg.Add(len(words))
	for _, word := range words {
		go func(ig *ItemGenerator, word string) {
			defer wg.Done()
			item, err := ig.GenerateItem(word)
			if err == nil {
				ch <- item
			}
		}(&ig, word)
	}
	wg.Wait()
}
