package domain

import (
	"time"

	"github.com/google/uuid"
)

type Chat struct {
	Uuid     uuid.UUID
	Owner    User
	Readonly bool
	Deadline time.Time
}
