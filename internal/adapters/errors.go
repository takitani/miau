package adapters

import "errors"

var (
	// ErrNotConnected is returned when operation requires connection
	ErrNotConnected = errors.New("not connected to server")

	// ErrNoAccount is returned when no account is configured
	ErrNoAccount = errors.New("no account configured")

	// ErrNotFound is returned when resource is not found
	ErrNotFound = errors.New("not found")
)
