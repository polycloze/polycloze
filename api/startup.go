// Copyright (c) 2022 Levi Gruspe
// License: GNU AGPLv3 or later

package api

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/polycloze/polycloze/basedir"
	"github.com/polycloze/polycloze/database"
)

// Version string of course files.
var dataVersion string

type Language struct {
	Code  string `json:"code"` // ISO 639-3
	Name  string `json:"name"` // in english
	BCP47 string `json:"bcp47"`
}

// For sorting languages by code.
type ByCode []Language

func (a ByCode) Len() int {
	return len(a)
}

func (a ByCode) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByCode) Less(i, j int) bool {
	return a[i].Code < a[j].Code
}

// Look for installed languages and courses.
func Startup() {
	// Look for courses and languages.
	courses := findCourses()
	languages := findL1Languages(courses)
	if len(languages) <= 0 {
		log.Fatal("Couldn't find installed courses. Please visit https://github.com/polycloze/polycloze/tree/main/python")
	}

	// Set version string.
	versionFile := filepath.Join(basedir.DataDir, "version.txt")
	version, err := os.ReadFile(versionFile)
	if err != nil {
		log.Fatal("Couldn't set version number.")
	}
	dataVersion = string(version)

	// Generate courses.json.
	coursesJSON := filepath.Join(basedir.StateDir, "courses.json")
	err = writeJSON(coursesJSON, map[string][]Course{
		"courses": courses,
	})
	if err != nil {
		log.Fatal("failed to write courses.json:", err)
	}

	languagesJSON := filepath.Join(basedir.StateDir, "languages.json")
	err = writeJSON(languagesJSON, map[string][]Language{
		"languages": languages,
	})
	if err != nil {
		log.Fatal("failed to write languages.json:", err)
	}

	// Compute hashes of static files.
	_ = computeHashes()
}

// Input: path to course db file.
func getCourseInfo(path string) (Course, error) {
	var course Course

	db, err := database.Open(path)
	if err != nil {
		return course, fmt.Errorf("could not open db to get course info: %w", err)
	}
	defer db.Close()

	query := `select id, code, name, bcp47 from language`
	rows, err := db.Query(query)
	if err != nil {
		return course, fmt.Errorf("could not get course info: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, code, name, bcp47 string
		if err := rows.Scan(&id, &code, &name, &bcp47); err != nil {
			return course, err
		}

		switch id {
		case "l1":
			course.L1.Code = code
			course.L1.Name = name
			course.L1.BCP47 = bcp47
		case "l2":
			course.L2.Code = code
			course.L2.Name = name
			course.L2.BCP47 = bcp47
		}
	}

	if course.L1.Code == "" || course.L2.Code == "" {
		return course, fmt.Errorf("invalid course database: %s\n", path)
	}
	return course, nil
}

// Look for installed courses in data directory.
func findCourses() []Course {
	var courses []Course
	matches, _ := filepath.Glob(filepath.Join(basedir.DataDir, "courses", "*.db"))
	for _, match := range matches {
		course, err := getCourseInfo(match)
		if err == nil {
			courses = append(courses, course)
		}
	}
	return courses
}

func findL1Languages(courses []Course) []Language {
	languages := make(map[Language]bool)
	for _, course := range courses {
		languages[course.L1] = true
	}

	var result []Language
	for language := range languages {
		result = append(result, language)
	}
	return result
}
