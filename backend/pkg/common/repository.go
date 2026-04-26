package common

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	metav1 "homelab/pkg/apis/meta/v1"
	"homelab/pkg/models/shared"
	"homelab/pkg/store"
)

type ResourceRepository[M any, S any] struct {
	store *store.ResourceStore[M, S]
}

func NewResourceRepository[M any, S any](module, model string) *ResourceRepository[M, S] {
	resource := strings.ToLower(strings.ReplaceAll(module+"-"+model, "/", "-"))
	return &ResourceRepository[M, S]{
		store: store.NewResourceStore[M, S](
			nil,
			"internal/v1",
			model,
			resource,
			func(ctx context.Context, spec *M) error {
				if validator, ok := any(spec).(shared.ConfigValidator); ok {
					if err := validator.Validate(ctx); err != nil {
						return err
					}
				}
				return nil
			},
		),
	}
}

func (r *ResourceRepository[M, S]) Save(ctx context.Context, res *shared.Resource[M, S]) error {
	if res == nil {
		return fmt.Errorf("%w: resource is required", ErrBadRequest)
	}

	existing, err := r.store.Get(ctx, res.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			obj := &metav1.Object[M, S]{
				Metadata: metav1.ObjectMeta{Name: res.ID},
				Spec:     res.Meta,
				Status:   res.Status,
			}
			return mapStoreError(r.store.Create(ctx, obj))
		}
		return mapStoreError(err)
	}

	existing.Spec = res.Meta
	existing.Status = res.Status
	if res.Generation > 0 {
		existing.Metadata.Generation = res.Generation
	}
	currentVersion, _ := strconv.ParseInt(existing.Metadata.ResourceVersion, 10, 64)
	nextVersion := currentVersion + 1
	if res.ResourceVersion >= currentVersion {
		nextVersion = res.ResourceVersion + 1
	}
	existing.Metadata.ResourceVersion = strconv.FormatInt(nextVersion, 10)
	return mapStoreError(r.store.Apply(ctx, existing))
}

func (r *ResourceRepository[M, S]) Get(ctx context.Context, id string) (*shared.Resource[M, S], error) {
	obj, err := r.store.Get(ctx, id)
	if err != nil {
		return nil, mapStoreError(err)
	}
	resource := toSharedResource(obj)
	return &resource, nil
}

func (r *ResourceRepository[M, S]) UpdateMeta(ctx context.Context, id string, expectedGeneration int64, apply func(*M)) error {
	existing, err := r.store.Get(ctx, id)
	if err != nil {
		return mapStoreError(err)
	}
	if expectedGeneration > 0 && existing.Metadata.Generation != expectedGeneration {
		return fmt.Errorf("%w: requested generation %d doesn't match current %d", ErrConflict, expectedGeneration, existing.Metadata.Generation)
	}
	return mapStoreError(r.store.UpdateSpec(ctx, id, existing.Metadata.ResourceVersion, func(spec *M) error {
		apply(spec)
		return nil
	}))
}

func (r *ResourceRepository[M, S]) UpdateStatus(ctx context.Context, id string, apply func(*S)) error {
	return mapStoreError(r.store.UpdateStatus(ctx, id, "", func(status *S) error {
		apply(status)
		return nil
	}))
}

func (r *ResourceRepository[M, S]) Delete(ctx context.Context, id string) error {
	return mapStoreError(r.store.Delete(ctx, id))
}

func (r *ResourceRepository[M, S]) List(ctx context.Context, cursor string, limit int, filter func(*shared.Resource[M, S]) bool) (*shared.PaginationResponse[shared.Resource[M, S]], error) {
	res, err := r.store.List(ctx, cursor, limit, func(obj *metav1.Object[M, S]) bool {
		if filter == nil {
			return true
		}
		resource := toSharedResource(obj)
		return filter(&resource)
	})
	if err != nil {
		return nil, mapStoreError(err)
	}

	items := make([]shared.Resource[M, S], 0, len(res.Items))
	for i := range res.Items {
		item := res.Items[i]
		items = append(items, toSharedResource(&item))
	}
	return &shared.PaginationResponse[shared.Resource[M, S]]{
		Items:      items,
		NextCursor: res.Metadata.Continue,
		HasMore:    res.Metadata.Continue != "",
	}, nil
}

func (r *ResourceRepository[M, S]) ListAll(ctx context.Context) ([]shared.Resource[M, S], error) {
	return r.ListAllFiltered(ctx, nil)
}

func (r *ResourceRepository[M, S]) ListAllFiltered(ctx context.Context, filter func(*shared.Resource[M, S]) bool) ([]shared.Resource[M, S], error) {
	res, err := r.store.ListAll(ctx, func(obj *metav1.Object[M, S]) bool {
		if filter == nil {
			return true
		}
		resource := toSharedResource(obj)
		return filter(&resource)
	})
	if err != nil {
		return nil, mapStoreError(err)
	}
	items := make([]shared.Resource[M, S], 0, len(res))
	for i := range res {
		item := res[i]
		items = append(items, toSharedResource(&item))
	}
	return items, nil
}

func toSharedResource[M any, S any](obj *metav1.Object[M, S]) shared.Resource[M, S] {
	resourceVersion, _ := strconv.ParseInt(obj.Metadata.ResourceVersion, 10, 64)
	return shared.Resource[M, S]{
		ID:              obj.Metadata.Name,
		Meta:            obj.Spec,
		Status:          obj.Status,
		Generation:      obj.Metadata.Generation,
		ResourceVersion: resourceVersion,
	}
}

func mapStoreError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, store.ErrNotFound):
		return fmt.Errorf("%w: %v", ErrNotFound, err)
	case errors.Is(err, store.ErrBadRequest):
		return fmt.Errorf("%w: %v", ErrBadRequest, err)
	case errors.Is(err, store.ErrConflict):
		return fmt.Errorf("%w: %v", ErrConflict, err)
	case errors.Is(err, store.ErrInvalidConfig):
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	default:
		return err
	}
}
