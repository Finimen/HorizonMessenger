package keying

import (
	"encoding/json"
	"massager/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyig_TableDrivn(t *testing.T) {
	tests := []struct {
		name         string
		action       func() bool
		targetResult bool
	}{
		{
			name: "Test Keying",
			action: func() bool {
				var messageString = "Hello world"
				var key, _ = GenerateKeyAES128()
				messageModel := models.Message{
					Type:    "message",
					Content: messageString,
					Key:     key,
				}
				bytedMessage, _ := json.Marshal(messageModel)
				var newMessage models.Message
				json.Unmarshal(bytedMessage, &newMessage)
				newMessage.Content, _ = Decrypt(newMessage.Key, newMessage.Content)

				return newMessage.Content == messageString
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.action()
			assert.Equal(t, result, tt.targetResult)
		})
	}
}
