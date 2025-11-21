package send

type BaseRequest struct {
	AccountID   string `json:"account_id" form:"account_id"`
	Phone       string `json:"phone" form:"phone"`
	Duration    *int   `json:"duration,omitempty" form:"duration"`
	IsForwarded bool   `json:"is_forwarded,omitempty" form:"is_forwarded"`
}
