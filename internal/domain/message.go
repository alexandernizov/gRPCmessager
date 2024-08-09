package domain

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	//Uuid       uuid.UUID
	AuthorUuid uuid.UUID
	Body       string
	Published  time.Time
}
