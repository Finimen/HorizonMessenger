//go:build swagger
// +build swagger

package handlers

// DTO strutcs only for Swagger documetation

// LoginRequest represents login request data
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest represents registration request data
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

// CreateChatRequest represents chat creation request data
type CreateChatRequest struct {
	MemberIDs []string `json:"member_ids" binding:"required"`
	ChatName  string   `json:"chat_name" binding:"required"`
}
