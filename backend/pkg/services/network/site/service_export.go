package site

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	sitemodel "homelab/pkg/models/network/site"
	"homelab/pkg/models/shared"
	repo "homelab/pkg/repositories/network/site"
	"io"
	"path/filepath"
	"time"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
)

func (s *SitePoolService) CreateExport(ctx context.Context, export *sitemodel.SiteExport) error {
	if err := requireSiteResource(ctx, siteResourceBase); err != nil {
		return err
	}
	export.ID = uuid.NewString()
	export.Status.CreatedAt = time.Now()
	export.Status.UpdatedAt = time.Now()
	err := repo.SaveExport(ctx, export)
	return err
}

func (s *SitePoolService) UpdateExport(ctx context.Context, export *sitemodel.SiteExport) error {
	if err := requireSiteResource(ctx, siteResourceBase); err != nil {
		return err
	}

	current, err := repo.GetExport(ctx, export.ID)
	if err != nil {
		return err
	}
	current.Meta = export.Meta
	current.Status.UpdatedAt = time.Now()
	return repo.SaveExport(ctx, current)
}

func (s *SitePoolService) DeleteExport(ctx context.Context, id string) error {
	if err := requireSiteResource(ctx, siteResourceBase); err != nil {
		return err
	}
	if s.exportManager != nil {
		s.exportManager.DeleteTasksByExportID(id)
	}
	return repo.DeleteExport(ctx, id)
}

func (s *SitePoolService) GetExport(ctx context.Context, id string) (*sitemodel.SiteExport, error) {
	if err := requireSiteResource(ctx, siteResourceBase); err != nil {
		return nil, err
	}
	return repo.GetExport(ctx, id)
}

func (s *SitePoolService) ScanExports(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[sitemodel.SiteExport], error) {
	if err := requireSiteResource(ctx, siteResourceBase); err != nil {
		return nil, err
	}
	return repo.ScanExports(ctx, cursor, limit, search)
}

func (s *SitePoolService) PreviewExport(ctx context.Context, req *sitemodel.SiteExportPreviewRequest) ([]sitemodel.SitePoolEntry, error) {
	program, err := expr.Compile(req.Rule, expr.Env(map[string]interface{}{
		"tags":   []string{},
		"domain": "",
		"type":   uint8(0),
	}))
	if err != nil {
		return nil, fmt.Errorf("%w: compile error: %v", common.ErrBadRequest, err)
	}

	var results []sitemodel.SitePoolEntry
	for _, gid := range req.GroupIDs {
		poolPath := filepath.Join(PoolsDir, gid+".bin")
		pf, err := s.deps.FS.Open(poolPath)
		if err != nil {
			continue
		}
		reader, err := NewReader(pf)
		if err != nil {
			pf.Close()
			continue
		}

		for len(results) < 50 {
			entry, err := reader.Next()
			if err == io.EOF {
				break
			}

			out, err := expr.Run(program, map[string]interface{}{
				"tags":   entry.Tags,
				"domain": entry.Value,
				"type":   entry.Type,
			})

			if err == nil && out == true {
				results = append(results, entry)
			}
		}
		pf.Close()
		if len(results) >= 50 {
			break
		}
	}

	return results, nil
}
