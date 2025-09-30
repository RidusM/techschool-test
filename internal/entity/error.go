package entity

import (
	"errors"
)

var (
	ErrDataNotFound     = errors.New("data not found")
	ErrConflictingData  = errors.New("data conflicts with existing data in unique column")
	ErrInvalidData      = errors.New("invalid data")
	ErrConfigPathNotSet = errors.New("CONFIG_PATH not set and -config flag not provided")
)
