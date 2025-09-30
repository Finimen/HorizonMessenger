package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"massager/app/config"
	"strconv"

	"gopkg.in/gomail.v2"
)

type EmailService struct {
	from   string
	dialer *gomail.Dialer
	logger *slog.Logger
}

func NewEmailService(config config.EmailConfig, loggger *slog.Logger) *EmailService {
	var port, _ = strconv.Atoi(config.SMTPort)
	var dialer = gomail.NewDialer(config.SMTHost, port, config.Username, config.Password)

	return &EmailService{
		logger: loggger,
		dialer: dialer,
		from:   config.From,
	}
}

func (e *EmailService) SendVerificationEmail(email, token string) error {
	verificationLink := fmt.Sprintf("http://localhost:8080/api/auth/verify-email?token=%s", token)

	var message = gomail.NewMessage()
	message.SetHeader("From", e.from)
	message.SetHeader("To", email)
	message.SetHeader("Subject", "Verify Your Email Address")

	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>Verify Your Email</title>
		</head>
		<body>
			<h2>Email Verification</h2>
			<p>Hello,</p>
			<p>Please verify your email address by clicking the button below:</p>
			<p>
				<a href="%s" style="
					background-color: #007bff; 
					color: white; 
					padding: 12px 24px; 
					text-decoration: none; 
					border-radius: 4px; 
					display: inline-block;
				">Verify Email</a>
			</p>
			<p>Or copy and paste this link in your browser:</p>
			<p>%s</p>
			<p>If you didn't create an account, please ignore this email.</p>
			<br>
			<p>Best regards,<br>Your App Team</p>
		</body>
		</html>
	`, verificationLink, verificationLink)

	message.SetBody("text/html", htmlBody)

	textBody := fmt.Sprintf(`
		Verify Your Email Address
		
		Please verify your email address by visiting the following link:
		%s
		
		If you didn't create an account, please ignore this email.
		
		Best regards,
		Your App Team
	`, verificationLink)

	message.AddAlternative("text/plain", textBody)

	if err := e.dialer.DialAndSend(message); err != nil {
		e.logger.Error("failed to send verification email", "error", err, "email", email)
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	e.logger.Info("sending verification email", "email", email, "link", verificationLink)
	fmt.Printf("Verification link for %s: %s\n", email, verificationLink)
	return nil
}

func generateVerificationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
