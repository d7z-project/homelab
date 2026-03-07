package site

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/site"
	"io"
	"path/filepath"
	"time"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
)

func (s *SitePoolService) CreateExport(ctx context.Context, export *models.SiteExport) error {
	export.ID = uuid.NewString()
	export.CreatedAt = time.Now()
	export.UpdatedAt = time.Now()
	return repo.SaveExport(ctx, export)
}

func (s *SitePoolService) UpdateExport(ctx context.Context, export *models.SiteExport) error {
	old, err := repo.GetExport(ctx, export.ID)
	if err != nil {
		return err
	}
	export.CreatedAt = old.CreatedAt
	export.UpdatedAt = time.Now()
	return repo.SaveExport(ctx, export)
}

func (s *SitePoolService) DeleteExport(ctx context.Context, id string) error {
	if s.exportManager != nil {
		s.exportManager.DeleteTasksByExportID(id)
	}
	return repo.DeleteExport(ctx, id)
}

func (s *SitePoolService) GetExport(ctx context.Context, id string) (*models.SiteExport, error) {
	return repo.GetExport(ctx, id)
}

func (s *SitePoolService) ListExports(ctx context.Context, page, pageSize int, search string) ([]models.SiteExport, int, error) {
	return repo.ListExports(ctx, page, pageSize, search)
}

func (s *SitePoolService) PreviewExport(ctx context.Context, req *models.SiteExportPreviewRequest) ([]models.SitePoolEntry, error) {
	program, err := expr.Compile(req.Rule, expr.Env(map[string]interface{}{
		"tags":   []string{},
		"domain": "",
		"type":   uint8(0),
	}))
	if err != nil {
		return nil, fmt.Errorf("%w: compile error: %v", common.ErrBadRequest, err)
	}

	var results []models.SitePoolEntry
	for _, gid := range req.GroupIDs {
		poolPath := filepath.Join(PoolsDir, gid+".bin")
		pf, err := common.FS.Open(poolPath)
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
