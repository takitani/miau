package mocks

import (
	"context"

	"github.com/opik/miau/internal/ports"
	"github.com/stretchr/testify/mock"
)

// GmailAPIPort is a mock implementation of ports.GmailAPIPort
type GmailAPIPort struct {
	mock.Mock
}

// Send sends an email via Gmail API
func (m *GmailAPIPort) Send(ctx context.Context, req *ports.SendRequest) (*ports.SendResult, error) {
	var args = m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.SendResult), args.Error(1)
}

// GetSignature retrieves the user's signature
func (m *GmailAPIPort) GetSignature(ctx context.Context) (string, error) {
	var args = m.Called(ctx)
	return args.String(0), args.Error(1)
}

// Archive archives an email
func (m *GmailAPIPort) Archive(ctx context.Context, messageID string) error {
	var args = m.Called(ctx, messageID)
	return args.Error(0)
}

// Ensure GmailAPIPort implements ports.GmailAPIPort
var _ ports.GmailAPIPort = (*GmailAPIPort)(nil)
