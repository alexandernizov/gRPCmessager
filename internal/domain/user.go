package domain

import "github.com/google/uuid"

type User struct {
	Uuid         uuid.UUID
	Login        string
	PasswordHash []byte
}

type UserUuidCtxKey struct {
}
