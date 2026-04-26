package secret

import (
	"context"
	"fmt"
	"strings"

	"homelab/pkg/common"
	secretmodel "homelab/pkg/models/core/secret"
	runtimepkg "homelab/pkg/runtime"

	"gopkg.d7z.net/middleware/kv"
)

var secretRepo = common.NewResourceRepository[secretmodel.SecretV1Meta, secretmodel.SecretV1Status]("system", "Secret")

func SecretID(ownerKind, ownerID, purpose string) string {
	return fmt.Sprintf("%s-%s-%s", strings.ToLower(ownerKind), strings.ToLower(purpose), strings.ToLower(ownerID))
}

func GetSecret(ctx context.Context, id string) (*secretmodel.Secret, error) {
	return secretRepo.Get(ctx, id)
}

func GetSecretByOwner(ctx context.Context, ownerKind, ownerID, purpose string) (*secretmodel.Secret, error) {
	return secretRepo.Get(ctx, SecretID(ownerKind, ownerID, purpose))
}

func SaveSecret(ctx context.Context, secret *secretmodel.Secret) error {
	return secretRepo.Save(ctx, secret)
}

func DeleteSecret(ctx context.Context, id string) error {
	return secretRepo.Delete(ctx, id)
}

func PutDigestIndex(ctx context.Context, purpose, digest, secretID string) error {
	db := runtimepkg.DBFromContext(ctx)
	if db == nil {
		return fmt.Errorf("db not configured")
	}
	return db.Child("system", "secrets", "index", purpose).Put(ctx, digest, secretID, kv.TTLKeep)
}

func GetDigestIndex(ctx context.Context, purpose, digest string) (string, error) {
	db := runtimepkg.DBFromContext(ctx)
	if db == nil {
		return "", fmt.Errorf("db not configured")
	}
	return db.Child("system", "secrets", "index", purpose).Get(ctx, digest)
}

func DeleteDigestIndex(ctx context.Context, purpose, digest string) error {
	db := runtimepkg.DBFromContext(ctx)
	if db == nil {
		return fmt.Errorf("db not configured")
	}
	_, err := db.Child("system", "secrets", "index", purpose).Delete(ctx, digest)
	return err
}
