package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	metav1 "homelab/pkg/apis/meta/v1"

	"gopkg.d7z.net/middleware/kv"
)

type Resource[Spec any, Status any] = metav1.Object[Spec, Status]
type ResourceList[Spec any, Status any] = metav1.List[Resource[Spec, Status]]

type SpecValidator[Spec any] func(ctx context.Context, spec *Spec) error
type ResourceFilter[Spec any, Status any] func(obj *Resource[Spec, Status]) bool
type SpecMutator[Spec any] func(spec *Spec) error
type StatusMutator[Status any] func(status *Status) error
type ResourceMutator[Spec any, Status any] func(obj *Resource[Spec, Status]) error

var (
	ErrNotFound      = errors.New("resource not found")
	ErrBadRequest    = errors.New("bad request")
	ErrConflict      = errors.New("resource conflict")
	ErrInvalidConfig = errors.New("invalid configuration")
)

type ResourceStore[Spec any, Status any] struct {
	db           kv.KV
	prefix       []string
	apiVersion   string
	kind         string
	listKind     string
	validateSpec SpecValidator[Spec]
}

func NewResourceStore[Spec any, Status any](db kv.KV, apiVersion string, kind string, resource string, validateSpec SpecValidator[Spec]) *ResourceStore[Spec, Status] {
	return &ResourceStore[Spec, Status]{
		db:           db,
		prefix:       []string{"resources", apiVersion, strings.ToLower(resource)},
		apiVersion:   apiVersion,
		kind:         kind,
		listKind:     kind + "List",
		validateSpec: validateSpec,
	}
}

func (s *ResourceStore[Spec, Status]) childDB(ctx context.Context) kv.KV {
	if s.db == nil {
		panic("resource store db is not configured in context")
	}
	return s.db.Child(s.prefix...)
}

func (s *ResourceStore[Spec, Status]) Create(ctx context.Context, obj *Resource[Spec, Status]) error {
	if obj == nil {
		return fmt.Errorf("%w: resource is required", ErrBadRequest)
	}
	name, err := normalizeName(obj.Metadata.Name)
	if err != nil {
		return err
	}
	obj.TypeMeta.APIVersion = s.apiVersion
	obj.TypeMeta.Kind = s.kind
	obj.Metadata.Name = name
	obj.Metadata.Generation = 1
	obj.Metadata.ResourceVersion = "1"
	if obj.Metadata.CreationTimestamp.IsZero() {
		obj.Metadata.CreationTimestamp = time.Now().UTC()
	}
	if err := s.validate(obj); err != nil {
		return err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	created, err := s.childDB(ctx).PutIfNotExists(ctx, name, string(data), kv.TTLKeep)
	if err != nil {
		return err
	}
	if !created {
		return fmt.Errorf("%w: resource %s already exists", ErrConflict, name)
	}
	return nil
}

func (s *ResourceStore[Spec, Status]) Get(ctx context.Context, name string) (*Resource[Spec, Status], error) {
	key, err := normalizeName(name)
	if err != nil {
		return nil, err
	}
	data, err := s.childDB(ctx).Get(ctx, key)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	obj := new(Resource[Spec, Status])
	if err := json.Unmarshal([]byte(data), obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func (s *ResourceStore[Spec, Status]) List(ctx context.Context, cursor string, limit int, filter ResourceFilter[Spec, Status]) (*ResourceList[Spec, Status], error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	fetchLimit := limit * 2
	if fetchLimit < 50 {
		fetchLimit = 50
	}

	resp, err := s.childDB(ctx).ListCurrentCursor(ctx, &kv.ListOptions{
		Limit:  int64(fetchLimit),
		Cursor: cursor,
	})
	if err != nil {
		return nil, err
	}

	res := &ResourceList[Spec, Status]{
		TypeMeta: metav1.TypeMeta{
			APIVersion: s.apiVersion,
			Kind:       s.listKind,
		},
		Metadata: metav1.ListMeta{},
		Items:    make([]Resource[Spec, Status], 0, limit),
	}

	for _, pair := range resp.Pairs {
		var item Resource[Spec, Status]
		if err := json.Unmarshal([]byte(pair.Value), &item); err != nil {
			return nil, err
		}
		if filter != nil && !filter(&item) {
			continue
		}
		res.Items = append(res.Items, item)
		if len(res.Items) == limit {
			res.Metadata.Continue = pair.Key
			return res, nil
		}
	}

	res.Metadata.Continue = resp.Cursor
	return res, nil
}

func (s *ResourceStore[Spec, Status]) UpdateSpec(ctx context.Context, name string, expectedResourceVersion string, mutate SpecMutator[Spec]) error {
	return s.compareAndSwap(ctx, name, func(obj *Resource[Spec, Status]) error {
		if expectedResourceVersion != "" && obj.Metadata.ResourceVersion != expectedResourceVersion {
			return fmt.Errorf("%w: requested resourceVersion %s does not match current %s", ErrConflict, expectedResourceVersion, obj.Metadata.ResourceVersion)
		}
		if err := mutate(&obj.Spec); err != nil {
			return err
		}
		obj.Metadata.Generation++
		obj.Metadata.ResourceVersion = nextResourceVersion(obj.Metadata.ResourceVersion)
		return s.validate(obj)
	})
}

func (s *ResourceStore[Spec, Status]) UpdateStatus(ctx context.Context, name string, expectedResourceVersion string, mutate StatusMutator[Status]) error {
	return s.compareAndSwap(ctx, name, func(obj *Resource[Spec, Status]) error {
		if expectedResourceVersion != "" && obj.Metadata.ResourceVersion != expectedResourceVersion {
			return fmt.Errorf("%w: requested resourceVersion %s does not match current %s", ErrConflict, expectedResourceVersion, obj.Metadata.ResourceVersion)
		}
		if err := mutate(&obj.Status); err != nil {
			return err
		}
		obj.Metadata.ResourceVersion = nextResourceVersion(obj.Metadata.ResourceVersion)
		return nil
	})
}

func (s *ResourceStore[Spec, Status]) Delete(ctx context.Context, name string) error {
	key, err := normalizeName(name)
	if err != nil {
		return err
	}
	deleted, err := s.childDB(ctx).Delete(ctx, key)
	if err != nil {
		if isNotFound(err) {
			return ErrNotFound
		}
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	return nil
}

func (s *ResourceStore[Spec, Status]) Apply(ctx context.Context, obj *Resource[Spec, Status]) error {
	if obj == nil {
		return fmt.Errorf("%w: resource is required", ErrBadRequest)
	}
	name, err := normalizeName(obj.Metadata.Name)
	if err != nil {
		return err
	}
	obj.TypeMeta.APIVersion = s.apiVersion
	obj.TypeMeta.Kind = s.kind
	obj.Metadata.Name = name
	if err := s.validate(obj); err != nil {
		return err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return s.childDB(ctx).Put(ctx, name, string(data), kv.TTLKeep)
}

func (s *ResourceStore[Spec, Status]) Mutate(ctx context.Context, name string, createIfMissing bool, mutate ResourceMutator[Spec, Status]) error {
	key, err := normalizeName(name)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 10; attempt++ {
		current, err := s.childDB(ctx).Get(ctx, key)
		if err != nil && !isNotFound(err) {
			return err
		}
		exists := err == nil
		if !exists && !createIfMissing {
			return ErrNotFound
		}

		obj := &Resource[Spec, Status]{
			TypeMeta: metav1.TypeMeta{
				APIVersion: s.apiVersion,
				Kind:       s.kind,
			},
			Metadata: metav1.ObjectMeta{
				Name: key,
			},
		}
		if exists {
			if err := json.Unmarshal([]byte(current), obj); err != nil {
				return err
			}
		} else {
			obj.Metadata.CreationTimestamp = time.Now().UTC()
		}

		if err := mutate(obj); err != nil {
			return err
		}
		obj.TypeMeta.APIVersion = s.apiVersion
		obj.TypeMeta.Kind = s.kind
		obj.Metadata.Name = key
		if err := s.validate(obj); err != nil {
			return err
		}

		updated, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		if !exists {
			ok, err := s.childDB(ctx).PutIfNotExists(ctx, key, string(updated), kv.TTLKeep)
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
		} else {
			ok, err := s.childDB(ctx).CompareAndSwap(ctx, key, current, string(updated))
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
		}
		time.Sleep(time.Duration(10+attempt*5) * time.Millisecond)
	}
	return fmt.Errorf("%w: failed to update resource %s after retries", ErrConflict, key)
}

func (s *ResourceStore[Spec, Status]) ListAll(ctx context.Context, filter ResourceFilter[Spec, Status]) ([]Resource[Spec, Status], error) {
	items, err := s.childDB(ctx).List(ctx, "")
	if err != nil {
		return nil, err
	}
	res := make([]Resource[Spec, Status], 0, len(items))
	for _, pair := range items {
		var item Resource[Spec, Status]
		if err := json.Unmarshal([]byte(pair.Value), &item); err != nil {
			return nil, err
		}
		if filter != nil && !filter(&item) {
			continue
		}
		res = append(res, item)
	}
	return res, nil
}

func (s *ResourceStore[Spec, Status]) compareAndSwap(ctx context.Context, name string, mutate func(*Resource[Spec, Status]) error) error {
	key, err := normalizeName(name)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 10; attempt++ {
		current, err := s.childDB(ctx).Get(ctx, key)
		if err != nil {
			if isNotFound(err) {
				return ErrNotFound
			}
			return err
		}

		var obj Resource[Spec, Status]
		if err := json.Unmarshal([]byte(current), &obj); err != nil {
			return err
		}
		if err := mutate(&obj); err != nil {
			return err
		}

		updated, err := json.Marshal(&obj)
		if err != nil {
			return err
		}
		ok, err := s.childDB(ctx).CompareAndSwap(ctx, key, current, string(updated))
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		time.Sleep(time.Duration(10+attempt*5) * time.Millisecond)
	}
	return fmt.Errorf("%w: failed to update resource %s after retries", ErrConflict, key)
}

func (s *ResourceStore[Spec, Status]) validate(obj *Resource[Spec, Status]) error {
	if obj.Metadata.Labels == nil {
		obj.Metadata.Labels = map[string]string{}
	}
	if obj.Metadata.Annotations == nil {
		obj.Metadata.Annotations = map[string]string{}
	}
	if s.validateSpec != nil {
		if err := s.validateSpec(context.Background(), &obj.Spec); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
		}
	}
	return nil
}

func normalizeName(name string) (string, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return "", fmt.Errorf("%w: metadata.name is required", ErrBadRequest)
	}
	return name, nil
}

func nextResourceVersion(current string) string {
	if current == "" {
		return "1"
	}
	version, err := strconv.ParseInt(current, 10, 64)
	if err != nil {
		return "1"
	}
	return strconv.FormatInt(version+1, 10)
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == kv.ErrKeyNotFound.Error() || err.Error() == "key not found"
}
