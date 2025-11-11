package web


type NotificationRequest struct { 
	Channel   string `json:"channel" binding:"required,oneof=email telegram"`
	Message   string `json:"message" binding:"required"`
	Recipient string `json:"recipient" binding:"required"`
	SendAt    string `json:"send_at" binding:"required,datetime=2006-01-02T15:04:05Z07:00"`
}