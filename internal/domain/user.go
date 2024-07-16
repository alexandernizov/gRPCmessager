package domain

import "github.com/google/uuid"

type User struct {
	Uuid     uuid.UUID
	Name     string
	Password string
}

type UserCtxKey struct {
}
