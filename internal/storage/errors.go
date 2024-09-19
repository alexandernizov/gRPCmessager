package storage

import "errors"

var (
	ErrNoConnection = errors.New("can't establish connection to db")

	ErrInternal = errors.New("internal error")

	ErrUserNotFound  = errors.New("user is not found")
	ErrTokenNotFound = errors.New("token is not found")
	ErrChatNotFound  = errors.New("chat is not found")

	ErrNoOutbox = errors.New("have no outbox to send")
)
