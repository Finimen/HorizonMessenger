package models

import "errors"

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"emal"`
}

func NewUser(username, password, email string) *User {
	return &User{Username: username, Password: password, Email: email}
}

func ValidateUser(user *User) error {
	if user.Username == "" || user.Password == "" || user.Email == "" {
		return errors.New("Empty fields detected")
	} else {
		return nil
	}
}
