package errs

import "errors"

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrSessionExpired    = errors.New("session expired")

	ErrChatDoesNotExist        = errors.New("chat doesn't exist")
	ErrOnlyOwnerCanPostMessage = errors.New("only owner of this chat can post messages")

	ErrMessageNotFound = errors.New("message not found")
)
