package common

import (
	"context"
	"encoding/json"
	"fmt"
	"homelab/pkg/models"
	"time"

	"gopkg.d7z.net/middleware/kv"
)

// BaseRepository provides standard operations for generic resources, implementing OCC and validation.
type BaseRepository[M any, S any] struct {
	db     kv.KV
	module string
	model  string
	prefix []string
}

// NewBaseRepository creates a new BaseRepository with the K8s-style storage path: {module}/{model}/v1.
func NewBaseRepository[M any, S any](module, model string) *BaseRepository[M, S] {
	return &BaseRepository[M, S]{
		db:     DB,
		module: module,
		model:  model,
		prefix: []string{module, model, "v1"},
	}
}

// childDB returns the scoped KV for this repository.
func (r *BaseRepository[M, S]) childDB() kv.KV {
	if r.db != nil {
		return r.db.Child(r.prefix...)
	}
	return DB.Child(r.prefix...)
}

// Cow executes a Copy-on-Write operation with optimistic concurrency control.
func (r *BaseRepository[M, S]) Cow(ctx context.Context, id string, mutate func(*models.Resource[M, S]) error) error {
	db := r.childDB()
	retries := 10

	for i := 0; i < retries; i++ {
		oldData, err := db.Get(ctx, id)
		if err != nil && err.Error() != kv.ErrKeyNotFound.Error() {
			// Try to handle 'not found' properly
			if !isNotFound(err) {
				return fmt.Errorf("failed to get resource for Cow: %w", err)
			}
			oldData = ""
		}

		var res models.Resource[M, S]
		if oldData != "" {
			if err := json.Unmarshal([]byte(oldData), &res); err != nil {
				return fmt.Errorf("failed to unmarshal resource: %w", err)
			}
		} else {
			res.ID = id
		}

		if err := mutate(&res); err != nil {
			return err
		}

		newData, err := json.Marshal(res)
		if err != nil {
			return fmt.Errorf("failed to marshal resource: %w", err)
		}

		if oldData == "" {
			// Creation
			success, err := db.PutIfNotExists(ctx, id, string(newData), kv.TTLKeep)
			if err != nil {
				return fmt.Errorf("failed to create resource: %w", err)
			}
			if success {
				return nil
			}
		} else {
			// Update
			success, err := db.CompareAndSwap(ctx, id, oldData, string(newData))
			if err != nil {
				return fmt.Errorf("failed to update resource: %w", err)
			}
			if success {
				return nil
			}
		}

		// CAS failed, retry
		// Wait a bit before retrying, maybe add jitter
		time.Sleep(time.Duration(10+i*5) * time.Millisecond)
	}

	return fmt.Errorf("%w: failed to update resource %s after %d retries", ErrConflict, id, retries)
}

func isNotFound(err error) bool {
	// Simple heuristic since kv.ErrKeyNotFound is used, but error wrapping could obfuscate it
	if err == nil {
		return false
	}
	return err.Error() == kv.ErrKeyNotFound.Error() || err.Error() == "key not found"
}

// Get retrieves a resource by ID.
func (r *BaseRepository[M, S]) Get(ctx context.Context, id string) (*models.Resource[M, S], error) {
	data, err := r.childDB().Get(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var res models.Resource[M, S]
	if err := json.Unmarshal([]byte(data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// PatchMeta modifies the configuration (Meta) of a resource.
// Validates expected generation if provided (>0).
// Automatically runs ConfigValidator if implemented by the Meta payload.
func (r *BaseRepository[M, S]) PatchMeta(ctx context.Context, id string, expectedGeneration int64, apply func(*M)) error {
	return r.Cow(ctx, id, func(res *models.Resource[M, S]) error {
		if expectedGeneration > 0 && res.Generation != expectedGeneration {
			return fmt.Errorf("%w: requested generation %d doesn't match current %d",
				ErrConflict, expectedGeneration, res.Generation)
		}

		apply(&res.Meta)

		// Type assertion for validation
		if validator, ok := any(res.Meta).(models.ConfigValidator); ok {
			if err := validator.Validate(ctx); err != nil {
				return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
			}
		}

		res.Generation++
		res.ResourceVersion++

		return nil
	})
}

// UpdateStatus modifies the running state (Status) of a resource.
// Does not validate generation, increments only ResourceVersion.
func (r *BaseRepository[M, S]) UpdateStatus(ctx context.Context, id string, apply func(*S)) error {
	return r.Cow(ctx, id, func(res *models.Resource[M, S]) error {
		apply(&res.Status)

		res.ResourceVersion++

		return nil
	})
}

// Delete removes a resource by ID.
func (r *BaseRepository[M, S]) Delete(ctx context.Context, id string) error {
	success, err := r.childDB().Delete(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return ErrNotFound
		}
		return err
	}
	if !success {
		return ErrNotFound
	}
	return nil
}

// List returns a paginated list of resources using cursor pagination.
func (r *BaseRepository[M, S]) List(ctx context.Context, cursor string, limit int, filter func(*models.Resource[M, S]) bool) (*models.PaginationResponse[models.Resource[M, S]], error) {
	db := r.childDB()
	count, _ := db.Count(ctx)

	fetchLimit := limit * 2
	if fetchLimit < 50 {
		fetchLimit = 50
	}

	resp, err := db.ListCurrentCursor(ctx, &kv.ListOptions{
		Limit:  int64(fetchLimit),
		Cursor: cursor,
	})
	if err != nil {
		return nil, err
	}

	res := make([]models.Resource[M, S], 0)
	for _, v := range resp.Pairs {
		var item models.Resource[M, S]
		if err := json.Unmarshal([]byte(v.Value), &item); err == nil {
			if filter == nil || filter(&item) {
				res = append(res, item)
			}
		}

		// If we've reached exactly the limit, return right away, and set the cursor to this item's key.
		if len(res) == limit {
			return &models.PaginationResponse[models.Resource[M, S]]{
				Items:      res,
				NextCursor: v.Key,
				HasMore:    resp.HasMore || len(resp.Pairs) > 0,
				Total:      int64(count),
			}, nil
		}
	}

	// Reached the end of fetched data, didn't hit limit exactly.
	return &models.PaginationResponse[models.Resource[M, S]]{
		Items:      res,
		NextCursor: resp.Cursor,
		HasMore:    resp.HasMore,
		Total:      int64(count),
	}, nil
}
