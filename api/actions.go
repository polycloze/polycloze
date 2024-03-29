// Copyright (c) 2022 Levi Gruspe
// License: GNU AGPLv3 or later

package api

import (
	"io"
	"log"
	"net/http"

	"github.com/polycloze/polycloze/auth"
	"github.com/polycloze/polycloze/basedir"
	"github.com/polycloze/polycloze/database"
	"github.com/polycloze/polycloze/sessions"
)

func handleSetCourse(w http.ResponseWriter, r *http.Request) {
	// Check request method and content type.
	if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "expected JSON body in POST request", http.StatusBadRequest)
		return
	}

	// Sign in.
	db := auth.GetDB(r)
	s, err := sessions.ResumeSession(db, w, r)
	if err != nil || !s.IsSignedIn() {
		http.Error(w, "Unauthorized.", http.StatusUnauthorized)
		return
	}

	// Read request data.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		http.Error(w, "Could not read request.", http.StatusInternalServerError)
		return
	}

	var data SetCourseRequest
	if err := parseJSON(w, body, &data); err != nil {
		return
	}

	// Check if course exists.
	if !courseExists(data.L1Code, data.L2Code) {
		http.Error(w, "invalid course", http.StatusBadRequest)
		return
	}

	// Check csrf token.
	token := r.Header.Get("X-CSRF-Token")
	if !sessions.CheckCSRFToken(s.ID, token) {
		http.Error(w, "Forbidden.", http.StatusForbidden)
		return
	}

	// Open user data DB.
	userID := s.Data["userID"].(int)
	db, err = database.OpenUserDB(basedir.UserData(userID))
	if err != nil {
		log.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Set active course.
	if err := setActiveCourse(db, userID, data.L1Code, data.L2Code); err != nil {
		log.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	sendJSON(w, SetCourseResponse{
		Ok: true,
	})
}
