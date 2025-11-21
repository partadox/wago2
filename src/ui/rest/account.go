package rest

import (
	"github.com/aldinokemal/go-whatsapp-web-multidevice/domains/account"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

func InitRestAccount(app fiber.Router, accountService account.IAccountUsecase) {
	app.Post("/accounts", createAccount(accountService))
	app.Get("/accounts", listAccounts(accountService))
	app.Get("/accounts/:accountId", getAccount(accountService))
	app.Delete("/accounts/:accountId", deleteAccount(accountService))
	app.Post("/accounts/:accountId/login", loginAccount(accountService))
	app.Post("/accounts/:accountId/login-with-code", loginAccountWithCode(accountService))
	app.Post("/accounts/:accountId/logout", logoutAccount(accountService))
	app.Post("/accounts/:accountId/reconnect", reconnectAccount(accountService))
	app.Post("/accounts/:accountId/webhook", setAccountWebhook(accountService))
	app.Get("/accounts/:accountId/webhook", getAccountWebhook(accountService))
}

func createAccount(service account.IAccountUsecase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			AccountID string `json:"account_id" validate:"required,min=1,max=50,alphanum"`
		}

		if err := c.BodyParser(&req); err != nil {
			response := utils.BadRequest("Invalid request body")
			return c.Status(response.Status).JSON(response)
		}

		if req.AccountID == "" {
			response := utils.BadRequest("account_id is required")
			return c.Status(response.Status).JSON(response)
		}

		if len(req.AccountID) < 1 || len(req.AccountID) > 50 {
			response := utils.BadRequest("account_id must be between 1 and 50 characters")
			return c.Status(response.Status).JSON(response)
		}

		result, err := service.CreateAccount(c.Context(), req.AccountID)
		if err != nil {
			response := utils.Error(500, err.Error())
			return c.Status(response.Status).JSON(response)
		}

		response := utils.Success("Account created successfully", result)
		return c.Status(response.Status).JSON(response)
	}
}

func listAccounts(service account.IAccountUsecase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		accounts, err := service.ListAccounts(c.Context())
		if err != nil {
			response := utils.Error(500, err.Error())
			return c.Status(response.Status).JSON(response)
		}

		response := utils.Success("Success get accounts", accounts)
		return c.Status(response.Status).JSON(response)
	}
}

func getAccount(service account.IAccountUsecase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		accountID := c.Params("accountId")
		if accountID == "" {
			response := utils.BadRequest("Account ID is required")
			return c.Status(response.Status).JSON(response)
		}

		acc, err := service.GetAccount(c.Context(), accountID)
		if err != nil {
			response := utils.Error(500, err.Error())
			return c.Status(response.Status).JSON(response)
		}

		response := utils.Success("Success get account", acc)
		return c.Status(response.Status).JSON(response)
	}
}

func deleteAccount(service account.IAccountUsecase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		accountID := c.Params("accountId")
		if accountID == "" {
			response := utils.BadRequest("Account ID is required")
			return c.Status(response.Status).JSON(response)
		}

		err := service.DeleteAccount(c.Context(), accountID)
		if err != nil {
			response := utils.Error(500, err.Error())
			return c.Status(response.Status).JSON(response)
		}

		response := utils.Success("Account deleted successfully", nil)
		return c.Status(response.Status).JSON(response)
	}
}

func loginAccount(service account.IAccountUsecase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		accountID := c.Params("accountId")
		if accountID == "" {
			response := utils.BadRequest("Account ID is required")
			return c.Status(response.Status).JSON(response)
		}

		result, err := service.LoginAccount(c.Context(), accountID)
		if err != nil {
			response := utils.Error(500, err.Error())
			return c.Status(response.Status).JSON(response)
		}

		response := utils.Success("Please scan the QR code", result)
		return c.Status(response.Status).JSON(response)
	}
}

func loginAccountWithCode(service account.IAccountUsecase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		accountID := c.Params("accountId")
		if accountID == "" {
			response := utils.BadRequest("Account ID is required")
			return c.Status(response.Status).JSON(response)
		}

		var req struct {
			PhoneNumber string `json:"phone_number" validate:"required,min=10,max=15"`
		}

		if err := c.BodyParser(&req); err != nil {
			response := utils.BadRequest("Invalid request body")
			return c.Status(response.Status).JSON(response)
		}

		if req.PhoneNumber == "" {
			response := utils.BadRequest("phone_number is required")
			return c.Status(response.Status).JSON(response)
		}

		if len(req.PhoneNumber) < 10 || len(req.PhoneNumber) > 15 {
			response := utils.BadRequest("phone_number must be between 10 and 15 characters")
			return c.Status(response.Status).JSON(response)
		}

		code, err := service.LoginAccountWithCode(c.Context(), accountID, req.PhoneNumber)
		if err != nil {
			response := utils.Error(500, err.Error())
			return c.Status(response.Status).JSON(response)
		}

		result := map[string]string{
			"code":    code,
			"message": "Please enter this code in your WhatsApp app",
		}

		response := utils.Success("Login code generated successfully", result)
		return c.Status(response.Status).JSON(response)
	}
}

func logoutAccount(service account.IAccountUsecase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		accountID := c.Params("accountId")
		if accountID == "" {
			response := utils.BadRequest("Account ID is required")
			return c.Status(response.Status).JSON(response)
		}

		err := service.LogoutAccount(c.Context(), accountID)
		if err != nil {
			response := utils.Error(500, err.Error())
			return c.Status(response.Status).JSON(response)
		}

		response := utils.Success("Account logged out successfully", nil)
		return c.Status(response.Status).JSON(response)
	}
}

func reconnectAccount(service account.IAccountUsecase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		accountID := c.Params("accountId")
		if accountID == "" {
			response := utils.BadRequest("Account ID is required")
			return c.Status(response.Status).JSON(response)
		}

		err := service.ReconnectAccount(c.Context(), accountID)
		if err != nil {
			response := utils.Error(500, err.Error())
			return c.Status(response.Status).JSON(response)
		}

		response := utils.Success("Account reconnected successfully", nil)
		return c.Status(response.Status).JSON(response)
	}
}

func setAccountWebhook(service account.IAccountUsecase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		accountID := c.Params("accountId")
		if accountID == "" {
			response := utils.BadRequest("Account ID is required")
			return c.Status(response.Status).JSON(response)
		}

		var req struct {
			WebhookURL string `json:"webhook_url" validate:"required,url"`
			Secret     string `json:"secret"`
		}

		if err := c.BodyParser(&req); err != nil {
			response := utils.BadRequest("Invalid request body")
			return c.Status(response.Status).JSON(response)
		}

		if req.WebhookURL == "" {
			response := utils.BadRequest("webhook_url is required")
			return c.Status(response.Status).JSON(response)
		}

		err := service.SetAccountWebhook(c.Context(), accountID, req.WebhookURL, req.Secret)
		if err != nil {
			response := utils.Error(500, err.Error())
			return c.Status(response.Status).JSON(response)
		}

		response := utils.Success("Webhook set successfully", nil)
		return c.Status(response.Status).JSON(response)
	}
}

func getAccountWebhook(service account.IAccountUsecase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		accountID := c.Params("accountId")
		if accountID == "" {
			response := utils.BadRequest("Account ID is required")
			return c.Status(response.Status).JSON(response)
		}

		webhook, err := service.GetAccountWebhook(c.Context(), accountID)
		if err != nil {
			response := utils.Error(500, err.Error())
			return c.Status(response.Status).JSON(response)
		}

		response := utils.Success("Success get webhook", webhook)
		return c.Status(response.Status).JSON(response)
	}
}
