package discovery

import (
	"context"
	"errors"

	commonauth "homelab/pkg/common/auth"
	discoverymodel "homelab/pkg/models/core/discovery"
	"homelab/pkg/models/shared"
	"homelab/pkg/runtime"
	registryruntime "homelab/pkg/runtime/registry"
)

type Service struct {
	registry *registryruntime.Registry
}

func NewService(deps runtime.ModuleDeps) *Service {
	return &Service{registry: deps.Registry}
}

func (s *Service) Lookup(ctx context.Context, req discoverymodel.LookupRequest) (*shared.PaginationResponse[discoverymodel.LookupItem], error) {
	if s.registry == nil {
		return nil, errors.New("registry not configured")
	}
	return s.registry.Lookup(ctx, req)
}

func (s *Service) ScanCodes(ctx context.Context) ([]string, error) {
	if s.registry == nil {
		return nil, errors.New("registry not configured")
	}
	codes := s.registry.ScanCodes()
	visible := make([]string, 0, len(codes))
	for _, code := range codes {
		_, err := s.registry.Lookup(ctx, discoverymodel.LookupRequest{
			Code:  code,
			Limit: 1,
		})
		if err != nil {
			if errors.Is(err, commonauth.ErrPermissionDenied) || errors.Is(err, commonauth.ErrUnauthorized) {
				continue
			}
			return nil, err
		}
		visible = append(visible, code)
	}
	return visible, nil
}

func (s *Service) SuggestResources(ctx context.Context, prefix string) ([]discoverymodel.DiscoverResult, error) {
	if s.registry == nil {
		return nil, errors.New("registry not configured")
	}
	return s.registry.SuggestResources(ctx, prefix)
}

func (s *Service) SuggestVerbs(ctx context.Context, resource string) ([]string, error) {
	if s.registry == nil {
		return nil, errors.New("registry not configured")
	}
	return s.registry.SuggestVerbs(ctx, resource)
}

func (s *Service) CheckSAUsage(ctx context.Context, id string) error {
	if s.registry == nil {
		return errors.New("registry not configured")
	}
	return s.registry.CheckSAUsage(ctx, id)
}
