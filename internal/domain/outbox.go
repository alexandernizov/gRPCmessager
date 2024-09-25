package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	ChatTopic    = "chats"
	MessageTopic = "messages"
)

type Outbox struct {
	Uuid    uuid.UUID
	Topic   string
	Message []byte
	Sent_at time.Time
}
