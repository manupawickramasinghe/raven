package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DBManager manages database connections for shared and per-user databases
type DBManager struct {
	basePath       string
	sharedDB       *sql.DB
	userDBCache    map[string]*sql.DB
	userDBLastUsed map[string]time.Time
	cacheMutex     sync.RWMutex
	stopCleanup    chan struct{}
	closeOnce      sync.Once
}

// NewDBManager creates a new database manager
func NewDBManager(basePath string) (*DBManager, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0750); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %v", err)
	}

	manager := &DBManager{
		basePath:       basePath,
		userDBCache:    make(map[string]*sql.DB),
		userDBLastUsed: make(map[string]time.Time),
		stopCleanup:    make(chan struct{}),
	}

	// Initialize shared database
	if err := manager.initSharedDB(); err != nil {
		return nil, fmt.Errorf("failed to initialize shared database: %v", err)
	}

	// Start background cleanup of idle connections
	go manager.cleanupLoop()

	return manager, nil
}

// GetSharedDB returns the shared database connection
func (m *DBManager) GetSharedDB() *sql.DB {
	return m.sharedDB
}

// GetUserDB returns a database connection for a specific user identified by email
func (m *DBManager) GetUserDB(email string) (*sql.DB, error) {
	// Check cache first
	m.cacheMutex.RLock()
	db, exists := m.userDBCache[email]
	m.cacheMutex.RUnlock()

	if exists {
		m.cacheMutex.Lock()
		m.userDBLastUsed[email] = time.Now()
		m.cacheMutex.Unlock()
		return db, nil
	}

	// Create or open user database
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	// Double-check after acquiring write lock
	if db, exists := m.userDBCache[email]; exists {
		m.userDBLastUsed[email] = time.Now()
		return db, nil
	}

	dbPath := m.getUserDBPath(email)

	// Check if database file exists
	exists := false
	if _, err := os.Stat(dbPath); err == nil {
		exists = true
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open user database: %v", err)
	}

	// Enable foreign key constraints
	if _, err = db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %v", err)
	}

	// Initialize schema if this is a new database
	if !exists {
		if err := m.initUserDB(db); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to initialize user database: %v", err)
		}
	}

	// Cache the connection
	m.userDBCache[email] = db
	m.userDBLastUsed[email] = time.Now()

	return db, nil
}

// initSharedDB initializes the shared database
func (m *DBManager) initSharedDB() error {
	sharedPath := filepath.Join(m.basePath, "shared.db")

	db, err := sql.Open("sqlite3", sharedPath)
	if err != nil {
		return err
	}

	// Enable foreign key constraints
	if _, err = db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return err
	}

	// Create blobs table in shared database for deduplication across all users
	if err := createBlobsTable(db); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to create blobs table: %v", err)
	}

	// Create shared database indexes
	if err := createSharedIndexes(db); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to create shared indexes: %v", err)
	}

	m.sharedDB = db
	return nil
}

// initUserDB initializes a per-user database
func (m *DBManager) initUserDB(db *sql.DB) error {
	// Create user tables
	// Note: blobs table is now in shared database for cross-user deduplication

	if err := createMailboxesTablePerUser(db); err != nil {
		return fmt.Errorf("failed to create mailboxes table: %v", err)
	}

	if err := createAliasesTablePerUser(db); err != nil {
		return fmt.Errorf("failed to create aliases table: %v", err)
	}

	if err := createMessagesTable(db); err != nil {
		return fmt.Errorf("failed to create messages table: %v", err)
	}

	if err := createSubscriptionsTablePerUser(db); err != nil {
		return fmt.Errorf("failed to create subscriptions table: %v", err)
	}

	if err := createAddressesTable(db); err != nil {
		return fmt.Errorf("failed to create addresses table: %v", err)
	}

	if err := createMessagePartsTablePerUser(db); err != nil {
		return fmt.Errorf("failed to create message_parts table: %v", err)
	}

	if err := createDeliveriesTablePerUser(db); err != nil {
		return fmt.Errorf("failed to create deliveries table: %v", err)
	}

	if err := createMessageMailboxTable(db); err != nil {
		return fmt.Errorf("failed to create message_mailbox table: %v", err)
	}

	if err := createMessageHeadersTable(db); err != nil {
		return fmt.Errorf("failed to create message_headers table: %v", err)
	}

	if err := createOutboundQueueTablePerUser(db); err != nil {
		return fmt.Errorf("failed to create outbound_queue table: %v", err)
	}

	// Create user database indexes
	if err := createUserIndexes(db); err != nil {
		return fmt.Errorf("failed to create user indexes: %v", err)
	}

	// Create default mailboxes for this per-user database.
	if err := createDefaultMailboxes(db); err != nil {
		return fmt.Errorf("failed to create default mailboxes: %v", err)
	}

	return nil
}

// getUserDBPath returns the file path for a user's database
func (m *DBManager) getUserDBPath(email string) string {
	if strings.HasSuffix(email, ".db") {
		// Preserve explicit mailbox identity paths (for example role_<name>@<domain>.db).
		return filepath.Join(m.basePath, email)
	}

	return filepath.Join(m.basePath, fmt.Sprintf("user_%s.db", email))
}

// Close closes all database connections
func (m *DBManager) Close() error {
	var lastErr error
	m.closeOnce.Do(func() {
		close(m.stopCleanup)

		// Close shared database
		if m.sharedDB != nil {
			if err := m.sharedDB.Close(); err != nil {
				lastErr = err
			}
		}

		// Close all user databases
		m.cacheMutex.Lock()
		defer m.cacheMutex.Unlock()

		for email, db := range m.userDBCache {
			if err := db.Close(); err != nil {
				lastErr = err
			}
			delete(m.userDBCache, email)
			delete(m.userDBLastUsed, email)
		}
	})

	return lastErr
}

func (m *DBManager) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupIdleConnections()
		case <-m.stopCleanup:
			return
		}
	}
}

func (m *DBManager) cleanupIdleConnections() {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	now := time.Now()
	idleTimeout := 15 * time.Minute

	for email, lastUsed := range m.userDBLastUsed {
		if now.Sub(lastUsed) > idleTimeout {
			if db, exists := m.userDBCache[email]; exists {
				_ = db.Close()
				delete(m.userDBCache, email)
				delete(m.userDBLastUsed, email)
			}
		}
	}
}
