// Copyright (c) 2022 Levi Gruspe
// License: GNU AGPLv3 or later

package flashcards

import (
	"context"
	"testing"

	"github.com/polycloze/polycloze/basedir"
	"github.com/polycloze/polycloze/database"
)

func pred(_ string) bool {
	return true
}

func BenchmarkGetFlashcards(b *testing.B) {
	db, err := database.OpenReviewDB(":memory:")
	if err != nil {
		b.Fatal("expected err to be nil:", err)
	}
	defer db.Close()

	hook := database.AttachCourse(basedir.Course("eng", "deu"))
	con, err := database.NewConnection(db, context.Background(), hook)
	if err != nil {
		b.Fatal("expected err to be nil:", err)
	}

	for i := 0; i < b.N; i++ {
		Get(con, 10, pred)
	}
}
