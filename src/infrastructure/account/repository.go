package account

import (
	"database/sql"
	"fmt"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/domains/account"
	_ "github.com/mattn/go-sqlite3"
)

// AccountRepository implements account.IAccountRepository
type AccountRepository struct {
	db *sql.DB
}

// NewAccountRepository creates a new account repository
func NewAccountRepository(dbPath string) (account.IAccountRepository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &AccountRepository{db: db}
	if err := repo.init(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return repo, nil
}

// init creates the necessary tables
func (r *AccountRepository) init() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS accounts (
			id TEXT PRIMARY KEY,
			status TEXT NOT NULL DEFAULT 'disconnected',
			phone_number TEXT,
			device_id TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_connected DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS account_webhooks (
			account_id TEXT PRIMARY KEY,
			url TEXT NOT NULL,
			secret TEXT,
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status)`,
	}

	for _, query := range queries {
		if _, err := r.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

// CreateAccount creates a new account
func (r *AccountRepository) CreateAccount(acc *account.Account) error {
	query := `INSERT INTO accounts (id, status, phone_number, device_id, created_at, last_connected)
			  VALUES (?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		acc.ID,
		acc.Status,
		acc.PhoneNumber,
		acc.DeviceID,
		acc.CreatedAt,
		acc.LastConnected,
	)

	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	return nil
}

// GetAccount retrieves an account by ID
func (r *AccountRepository) GetAccount(accountID string) (*account.Account, error) {
	query := `SELECT id, status, phone_number, device_id, created_at, last_connected
			  FROM accounts WHERE id = ?`

	acc := &account.Account{}
	var phoneNumber, deviceID sql.NullString
	var lastConnected sql.NullTime

	err := r.db.QueryRow(query, accountID).Scan(
		&acc.ID,
		&acc.Status,
		&phoneNumber,
		&deviceID,
		&acc.CreatedAt,
		&lastConnected,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("account not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if phoneNumber.Valid {
		acc.PhoneNumber = phoneNumber.String
	}
	if deviceID.Valid {
		acc.DeviceID = deviceID.String
	}
	if lastConnected.Valid {
		acc.LastConnected = lastConnected.Time
	}

	return acc, nil
}

// UpdateAccount updates an existing account
func (r *AccountRepository) UpdateAccount(acc *account.Account) error {
	query := `UPDATE accounts
			  SET status = ?, phone_number = ?, device_id = ?, last_connected = ?
			  WHERE id = ?`

	result, err := r.db.Exec(query,
		acc.Status,
		acc.PhoneNumber,
		acc.DeviceID,
		acc.LastConnected,
		acc.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("account not found")
	}

	return nil
}

// DeleteAccount deletes an account by ID
func (r *AccountRepository) DeleteAccount(accountID string) error {
	query := `DELETE FROM accounts WHERE id = ?`

	result, err := r.db.Exec(query, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("account not found")
	}

	return nil
}

// ListAccounts retrieves all accounts
func (r *AccountRepository) ListAccounts() ([]*account.Account, error) {
	query := `SELECT id, status, phone_number, device_id, created_at, last_connected
			  FROM accounts ORDER BY created_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []*account.Account

	for rows.Next() {
		acc := &account.Account{}
		var phoneNumber, deviceID sql.NullString
		var lastConnected sql.NullTime

		err := rows.Scan(
			&acc.ID,
			&acc.Status,
			&phoneNumber,
			&deviceID,
			&acc.CreatedAt,
			&lastConnected,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}

		if phoneNumber.Valid {
			acc.PhoneNumber = phoneNumber.String
		}
		if deviceID.Valid {
			acc.DeviceID = deviceID.String
		}
		if lastConnected.Valid {
			acc.LastConnected = lastConnected.Time
		}

		accounts = append(accounts, acc)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return accounts, nil
}

// SetWebhook sets the webhook for an account
func (r *AccountRepository) SetWebhook(accountID string, webhookURL string, secret string) error {
	query := `INSERT INTO account_webhooks (account_id, url, secret)
			  VALUES (?, ?, ?)
			  ON CONFLICT(account_id)
			  DO UPDATE SET url = ?, secret = ?`

	_, err := r.db.Exec(query, accountID, webhookURL, secret, webhookURL, secret)
	if err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}

	return nil
}

// GetWebhook retrieves the webhook for an account
func (r *AccountRepository) GetWebhook(accountID string) (*account.WebhookInfo, error) {
	query := `SELECT url, secret FROM account_webhooks WHERE account_id = ?`

	webhook := &account.WebhookInfo{}
	var secret sql.NullString

	err := r.db.QueryRow(query, accountID).Scan(&webhook.URL, &secret)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("webhook not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}

	if secret.Valid {
		webhook.Secret = secret.String
	}

	return webhook, nil
}

// Close closes the database connection
func (r *AccountRepository) Close() error {
	return r.db.Close()
}
