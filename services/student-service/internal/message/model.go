package message

type SendMessageRequest struct {
	Message string `json:"message" validate:"required"`
}

type MessageEvent struct {
	Email   string `json:"email"`
	Message string `json:"message"`
}
