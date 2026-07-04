package utils

import "errors"

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden - not a participant")
	ErrConversationNotFound = errors.New("conversation not found")
	ErrMessageNotFound    = errors.New("message not found")
	ErrUserBlocked        = errors.New("user blocked communication")
	ErrInvalidPayload     = errors.New("invalid event payload")
	ErrDatabaseError      = errors.New("database error")
	ErrUserNotParticipant = errors.New("user not a participant in this conversation")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
)
