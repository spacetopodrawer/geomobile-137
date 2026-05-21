package service

import "errors"

var (
	ErrInvalidParcelID      = errors.New("invalid parcel ID")
	ErrInvalidClientID      = errors.New("invalid client ID")
	ErrInvalidOperationType = errors.New("invalid operation type")
	ErrInvalidTimestamp     = errors.New("invalid timestamp")
	ErrParcelNotFound       = errors.New("parcel not found")
	ErrUserNotFound         = errors.New("user not found")
	ErrConversionFailed     = errors.New("CAD conversion failed")
	ErrUnauthorized         = errors.New("unauthorized")
)
