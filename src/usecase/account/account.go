package account

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainAccount "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/account"
	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	"github.com/sirupsen/logrus"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.mau.fi/whatsmeow/types/events"
)

type AccountUsecase struct {
	repo            domainAccount.IAccountRepository
	manager         domainAccount.IAccountManager
	chatStorageRepo domainChatStorage.IChatStorageRepository
}

func NewAccountUsecase(
	repo domainAccount.IAccountRepository,
	manager domainAccount.IAccountManager,
	chatStorageRepo domainChatStorage.IChatStorageRepository,
) domainAccount.IAccountUsecase {
	return &AccountUsecase{
		repo:            repo,
		manager:         manager,
		chatStorageRepo: chatStorageRepo,
	}
}

// CreateAccount creates a new account
func (u *AccountUsecase) CreateAccount(ctx context.Context, accountID string) (domainAccount.CreateAccountResponse, error) {
	// Check if account already exists
	if _, err := u.repo.GetAccount(accountID); err == nil {
		return domainAccount.CreateAccountResponse{}, fmt.Errorf("account already exists")
	}

	// Create new account
	acc := &domainAccount.Account{
		ID:        accountID,
		Status:    domainAccount.StatusDisconnected,
		CreatedAt: time.Now(),
	}

	if err := u.repo.CreateAccount(acc); err != nil {
		return domainAccount.CreateAccountResponse{}, fmt.Errorf("failed to create account: %w", err)
	}

	return domainAccount.CreateAccountResponse{
		AccountID: accountID,
		Message:   "Account created successfully",
	}, nil
}

// DeleteAccount deletes an account
func (u *AccountUsecase) DeleteAccount(ctx context.Context, accountID string) error {
	// Get account to verify it exists
	if _, err := u.repo.GetAccount(accountID); err != nil {
		return fmt.Errorf("account not found")
	}

	// Disconnect and remove client if exists
	if client := u.manager.GetClient(accountID); client != nil {
		client.Disconnect()

		// Clean up database files for this account
		if err := u.cleanupAccountDatabase(accountID); err != nil {
			logrus.Warnf("Failed to cleanup database for account %s: %v", accountID, err)
		}

		u.manager.RemoveClient(accountID)
	}

	// Delete from repository
	if err := u.repo.DeleteAccount(accountID); err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	// Clean up QR codes and other files
	u.cleanupAccountFiles(accountID)

	return nil
}

// ListAccounts lists all accounts
func (u *AccountUsecase) ListAccounts(ctx context.Context) ([]domainAccount.AccountInfo, error) {
	accounts, err := u.repo.ListAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	var result []domainAccount.AccountInfo
	for _, acc := range accounts {
		info := u.buildAccountInfo(acc)
		result = append(result, info)
	}

	return result, nil
}

// GetAccount gets account details
func (u *AccountUsecase) GetAccount(ctx context.Context, accountID string) (domainAccount.AccountInfo, error) {
	acc, err := u.repo.GetAccount(accountID)
	if err != nil {
		return domainAccount.AccountInfo{}, fmt.Errorf("account not found")
	}

	return u.buildAccountInfo(acc), nil
}

// LoginAccount logs in an account using QR code
func (u *AccountUsecase) LoginAccount(ctx context.Context, accountID string) (domainAccount.LoginResponse, error) {
	// Get account
	acc, err := u.repo.GetAccount(accountID)
	if err != nil {
		return domainAccount.LoginResponse{}, fmt.Errorf("account not found")
	}

	// Check if already logged in
	if client := u.manager.GetClient(accountID); client != nil && client.IsLoggedIn() {
		return domainAccount.LoginResponse{}, fmt.Errorf("account is already logged in")
	}

	// Initialize database for this account
	dbPath := u.getAccountDBPath(accountID)
	db, err := u.initAccountDatabase(ctx, dbPath)
	if err != nil {
		return domainAccount.LoginResponse{}, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize keys database if configured
	var keysDB *sqlstore.Container
	if config.DBKeysURI != "" {
		keysDBPath := u.getAccountKeysDBPath(accountID)
		keysDB, err = u.initAccountDatabase(ctx, keysDBPath)
		if err != nil {
			logrus.Warnf("Failed to initialize keys database: %v", err)
		} else {
			u.manager.SetKeysDB(accountID, keysDB)
		}
	}

	// Get or create device
	device, err := db.GetFirstDevice(ctx)
	if err != nil {
		return domainAccount.LoginResponse{}, fmt.Errorf("failed to get device: %w", err)
	}

	// Configure device properties
	osName := fmt.Sprintf("%s %s", config.AppOs, config.AppVersion)
	store.DeviceProps.PlatformType = &config.AppPlatform
	store.DeviceProps.Os = &osName

	// Configure encryption cache database if keysDB exists
	if keysDB != nil && device.ID != nil {
		innerStore := sqlstore.NewSQLStore(keysDB, *device.ID)
		device.Identities = innerStore
		device.Sessions = innerStore
		device.PreKeys = innerStore
		device.SenderKeys = innerStore
		device.MsgSecrets = innerStore
		device.PrivacyTokens = innerStore
	}

	// Create WhatsApp client
	log := waLog.Stdout("Client-"+accountID, config.WhatsappLogLevel, true)
	client := whatsmeow.NewClient(device, log)
	client.EnableAutoReconnect = true
	client.AutoTrustIdentity = true

	// Add event handler
	client.AddEventHandler(func(evt interface{}) {
		u.handleEvent(ctx, accountID, evt)
	})

	// Register client in manager
	u.manager.SetClient(accountID, client, db)

	// IMPORTANT: Get QR channel BEFORE connecting (required by whatsmeow)
	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		return domainAccount.LoginResponse{}, fmt.Errorf("failed to get QR channel: %w", err)
	}

	// Now connect to WhatsApp
	if err := client.Connect(); err != nil {
		return domainAccount.LoginResponse{}, fmt.Errorf("failed to connect: %w", err)
	}

	// If already logged in, update status and return
	if client.IsLoggedIn() {
		acc.Status = domainAccount.StatusLoggedIn
		acc.LastConnected = time.Now()
		if client.Store.ID != nil {
			acc.DeviceID = client.Store.ID.String()
			phoneNumber := strings.Split(client.Store.ID.String(), "@")[0]
			acc.PhoneNumber = phoneNumber
		}
		u.repo.UpdateAccount(acc)

		return domainAccount.LoginResponse{
			ImagePath: "",
			Duration:  0,
			Code:      "ALREADY_LOGGED_IN",
		}, nil
	}

	// Wait for QR code
	select {
	case evt := <-qrChan:
		switch evt.Event {
		case "code":
			// Generate QR code image
			qrPath := u.getQRPath(accountID)
			if err := qrcode.WriteFile(evt.Code, qrcode.Medium, 256, qrPath); err != nil {
				return domainAccount.LoginResponse{}, fmt.Errorf("failed to generate QR code: %w", err)
			}

			return domainAccount.LoginResponse{
				ImagePath: qrPath,
				Duration:  time.Duration(evt.Timeout.Seconds()) * time.Second,
				Code:      evt.Code,
			}, nil

		case "success":
			// Already logged in
			acc.Status = domainAccount.StatusLoggedIn
			acc.LastConnected = time.Now()
			if client.Store.ID != nil {
				acc.DeviceID = client.Store.ID.String()
				phoneNumber := strings.Split(client.Store.ID.String(), "@")[0]
				acc.PhoneNumber = phoneNumber
			}
			u.repo.UpdateAccount(acc)

			return domainAccount.LoginResponse{
				Code: "SUCCESS",
			}, nil

		default:
			return domainAccount.LoginResponse{}, fmt.Errorf("unexpected QR event: %s", evt.Event)
		}

	case <-time.After(60 * time.Second):
		return domainAccount.LoginResponse{}, fmt.Errorf("timeout waiting for QR code")
	}
}

// LoginAccountWithCode logs in an account using pairing code
func (u *AccountUsecase) LoginAccountWithCode(ctx context.Context, accountID string, phoneNumber string) (string, error) {
	// Get account
	acc, err := u.repo.GetAccount(accountID)
	if err != nil {
		return "", fmt.Errorf("account not found")
	}

	// Check if already logged in
	if client := u.manager.GetClient(accountID); client != nil && client.IsLoggedIn() {
		return "", fmt.Errorf("account is already logged in")
	}

	// Initialize database for this account
	dbPath := u.getAccountDBPath(accountID)
	db, err := u.initAccountDatabase(ctx, dbPath)
	if err != nil {
		return "", fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize keys database if configured
	var keysDB *sqlstore.Container
	if config.DBKeysURI != "" {
		keysDBPath := u.getAccountKeysDBPath(accountID)
		keysDB, err = u.initAccountDatabase(ctx, keysDBPath)
		if err != nil {
			logrus.Warnf("Failed to initialize keys database: %v", err)
		} else {
			u.manager.SetKeysDB(accountID, keysDB)
		}
	}

	// Get or create device
	device, err := db.GetFirstDevice(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get device: %w", err)
	}

	// Configure device properties
	osName := fmt.Sprintf("%s %s", config.AppOs, config.AppVersion)
	store.DeviceProps.PlatformType = &config.AppPlatform
	store.DeviceProps.Os = &osName

	// Configure encryption cache database if keysDB exists
	if keysDB != nil && device.ID != nil {
		innerStore := sqlstore.NewSQLStore(keysDB, *device.ID)
		device.Identities = innerStore
		device.Sessions = innerStore
		device.PreKeys = innerStore
		device.SenderKeys = innerStore
		device.MsgSecrets = innerStore
		device.PrivacyTokens = innerStore
	}

	// Create WhatsApp client
	log := waLog.Stdout("Client-"+accountID, config.WhatsappLogLevel, true)
	client := whatsmeow.NewClient(device, log)
	client.EnableAutoReconnect = true
	client.AutoTrustIdentity = true

	// Add event handler
	client.AddEventHandler(func(evt interface{}) {
		u.handleEvent(ctx, accountID, evt)
	})

	// Register client in manager
	u.manager.SetClient(accountID, client, db)

	// Connect to WhatsApp
	if err := client.Connect(); err != nil {
		return "", fmt.Errorf("failed to connect: %w", err)
	}

	// Request pairing code
	code, err := client.PairPhone(ctx, phoneNumber, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		return "", fmt.Errorf("failed to request pairing code: %w", err)
	}

	// Update account with phone number
	acc.PhoneNumber = phoneNumber
	u.repo.UpdateAccount(acc)

	return code, nil
}

// LogoutAccount logs out an account
func (u *AccountUsecase) LogoutAccount(ctx context.Context, accountID string) error {
	// Get client
	client := u.manager.GetClient(accountID)
	if client == nil {
		return fmt.Errorf("account not connected")
	}

	// Logout from WhatsApp
	if err := client.Logout(ctx); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	// Update account status
	acc, err := u.repo.GetAccount(accountID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	acc.Status = domainAccount.StatusDisconnected
	acc.DeviceID = ""
	if err := u.repo.UpdateAccount(acc); err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	// Clean up database
	if err := u.cleanupAccountDatabase(accountID); err != nil {
		logrus.Warnf("Failed to cleanup database for account %s: %v", accountID, err)
	}

	// Remove client
	u.manager.RemoveClient(accountID)

	return nil
}

// ReconnectAccount reconnects an account
func (u *AccountUsecase) ReconnectAccount(ctx context.Context, accountID string) error {
	// Get client
	client := u.manager.GetClient(accountID)
	if client == nil {
		return fmt.Errorf("account not initialized, please login first")
	}

	// Disconnect if connected
	if client.IsConnected() {
		client.Disconnect()
	}

	// Reconnect
	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to reconnect: %w", err)
	}

	// Update account status
	acc, err := u.repo.GetAccount(accountID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	if client.IsLoggedIn() {
		acc.Status = domainAccount.StatusLoggedIn
	} else {
		acc.Status = domainAccount.StatusConnected
	}
	acc.LastConnected = time.Now()

	if err := u.repo.UpdateAccount(acc); err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	return nil
}

// SetAccountWebhook sets webhook for an account
func (u *AccountUsecase) SetAccountWebhook(ctx context.Context, accountID string, webhookURL string, secret string) error {
	// Verify account exists
	if _, err := u.repo.GetAccount(accountID); err != nil {
		return fmt.Errorf("account not found")
	}

	return u.repo.SetWebhook(accountID, webhookURL, secret)
}

// GetAccountWebhook gets webhook for an account
func (u *AccountUsecase) GetAccountWebhook(ctx context.Context, accountID string) (domainAccount.WebhookInfo, error) {
	webhook, err := u.repo.GetWebhook(accountID)
	if err != nil {
		return domainAccount.WebhookInfo{}, err
	}

	return *webhook, nil
}

// Helper methods

func (u *AccountUsecase) buildAccountInfo(acc *domainAccount.Account) domainAccount.AccountInfo {
	info := domainAccount.AccountInfo{
		AccountID:     acc.ID,
		Status:        acc.Status,
		PhoneNumber:   acc.PhoneNumber,
		DeviceID:      acc.DeviceID,
		CreatedAt:     acc.CreatedAt,
		LastConnected: acc.LastConnected,
	}

	// Get real-time connection status from client
	if client := u.manager.GetClient(acc.ID); client != nil {
		info.IsConnected = client.IsConnected()
		info.IsLoggedIn = client.IsLoggedIn()
	}

	// Get webhook info
	if webhook, err := u.repo.GetWebhook(acc.ID); err == nil {
		info.WebhookURL = webhook.URL
	}

	return info
}

func (u *AccountUsecase) getAccountDBPath(accountID string) string {
	baseDir := filepath.Join(config.PathStorages, "accounts", accountID)
	os.MkdirAll(baseDir, 0755)
	return fmt.Sprintf("file:%s/whatsapp.db?_foreign_keys=on", baseDir)
}

func (u *AccountUsecase) getAccountKeysDBPath(accountID string) string {
	baseDir := filepath.Join(config.PathStorages, "accounts", accountID)
	os.MkdirAll(baseDir, 0755)
	return fmt.Sprintf("file:%s/keys.db?_foreign_keys=on", baseDir)
}

func (u *AccountUsecase) getQRPath(accountID string) string {
	return filepath.Join(config.PathQrCode, fmt.Sprintf("scan-%s-%d.png", accountID, time.Now().Unix()))
}

func (u *AccountUsecase) initAccountDatabase(ctx context.Context, dbURI string) (*sqlstore.Container, error) {
	log := waLog.Stdout("Database-"+dbURI, config.WhatsappLogLevel, true)

	var container *sqlstore.Container
	var err error

	if strings.HasPrefix(dbURI, "file:") {
		container, err = sqlstore.New(ctx, "sqlite3", dbURI, log)
	} else if strings.HasPrefix(dbURI, "postgres:") {
		container, err = sqlstore.New(ctx, "postgres", dbURI, log)
	} else {
		return nil, fmt.Errorf("unsupported database type: %s", dbURI)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return container, nil
}

func (u *AccountUsecase) cleanupAccountDatabase(accountID string) error {
	// Close database connections
	if db := u.manager.GetDB(accountID); db != nil {
		db.Close()
	}

	if keysDB := u.manager.GetKeysDB(accountID); keysDB != nil {
		keysDB.Close()
	}

	// Remove database files
	accountDir := filepath.Join(config.PathStorages, "accounts", accountID)
	if err := os.RemoveAll(accountDir); err != nil {
		return fmt.Errorf("failed to remove account directory: %w", err)
	}

	return nil
}

func (u *AccountUsecase) cleanupAccountFiles(accountID string) {
	// Clean up QR codes
	pattern := filepath.Join(config.PathQrCode, fmt.Sprintf("scan-%s-*.png", accountID))
	if files, err := filepath.Glob(pattern); err == nil {
		for _, f := range files {
			os.Remove(f)
		}
	}
}

func (u *AccountUsecase) handleEvent(ctx context.Context, accountID string, evt interface{}) {
	acc, err := u.repo.GetAccount(accountID)
	if err != nil {
		logrus.Errorf("[%s] Failed to get account: %v", accountID, err)
		return
	}

	switch e := evt.(type) {
	case *events.Connected:
		logrus.Infof("[%s] Connected to WhatsApp", accountID)
		acc.Status = domainAccount.StatusConnected
		acc.LastConnected = time.Now()
		u.repo.UpdateAccount(acc)

	case *events.LoggedOut:
		logrus.Infof("[%s] Logged out from WhatsApp", accountID)
		acc.Status = domainAccount.StatusDisconnected
		acc.DeviceID = ""
		u.repo.UpdateAccount(acc)

		// Cleanup
		u.cleanupAccountDatabase(accountID)
		u.manager.RemoveClient(accountID)

	case *events.PairSuccess:
		logrus.Infof("[%s] Successfully paired with %s", accountID, e.ID.String())
		acc.Status = domainAccount.StatusLoggedIn
		acc.DeviceID = e.ID.String()
		acc.PhoneNumber = strings.Split(e.ID.String(), "@")[0]
		acc.LastConnected = time.Now()
		u.repo.UpdateAccount(acc)

	case *events.Message:
		// Store message in chat storage
		if u.chatStorageRepo != nil {
			if err := u.chatStorageRepo.CreateMessage(ctx, e); err != nil {
				logrus.Errorf("[%s] Failed to store message: %v", accountID, err)
			}
		}

		// Forward to webhook if configured
		if webhook, err := u.repo.GetWebhook(accountID); err == nil && webhook.URL != "" {
			go u.forwardToWebhook(accountID, webhook, e)
		}
	}
}

func (u *AccountUsecase) forwardToWebhook(accountID string, webhook *domainAccount.WebhookInfo, message interface{}) {
	// TODO: Implement webhook forwarding
	// This should be similar to the existing webhook implementation but per account
	logrus.Debugf("[%s] Forwarding message to webhook: %s", accountID, webhook.URL)
}
