package secret

import (
	"context"
	"errors"
	"strings"
	"time"

	"homelab/pkg/models/shared"
)

const (
	OwnerKindWorkflow       = "workflow"
	OwnerKindServiceAccount = "serviceaccount"

	PurposeWebhookToken = "webhook_token"
	PurposeAuthToken    = "auth_token"
)

type SecretV1Meta struct {
	// OwnerKind identifies the owning resource kind, for example "workflow" or "serviceaccount".
	OwnerKind string `json:"ownerKind"`
	// OwnerID identifies the owning resource instance.
	OwnerID string `json:"ownerId"`
	// Purpose identifies the secret usage, for example "webhook_token" or "auth_token".
	Purpose string `json:"purpose"`
	// Algorithm records the encryption algorithm used for the ciphertext payload.
	Algorithm string `json:"algorithm"`
}

func (m *SecretV1Meta) Validate(_ context.Context) error {
	m.OwnerKind = strings.TrimSpace(m.OwnerKind)
	m.OwnerID = strings.TrimSpace(m.OwnerID)
	m.Purpose = strings.TrimSpace(m.Purpose)
	m.Algorithm = strings.TrimSpace(m.Algorithm)
	if m.OwnerKind == "" || m.OwnerID == "" || m.Purpose == "" {
		return errors.New("ownerKind, ownerId and purpose are required")
	}
	if m.Algorithm == "" {
		return errors.New("algorithm is required")
	}
	return nil
}

type SecretV1Status struct {
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	Version    int64     `json:"version"`
	LastUsedAt time.Time `json:"lastUsedAt,omitempty"`
	// Digest stores an HMAC digest used for O(1) lookup without decrypting the secret body.
	Digest string `json:"digest"`
	// Nonce stores the AES-GCM nonce used to encrypt CipherText.
	Nonce string `json:"nonce"`
	// CipherText stores the encrypted secret payload.
	CipherText string `json:"cipherText"`
}

type Secret = shared.Resource[SecretV1Meta, SecretV1Status]
