package account

import (
	"sync"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/domains/account"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

// AccountManager manages multiple WhatsApp clients for different accounts
type AccountManager struct {
	clients map[string]*whatsmeow.Client
	dbs     map[string]*sqlstore.Container
	keysDBs map[string]*sqlstore.Container
	mu      sync.RWMutex
}

// NewAccountManager creates a new account manager
func NewAccountManager() account.IAccountManager {
	return &AccountManager{
		clients: make(map[string]*whatsmeow.Client),
		dbs:     make(map[string]*sqlstore.Container),
		keysDBs: make(map[string]*sqlstore.Container),
	}
}

// GetClient returns the WhatsApp client for the given account ID
func (m *AccountManager) GetClient(accountID string) *whatsmeow.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clients[accountID]
}

// SetClient sets the WhatsApp client for the given account ID
func (m *AccountManager) SetClient(accountID string, client *whatsmeow.Client, db *sqlstore.Container) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[accountID] = client
	m.dbs[accountID] = db
}

// RemoveClient removes the WhatsApp client for the given account ID
func (m *AccountManager) RemoveClient(accountID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close the client connection if it exists
	if client, exists := m.clients[accountID]; exists && client != nil {
		client.Disconnect()
	}

	// Close the database if it exists
	if db, exists := m.dbs[accountID]; exists && db != nil {
		db.Close()
	}

	// Close the keys database if it exists
	if keysDB, exists := m.keysDBs[accountID]; exists && keysDB != nil {
		keysDB.Close()
	}

	delete(m.clients, accountID)
	delete(m.dbs, accountID)
	delete(m.keysDBs, accountID)
}

// ListClients returns all registered clients
func (m *AccountManager) ListClients() map[string]*whatsmeow.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy to avoid race conditions
	clients := make(map[string]*whatsmeow.Client)
	for k, v := range m.clients {
		clients[k] = v
	}
	return clients
}

// GetDB returns the database container for the given account ID
func (m *AccountManager) GetDB(accountID string) *sqlstore.Container {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dbs[accountID]
}

// GetKeysDB returns the keys database container for the given account ID
func (m *AccountManager) GetKeysDB(accountID string) *sqlstore.Container {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.keysDBs[accountID]
}

// SetKeysDB sets the keys database for the given account ID
func (m *AccountManager) SetKeysDB(accountID string, keysDB *sqlstore.Container) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keysDBs[accountID] = keysDB
}
