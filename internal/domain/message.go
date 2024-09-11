package domain

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	//Uuid       uuid.UUID
	Id         int
	AuthorUuid uuid.UUID
	Body       string
	Published  time.Time
}
