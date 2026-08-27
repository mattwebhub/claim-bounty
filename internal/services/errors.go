package services

import "errors"

var ErrInvalidDependencies = errors.New("service dependencies are incomplete")
