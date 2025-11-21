package account

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

type IAccountUsecase interface {
	CreateAccount(ctx context.Context, accountID string) (response CreateAccountResponse, err error)
	DeleteAccount(ctx context.Context, accountID string) (err error)
	ListAccounts(ctx context.Context) (response []AccountInfo, err error)
	GetAccount(ctx context.Context, accountID string) (response AccountInfo, err error)
	LoginAccount(ctx context.Context, accountID string) (response LoginResponse, err error)
	LoginAccountWithCode(ctx context.Context, accountID string, phoneNumber string) (loginCode string, err error)
	LogoutAccount(ctx context.Context, accountID string) (err error)
	ReconnectAccount(ctx context.Context, accountID string) (err error)
	SetAccountWebhook(ctx context.Context, accountID string, webhookURL string, secret string) (err error)
	GetAccountWebhook(ctx context.Context, accountID string) (webhook WebhookInfo, err error)
}

type IAccountRepository interface {
	CreateAccount(account *Account) error
	GetAccount(accountID string) (*Account, error)
	UpdateAccount(account *Account) error
	DeleteAccount(accountID string) error
	ListAccounts() ([]*Account, error)
	SetWebhook(accountID string, webhookURL string, secret string) error
	GetWebhook(accountID string) (*WebhookInfo, error)
}

type IAccountManager interface {
	GetClient(accountID string) *whatsmeow.Client
	SetClient(accountID string, client *whatsmeow.Client, db *sqlstore.Container)
	RemoveClient(accountID string)
	ListClients() map[string]*whatsmeow.Client
	GetDB(accountID string) *sqlstore.Container
	GetKeysDB(accountID string) *sqlstore.Container
	SetKeysDB(accountID string, keysDB *sqlstore.Container)
}

// Response structs
type CreateAccountResponse struct {
	AccountID string `json:"account_id"`
	Message   string `json:"message"`
}

type AccountInfo struct {
	AccountID     string    `json:"account_id"`
	Status        string    `json:"status"`
	IsConnected   bool      `json:"is_connected"`
	IsLoggedIn    bool      `json:"is_logged_in"`
	DeviceID      string    `json:"device_id,omitempty"`
	PhoneNumber   string    `json:"phone_number,omitempty"`
	WebhookURL    string    `json:"webhook_url,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	LastConnected time.Time `json:"last_connected,omitempty"`
}

type LoginResponse struct {
	ImagePath string        `json:"image_path"`
	Duration  time.Duration `json:"duration"`
	Code      string        `json:"code"`
}

type WebhookInfo struct {
	URL    string `json:"url"`
	Secret string `json:"secret"`
}

// Domain models
type Account struct {
	ID            string    `json:"id" db:"id"`
	Status        string    `json:"status" db:"status"`
	PhoneNumber   string    `json:"phone_number" db:"phone_number"`
	DeviceID      string    `json:"device_id" db:"device_id"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	LastConnected time.Time `json:"last_connected" db:"last_connected"`
}

const (
	StatusDisconnected = "disconnected"
	StatusConnected    = "connected"
	StatusLoggedIn     = "logged_in"
)
