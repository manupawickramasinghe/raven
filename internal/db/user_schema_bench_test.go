package db

import (
	"fmt"
	"testing"
)

func BenchmarkRenameMailboxPerUser_WithManyChildren(b *testing.B) {
	db := setupTestDBPerUser(&testing.T{})
	defer func() { _ = db.Close() }()

	// Create parent
	_, _ = CreateMailboxPerUser(db, "Parent", "")

	// Create 1000 children
	for i := 0; i < 1000; i++ {
		_, err := CreateMailboxPerUser(db, fmt.Sprintf("Parent/Child%d", i), "")
		if err != nil {
			b.Fatalf("failed to create child: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		oldName := "Parent"
		newName := "NewParent"
		if i%2 != 0 {
			oldName = "NewParent"
			newName = "Parent"
		}

		err := RenameMailboxPerUser(db, oldName, newName)
		if err != nil {
			b.Fatalf("RenameMailboxPerUser failed: %v", err)
		}
	}
}

func BenchmarkRenameMailboxPerUser_WithManyChildren_Optimized(b *testing.B) {
	db := setupTestDBPerUser(&testing.T{})
	defer func() { _ = db.Close() }()

	// Create parent
	_, _ = CreateMailboxPerUser(db, "Parent", "")

	// Create 1000 children
	for i := 0; i < 1000; i++ {
		_, err := CreateMailboxPerUser(db, fmt.Sprintf("Parent/Child%d", i), "")
		if err != nil {
			b.Fatalf("failed to create child: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		oldName := "Parent"
		newName := "NewParent"
		if i%2 != 0 {
			oldName = "NewParent"
			newName = "Parent"
		}

		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			b.Fatalf("Begin failed: %v", err)
		}

		mailboxID, _ := GetMailboxByNamePerUser(db, oldName)
		_, err = tx.Exec("UPDATE mailboxes SET name = ? WHERE id = ?", newName, mailboxID)
		if err != nil {
			b.Fatalf("Rename parent failed: %v", err)
		}

		hierarchyPattern := oldName + "/%"
		_, err = tx.Exec("UPDATE mailboxes SET name = ? || SUBSTR(name, length(?) + 1) WHERE name LIKE ?", newName, oldName, hierarchyPattern)
		if err != nil {
			b.Fatalf("Rename children failed: %v", err)
		}

		_ = tx.Commit()
	}
}
