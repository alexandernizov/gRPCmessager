package domain

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	SessionUUID uuid.UUID
	UserUUID    uuid.UUID
	ExpiresAt   time.Time
}
