package session

import (
	"github.com/google/uuid"
)

// GenerateConversationID generates a unique conversation ID for temporary conversations
func GenerateConversationID() string {
	return uuid.New().String()
}