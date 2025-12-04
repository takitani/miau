package mocks

import (
	"context"

	"github.com/opik/miau/internal/ports"
	"github.com/stretchr/testify/mock"
)

// SMTPPort is a mock implementation of ports.SMTPPort
type SMTPPort struct {
	mock.Mock
}

// Send sends an email via SMTP
func (m *SMTPPort) Send(ctx context.Context, req *ports.SendRequest) (*ports.SendResult, error) {
	var args = m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.SendResult), args.Error(1)
}

// Ensure SMTPPort implements ports.SMTPPort
var _ ports.SMTPPort = (*SMTPPort)(nil)
