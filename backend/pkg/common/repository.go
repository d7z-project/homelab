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

// BaseRepository is a transitional wrapper over the new resource store.
// It preserves the old call surface while routing persistence through pkg/store.
type BaseRepository[M any, S any] struct {
	store *store.ResourceStore[M, S]
}

// NewBaseRepository creates a repository backed by the new resource store.
func NewBaseRepository[M any, S any](module, model string) *BaseRepository[M, S] {
	resource := strings.ToLower(strings.ReplaceAll(module+"-"+model, "/", "-"))
	return &BaseRepository[M, S]{
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

// Cow executes a copy-on-write mutation using the new resource store.
func (r *BaseRepository[M, S]) Cow(ctx context.Context, id string, mutate func(*shared.Resource[M, S]) error) error {
	return mapStoreError(r.store.Mutate(ctx, id, true, func(obj *metav1.Object[M, S]) error {
		legacy := fromStoreObject(obj)
		if err := mutate(&legacy); err != nil {
			return err
		}
		applyLegacyObject(obj, &legacy)
		return nil
	}))
}

// Save persists a full resource, automatically incrementing ResourceVersion.
func (r *BaseRepository[M, S]) Save(ctx context.Context, res *shared.Resource[M, S]) error {
	if res == nil {
		return fmt.Errorf("%w: resource is required", ErrBadRequest)
	}
	return mapStoreError(r.store.Mutate(ctx, res.ID, true, func(obj *metav1.Object[M, S]) error {
		legacy := fromStoreObject(obj)
		legacy.ID = res.ID
		legacy.Meta = res.Meta
		legacy.Status = res.Status
		legacy.Generation = res.Generation
		legacy.ResourceVersion = res.ResourceVersion + 1
		applyLegacyObject(obj, &legacy)
		return nil
	}))
}

// Get retrieves a resource by ID.
func (r *BaseRepository[M, S]) Get(ctx context.Context, id string) (*shared.Resource[M, S], error) {
	obj, err := r.store.Get(ctx, id)
	if err != nil {
		return nil, mapStoreError(err)
	}
	legacy := fromStoreObject(obj)
	return &legacy, nil
}

// PatchMeta modifies resource spec and increments generation/resourceVersion.
func (r *BaseRepository[M, S]) PatchMeta(ctx context.Context, id string, expectedGeneration int64, apply func(*M)) error {
	return mapStoreError(r.store.Mutate(ctx, id, false, func(obj *metav1.Object[M, S]) error {
		legacy := fromStoreObject(obj)
		if expectedGeneration > 0 && legacy.Generation != expectedGeneration {
			return fmt.Errorf("%w: requested generation %d doesn't match current %d", ErrConflict, expectedGeneration, legacy.Generation)
		}
		apply(&legacy.Meta)
		legacy.Generation++
		legacy.ResourceVersion++
		applyLegacyObject(obj, &legacy)
		return nil
	}))
}

// UpdateStatus modifies resource status and increments only resourceVersion.
func (r *BaseRepository[M, S]) UpdateStatus(ctx context.Context, id string, apply func(*S)) error {
	return mapStoreError(r.store.Mutate(ctx, id, false, func(obj *metav1.Object[M, S]) error {
		legacy := fromStoreObject(obj)
		apply(&legacy.Status)
		legacy.ResourceVersion++
		applyLegacyObject(obj, &legacy)
		return nil
	}))
}

// Delete removes a resource by ID.
func (r *BaseRepository[M, S]) Delete(ctx context.Context, id string) error {
	return mapStoreError(r.store.Delete(ctx, id))
}

// List returns paginated resources using the new store list semantics.
func (r *BaseRepository[M, S]) List(ctx context.Context, cursor string, limit int, filter func(*shared.Resource[M, S]) bool) (*shared.PaginationResponse[shared.Resource[M, S]], error) {
	res, err := r.store.List(ctx, cursor, limit, func(obj *metav1.Object[M, S]) bool {
		if filter == nil {
			return true
		}
		legacy := fromStoreObject(obj)
		return filter(&legacy)
	})
	if err != nil {
		return nil, mapStoreError(err)
	}

	items := make([]shared.Resource[M, S], 0, len(res.Items))
	for i := range res.Items {
		item := res.Items[i]
		items = append(items, fromStoreObject(&item))
	}
	return &shared.PaginationResponse[shared.Resource[M, S]]{
		Items:      items,
		NextCursor: res.Metadata.Continue,
		HasMore:    res.Metadata.Continue != "",
	}, nil
}

// ListAll returns all resources from the new store.
func (r *BaseRepository[M, S]) ListAll(ctx context.Context) ([]shared.Resource[M, S], error) {
	res, err := r.store.ListAll(ctx, nil)
	if err != nil {
		return nil, mapStoreError(err)
	}
	items := make([]shared.Resource[M, S], 0, len(res))
	for i := range res {
		item := res[i]
		items = append(items, fromStoreObject(&item))
	}
	return items, nil
}

func fromStoreObject[M any, S any](obj *metav1.Object[M, S]) shared.Resource[M, S] {
	resourceVersion, _ := strconv.ParseInt(obj.Metadata.ResourceVersion, 10, 64)
	return shared.Resource[M, S]{
		ID:              obj.Metadata.Name,
		Meta:            obj.Spec,
		Status:          obj.Status,
		Generation:      obj.Metadata.Generation,
		ResourceVersion: resourceVersion,
	}
}

func applyLegacyObject[M any, S any](obj *metav1.Object[M, S], legacy *shared.Resource[M, S]) {
	obj.Metadata.Name = legacy.ID
	obj.Metadata.Generation = legacy.Generation
	obj.Metadata.ResourceVersion = strconv.FormatInt(legacy.ResourceVersion, 10)
	obj.Spec = legacy.Meta
	obj.Status = legacy.Status
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
