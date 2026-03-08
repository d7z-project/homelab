package ip

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/ip"
	"io"
	"path/filepath"
	"time"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
)

func (s *IPPoolService) CreateExport(ctx context.Context, export *models.IPExport) error {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("network/ip") {
		return fmt.Errorf("%w: network/ip", commonauth.ErrPermissionDenied)
	}
	if export.ID == "" {
		export.ID = uuid.NewString()
	}
	export.CreatedAt = time.Now()
	export.UpdatedAt = time.Now()
	err := repo.SaveExport(ctx, export)
	commonaudit.FromContext(ctx).Log("CreateIPExport", export.Name, "Created", err == nil)
	return err
}

func (s *IPPoolService) UpdateExport(ctx context.Context, export *models.IPExport) error {
	resource := "network/ip/export/" + export.ID
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	old, err := repo.GetExport(ctx, export.ID)
	if err != nil {
		return err
	}
	export.CreatedAt = old.CreatedAt
	export.UpdatedAt = time.Now()
	err = repo.SaveExport(ctx, export)
	commonaudit.FromContext(ctx).Log("UpdateIPExport", export.Name, "Updated", err == nil)
	return err
}

func (s *IPPoolService) DeleteExport(ctx context.Context, id string) error {
	resource := "network/ip/export/" + id
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	// 级联删除相关的任务和物理文件
	if s.exportManager != nil {
		s.exportManager.DeleteTasksByExportID(id)
	}
	err := repo.DeleteExport(ctx, id)
	commonaudit.FromContext(ctx).Log("DeleteIPExport", id, "Deleted", err == nil)
	return err
}

func (s *IPPoolService) GetExport(ctx context.Context, id string) (*models.IPExport, error) {
	return repo.GetExport(ctx, id)
}

func (s *IPPoolService) ScanExports(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.IPExport], error) {
	return repo.ScanExports(ctx, cursor, limit, search)
}

func (s *IPPoolService) PreviewExport(ctx context.Context, req *models.IPExportPreviewRequest) ([]models.IPPoolEntry, error) {
	program, err := expr.Compile(req.Rule, expr.Env(map[string]interface{}{
		"tags": []string{},
		"cidr": "",
		"ip":   "",
	}))
	if err != nil {
		return nil, fmt.Errorf("%w: compile error: %v", common.ErrBadRequest, err)
	}

	var results []models.IPPoolEntry
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
			prefix, tags, err := reader.Next()
			if err == io.EOF {
				break
			}

			output, err := expr.Run(program, map[string]interface{}{
				"tags": tags,
				"cidr": prefix.String(),
				"ip":   prefix.Addr().String(),
			})

			if err == nil && output == true {
				results = append(results, models.IPPoolEntry{
					CIDR: prefix.String(),
					Tags: tags,
				})
			}
		}
		pf.Close()
		if len(results) >= 50 {
			break
		}
	}

	return results, nil
}

func (s *IPPoolService) LookupExport(ctx context.Context, id string) (interface{}, error) {
	return repo.GetExport(ctx, id)
}
