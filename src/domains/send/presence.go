package send

type PresenceRequest struct {
	AccountID   string `json:"account_id" form:"account_id"`
	Type        string `json:"type" form:"type"`
	IsForwarded bool   `json:"is_forwarded" form:"is_forwarded"`
}
