package message

import (
	"time"

	"github.com/uptrace/bun"
)

type Message struct {
	bun.BaseModel `bun:"table:messages,alias:m"`

	ID        int       `bun:"id,pk,autoincrement" json:"id"`
	Email     string    `bun:"email,notnull" json:"email"`
	Message   string    `bun:"message,notnull" json:"message"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
}

type MessageEvent struct {
	Email   string `json:"email"`
	Message string `json:"message"`
}
