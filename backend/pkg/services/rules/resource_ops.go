package rules

import (
	"context"
	"strings"

	"homelab/pkg/models/shared"
)

type ResourceRepository[M any, S any] interface {
	Cow(ctx context.Context, id string, mutate func(*shared.Resource[M, S]) error) error
	PatchMeta(ctx context.Context, id string, expectedGeneration int64, apply func(*M)) error
	Get(ctx context.Context, id string) (*shared.Resource[M, S], error)
	List(ctx context.Context, cursor string, limit int, filter func(*shared.Resource[M, S]) bool) (*shared.PaginationResponse[shared.Resource[M, S]], error)
}

func CreateAndLoad[M any, S any](ctx context.Context, repo ResourceRepository[M, S], target *shared.Resource[M, S], initialize func(*shared.Resource[M, S]) error) error {
	if err := repo.Cow(ctx, target.ID, func(res *shared.Resource[M, S]) error {
		if err := initialize(res); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	updated, err := repo.Get(ctx, target.ID)
	if err != nil {
		return err
	}
	if updated != nil {
		*target = *updated
	}
	return nil
}

func ReplaceMeta[M any, S any](ctx context.Context, repo ResourceRepository[M, S], target *shared.Resource[M, S]) error {
	return repo.PatchMeta(ctx, target.ID, target.Generation, func(meta *M) {
		*meta = target.Meta
	})
}

func ScanBySearch[M any, S any](ctx context.Context, repo ResourceRepository[M, S], cursor string, limit int, search string, visible func(*shared.Resource[M, S]) bool, name func(*M) string) (*shared.PaginationResponse[shared.Resource[M, S]], error) {
	search = strings.ToLower(search)
	return repo.List(ctx, cursor, limit, func(item *shared.Resource[M, S]) bool {
		if visible != nil && !visible(item) {
			return false
		}
		if search == "" {
			return true
		}
		return strings.Contains(strings.ToLower(name(&item.Meta)), search) || strings.Contains(strings.ToLower(item.ID), search)
	})
}
