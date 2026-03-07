package auth

import (
	commonauth "homelab/pkg/common/auth"
)

var (
	ErrUnauthorized = commonauth.ErrUnauthorized
	ErrTotpRequired = commonauth.ErrTotpRequired
)
