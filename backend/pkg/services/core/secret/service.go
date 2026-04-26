package secret

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"homelab/pkg/common"
	secretmodel "homelab/pkg/models/core/secret"
	secretrepo "homelab/pkg/repositories/core/secret"
)

const aes256GCMMode = "aes256-gcm"
const plainMode = "plain"

type config struct {
	algorithm string
	key       []byte
	digestKey []byte
}

func ValidateConfig() error {
	_, err := loadConfig()
	return err
}

func Put(ctx context.Context, ownerKind, ownerID, purpose, plaintext string) error {
	if plaintext == "" {
		return errors.New("secret plaintext is required")
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	nonce := ""
	ciphertext := plaintext
	if cfg.algorithm == aes256GCMMode {
		nonce, ciphertext, err = encryptAESGCM(cfg.key, plaintext)
		if err != nil {
			return err
		}
	}
	now := time.Now()
	secretID := secretrepo.SecretID(ownerKind, ownerID, purpose)
	current, err := secretrepo.GetSecret(ctx, secretID)
	version := int64(1)
	createdAt := now
	if err == nil && current != nil {
		version = current.Status.Version + 1
		createdAt = current.Status.CreatedAt
	}
	digest := computeDigest(cfg.digestKey, plaintext)
	secret := &secretmodel.Secret{
		ID: secretID,
		Meta: secretmodel.SecretV1Meta{
			OwnerKind: ownerKind,
			OwnerID:   ownerID,
			Purpose:   purpose,
			Mode:      cfg.algorithm,
		},
		Status: secretmodel.SecretV1Status{
			CreatedAt: createdAt,
			UpdatedAt: now,
			Version:   version,
			Digest:    digest,
			Nonce:     nonce,
			Payload:   ciphertext,
		},
	}
	if current != nil && current.Status.Digest != "" && current.Status.Digest != digest {
		_ = secretrepo.DeleteDigestIndex(ctx, purpose, current.Status.Digest)
	}
	if err := secretrepo.SaveSecret(ctx, secret); err != nil {
		return err
	}
	return secretrepo.PutDigestIndex(ctx, purpose, digest, secretID)
}

func Delete(ctx context.Context, ownerKind, ownerID, purpose string) error {
	secret, err := secretrepo.GetSecretByOwner(ctx, ownerKind, ownerID, purpose)
	if err != nil {
		return err
	}
	if secret.Status.Digest != "" {
		_ = secretrepo.DeleteDigestIndex(ctx, purpose, secret.Status.Digest)
	}
	return secretrepo.DeleteSecret(ctx, secret.ID)
}

func Get(ctx context.Context, ownerKind, ownerID, purpose string) (string, error) {
	secret, err := secretrepo.GetSecretByOwner(ctx, ownerKind, ownerID, purpose)
	if err != nil {
		return "", err
	}
	switch secret.Meta.Mode {
	case plainMode:
		return secret.Status.Payload, nil
	case aes256GCMMode:
		cfg, err := loadConfig()
		if err != nil {
			return "", err
		}
		if cfg.algorithm != aes256GCMMode {
			return "", errors.New("secret storage requires aes256 configuration")
		}
		return decryptAESGCM(cfg.key, secret.Status.Nonce, secret.Status.Payload)
	default:
		return "", errors.New("unsupported secret algorithm")
	}
}

func Has(ctx context.Context, ownerKind, ownerID, purpose string) bool {
	secret, err := secretrepo.GetSecretByOwner(ctx, ownerKind, ownerID, purpose)
	return err == nil && secret != nil
}

func Matches(ctx context.Context, ownerKind, ownerID, purpose, plaintext string) bool {
	cfg, err := loadConfig()
	if err != nil {
		return false
	}
	secret, err := secretrepo.GetSecretByOwner(ctx, ownerKind, ownerID, purpose)
	if err != nil || secret == nil {
		return false
	}
	return hmac.Equal([]byte(secret.Status.Digest), []byte(computeDigest(cfg.digestKey, plaintext)))
}

func FindOwnerIDByPlaintext(ctx context.Context, purpose, plaintext string) (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	digest := computeDigest(cfg.digestKey, plaintext)
	secretID, err := secretrepo.GetDigestIndex(ctx, purpose, digest)
	if err != nil {
		return "", err
	}
	secret, err := secretrepo.GetSecret(ctx, secretID)
	if err != nil {
		return "", err
	}
	return secret.Meta.OwnerID, nil
}

func Touch(ctx context.Context, ownerKind, ownerID, purpose string) error {
	secret, err := secretrepo.GetSecretByOwner(ctx, ownerKind, ownerID, purpose)
	if err != nil {
		return err
	}
	secret.Status.LastUsedAt = time.Now()
	return secretrepo.SaveSecret(ctx, secret)
}

func loadConfig() (*config, error) {
	raw := strings.TrimSpace(common.Opts.Secret)
	if raw == "" {
		return nil, errors.New("secret mode is required")
	}
	if raw == "plain" {
		digest := sha256.Sum256([]byte("plain"))
		return &config{
			algorithm: plainMode,
			digestKey: digest[:],
		}, nil
	}
	if !strings.HasPrefix(raw, "aes256:") {
		return nil, errors.New("secret mode must be 'plain' or 'aes256:<key>'")
	}
	key, err := decodeKey(strings.TrimSpace(strings.TrimPrefix(raw, "aes256:")))
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, errors.New("secret aes256 key must be 32 bytes")
	}
	digestSeed := append([]byte("digest:"), key...)
	digest := sha256.Sum256(digestSeed)
	return &config{
		algorithm: aes256GCMMode,
		key:       key,
		digestKey: digest[:],
	}, nil
}

func decodeKey(raw string) ([]byte, error) {
	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil {
		return decoded, nil
	}
	return []byte(raw), nil
}

func computeDigest(key []byte, plaintext string) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(plaintext))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func encryptAESGCM(key []byte, plaintext string) (string, string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(nonce), base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptAESGCM(key []byte, nonceB64, ciphertextB64 string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return "", err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
