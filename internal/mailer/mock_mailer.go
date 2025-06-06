package mailer

import (
	"sync"
)

// Email represents a sent email
type Email struct {
	Recipient    string
	TemplateFile string
	Data         any
}

// MockMailer is a mock implementation of the Mailer interface for testing
type MockMailer struct {
	mu     sync.RWMutex
	emails []Email
}

// NewMockMailer creates a new MockMailer instance
func NewMockMailer() *MockMailer {
	return &MockMailer{
		emails: make([]Email, 0),
	}
}

// Send records the email that would have been sent
func (m *MockMailer) Send(recipient, templateFile string, data any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.emails = append(m.emails, Email{
		Recipient:    recipient,
		TemplateFile: templateFile,
		Data:         data,
	})

	return nil
}

// GetSentEmails returns a copy of all sent emails
func (m *MockMailer) GetSentEmails() []Email {
	m.mu.RLock()
	defer m.mu.RUnlock()

	emails := make([]Email, len(m.emails))
	copy(emails, m.emails)
	return emails
}

// Reset clears the record of sent emails
func (m *MockMailer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.emails = make([]Email, 0)
}
