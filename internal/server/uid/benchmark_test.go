package uid_test

import (
	"fmt"
	"testing"
	"io"
	"log"

	"raven/internal/models"
	"raven/internal/server"
)

func BenchmarkUIDCopy_MultiMessage(b *testing.B) {
	log.SetOutput(io.Discard)

	t := &testing.T{}
	srv := server.SetupTestServerSimple(t)
	database := server.GetDatabaseFromServer(srv)

	userID := server.CreateTestUser(t, database, "benchuser")

	numMessages := 500

	for i := 1; i <= numMessages; i++ {
		server.InsertTestMail(t, database, "benchuser", fmt.Sprintf("Subject %d", i), "sender@test.com", "benchuser@test.com", "INBOX")
	}

	uidSeq := fmt.Sprintf("1:%d", numMessages)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		conn := server.NewMockConn()

		state := &models.ClientState{
			Authenticated:     true,
			UserID:            userID,
			Username:          "benchuser",
			Email:             "benchuser@localhost",
			// Don't set SelectedFolder, rely on HandleSelect to set it
		}

		srv.HandleSelect(conn, "A001", []string{"SELECT", "INBOX"}, state)
		conn.ClearWriteBuffer()

		b.StartTimer()
		srv.HandleUID(conn, "A002", []string{"UID", "UID", "COPY", uidSeq, "Archive"}, state)
		b.StopTimer()
	}
}
