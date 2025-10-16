package models

import "fmt"

// Common error variables used across all models
var (
	ErrInvalidInput     = fmt.Errorf("invalid input")
	ErrNotFound         = fmt.Errorf("not found")
	ErrUnauthorized     = fmt.Errorf("unauthorized")
	ErrPermissionDenied = fmt.Errorf("permission denied")
	ErrDuplicateEntry   = fmt.Errorf("duplicate entry")
)
