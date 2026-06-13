package models

import (
	"testing"
)

func TestClientState_DefaultValues(t *testing.T) {
	state := &ClientState{}

	if state.Authenticated != false {
		t.Errorf("Expected Authenticated to be false, got %v", state.Authenticated)
	}
	if state.SelectedFolder != "" {
		t.Errorf("Expected SelectedFolder to be empty, got %v", state.SelectedFolder)
	}
	if state.SelectedMailboxID != 0 {
		t.Errorf("Expected SelectedMailboxID to be 0, got %v", state.SelectedMailboxID)
	}
	if state.Conn != nil {
		t.Errorf("Expected Conn to be nil, got %v", state.Conn)
	}
	if state.Username != "" {
		t.Errorf("Expected Username to be empty, got %v", state.Username)
	}
	if state.Email != "" {
		t.Errorf("Expected Email to be empty, got %v", state.Email)
	}
	if state.UserID != 0 {
		t.Errorf("Expected UserID to be 0, got %v", state.UserID)
	}
	if state.DomainID != 0 {
		t.Errorf("Expected DomainID to be 0, got %v", state.DomainID)
	}
	if state.LastMessageCount != 0 {
		t.Errorf("Expected LastMessageCount to be 0, got %v", state.LastMessageCount)
	}
	if state.LastRecentCount != 0 {
		t.Errorf("Expected LastRecentCount to be 0, got %v", state.LastRecentCount)
	}
	if state.UIDValidity != 0 {
		t.Errorf("Expected UIDValidity to be 0, got %v", state.UIDValidity)
	}
	if state.UIDNext != 0 {
		t.Errorf("Expected UIDNext to be 0, got %v", state.UIDNext)
	}
}

func TestClientState_Updates(t *testing.T) {
	state := &ClientState{}

	// Test authentication
	state.Authenticated = true
	state.Username = "user"
	state.Email = "user@example.com"

	if !state.Authenticated {
		t.Errorf("Expected Authenticated to be true")
	}
	if state.Username != "user" {
		t.Errorf("Expected Username to be 'user', got %v", state.Username)
	}
	if state.Email != "user@example.com" {
		t.Errorf("Expected Email to be 'user@example.com', got %v", state.Email)
	}

	// Test mailbox selection
	state.SelectedFolder = "INBOX"
	state.SelectedMailboxID = 42
	state.LastMessageCount = 100
	state.LastRecentCount = 5
	state.UIDValidity = 12345
	state.UIDNext = 101

	if state.SelectedFolder != "INBOX" {
		t.Errorf("Expected SelectedFolder to be 'INBOX', got %v", state.SelectedFolder)
	}
	if state.SelectedMailboxID != 42 {
		t.Errorf("Expected SelectedMailboxID to be 42, got %v", state.SelectedMailboxID)
	}
	if state.LastMessageCount != 100 {
		t.Errorf("Expected LastMessageCount to be 100, got %v", state.LastMessageCount)
	}
	if state.LastRecentCount != 5 {
		t.Errorf("Expected LastRecentCount to be 5, got %v", state.LastRecentCount)
	}
	if state.UIDValidity != 12345 {
		t.Errorf("Expected UIDValidity to be 12345, got %v", state.UIDValidity)
	}
	if state.UIDNext != 101 {
		t.Errorf("Expected UIDNext to be 101, got %v", state.UIDNext)
	}
}
