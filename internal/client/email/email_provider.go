package email

import (
	"log"
)

type EmailProviderStub struct{}

// NewEmailProviderStub creates a new instance of EmailProviderStub
func NewEmailProviderStub() *EmailProviderStub {
	return &EmailProviderStub{}
}

// SendVerificationEmail is a stub method that simulates sending a verification email
func (e *EmailProviderStub) SendVerificationEmail(to string, subject string, body string) error {
	// Simulate sending an email by printing to console
	// In a real implementation, this would interface with an email service provider
	log.Printf("Sending verification email to: %s, subject: %s, body: %s", to, subject, body)
	return nil // Simulate success
}
