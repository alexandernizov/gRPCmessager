package domain

import (
	"time"

	"github.com/google/uuid"
)

type Outbox struct {
	Uuid    uuid.UUID
	Topic   string
	Message []byte
	Sent_at time.Time
}
