package domain

import (
	"container/list"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Chat struct {
	Uuid     uuid.UUID
	Owner    User
	ReadOnly bool
	Capacity int
	Mux      *sync.RWMutex
	Messages map[uuid.UUID]*Message
	Queue    *list.List
}

type Message struct {
	Uuid      uuid.UUID
	Author    User
	Published time.Time
	Message   string
}
