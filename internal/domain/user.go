package domain

import "github.com/google/uuid"

type User struct {
	Uuid         uuid.UUID
	Login        string
	PasswordHash []byte
}

type Tokens struct {
	AccessToken  string
	RefreshToken string
}

type UserUuidCtxKey struct {
}
