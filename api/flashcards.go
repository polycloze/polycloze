// Copyright (c) 2022 Levi Gruspe
// License: GNU AGPLv3 or later

package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/polycloze/polycloze/auth"
	"github.com/polycloze/polycloze/basedir"
	"github.com/polycloze/polycloze/database"
	"github.com/polycloze/polycloze/difficulty"
	"github.com/polycloze/polycloze/flashcards"
	"github.com/polycloze/polycloze/sessions"
	"github.com/polycloze/polycloze/text"
	"github.com/polycloze/polycloze/word_scheduler"
)

// Returns predicate to pass to item generator.
func excludeWords(words []string) func(string) bool {
	exclude := make(map[string]bool)
	for _, word := range words {
		exclude[text.Casefold(word)] = true
	}
	return func(word string) bool {
		_, found := exclude[text.Casefold(word)]
		return !found
	}
}

func handleFlashcards(w http.ResponseWriter, r *http.Request) {
	// Check request method and content type.
	if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "expected JSON body in POST request", http.StatusBadRequest)
		return
	}

	// Check if course exists.
	l1 := chi.URLParam(r, "l1")
	l2 := chi.URLParam(r, "l2")
	if !courseExists(l1, l2) {
		http.NotFound(w, r)
		return
	}

	// Sign in.
	db := auth.GetDB(r)
	s, err := sessions.ResumeSession(db, w, r)
	if err != nil || !s.IsSignedIn() {
		http.NotFound(w, r)
		return
	}

	// Open user's review DB.
	userID := s.Data["userID"].(int)
	db, err = database.OpenReviewDB(basedir.Review(userID, l1, l2))
	if err != nil {
		log.Println(fmt.Errorf("could not open review database (%v-%v): %w", l1, l2, err))
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Create database connection with access to review and course DB.
	hook := database.AttachCourse(basedir.Course(l1, l2))
	con, err := database.NewConnection(db, r.Context(), hook)
	if err != nil {
		log.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	defer con.Close()

	// Read request data.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		http.Error(w, "Could not read request.", http.StatusInternalServerError)
		return
	}

	var data FlashcardsRequest
	if err := parseJSON(w, body, &data); err != nil {
		return
	}

	// Save uploaded reviews and difficulty stats.
	if len(data.Reviews) > 0 {
		// Look for csrf token in request headers or in the request body.
		token := r.Header.Get("X-CSRF-Token")
		if token == "" {
			token = data.CSRFToken
		}

		// Check csrf token.
		if !sessions.CheckCSRFToken(s.ID, token) {
			http.Error(w, "Forbidden.", http.StatusForbidden)
			return
		}

		// Save review results.
		if err := word_scheduler.BulkSaveWords(con, data.Reviews, time.Now()); err != nil {
			log.Println(err)
			http.Error(w, "Something went wrong.", http.StatusInternalServerError)
			return
		}

		if data.Difficulty != nil {
			if err := difficulty.Update(con, *data.Difficulty); err != nil {
				log.Println(err)
				http.Error(w, "Something went wrong.", http.StatusInternalServerError)
				return
			}
		}
	}

	// Generate flashcards.
	items := flashcards.Get(con, data.Limit, excludeWords(data.Exclude))
	newDiff := difficulty.GetLatest(con)
	sendJSON(w, FlashcardsResponse{
		Items:      items,
		Difficulty: &newDiff,
	})
}
