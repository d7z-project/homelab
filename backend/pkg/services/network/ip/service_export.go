package ip

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	ipmodel "homelab/pkg/models/network/ip"
	"homelab/pkg/models/shared"
	repo "homelab/pkg/repositories/network/ip"
	"io"
	"path/filepath"
	"time"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
)

func (s *IPPoolService) CreateExport(ctx context.Context, export *ipmodel.IPExport) error {
	if err := requireIPResource(ctx, ipResourceBase); err != nil {
		return err
	}
	export.ID = uuid.NewString()
	export.Status.CreatedAt = time.Now()
	export.Status.UpdatedAt = time.Now()
	err := repo.SaveExport(ctx, export)
	commonaudit.FromContext(ctx).Log("CreateIPExport", export.Meta.Name, "Created", err == nil)
	return err
}

func (s *IPPoolService) UpdateExport(ctx context.Context, export *ipmodel.IPExport) error {
	if err := requireIPResource(ctx, ipExportResource(export.ID)); err != nil {
		return err
	}

	current, err := repo.GetExport(ctx, export.ID)
	if err == nil {
		current.Meta = export.Meta
		current.Status.UpdatedAt = time.Now()
		err = repo.SaveExport(ctx, current)
	}
	commonaudit.FromContext(ctx).Log("UpdateIPExport", export.Meta.Name, "Updated", err == nil)
	return err
}

func (s *IPPoolService) DeleteExport(ctx context.Context, id string) error {
	if err := requireIPResource(ctx, ipExportResource(id)); err != nil {
		return err
	}

	// 级联删除相关的任务和物理文件
	if s.exportManager != nil {
		s.exportManager.DeleteTasksByExportID(id)
	}
	err := repo.DeleteExport(ctx, id)
	commonaudit.FromContext(ctx).Log("DeleteIPExport", id, "Deleted", err == nil)
	return err
}

func (s *IPPoolService) GetExport(ctx context.Context, id string) (*ipmodel.IPExport, error) {
	if err := requireIPResourceOrGlobal(ctx, ipExportResource(id)); err != nil {
		return nil, err
	}
	return repo.GetExport(ctx, id)
}

func (s *IPPoolService) ScanExports(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[ipmodel.IPExport], error) {
	perms := commonauth.PermissionsFromContext(ctx)
	hasGlobal := perms.IsAllowed(ipResourceBase)
	res, err := repo.ScanExports(ctx, cursor, limit, search)
	if err != nil {
		return nil, err
	}
	if hasGlobal {
		return res, nil
	}
	filtered := make([]ipmodel.IPExport, 0, len(res.Items))
	for _, item := range res.Items {
		if perms.IsAllowed(ipExportResource(item.ID)) {
			filtered = append(filtered, item)
		}
	}
	res.Items = filtered
	return res, nil
}

func (s *IPPoolService) PreviewExport(ctx context.Context, req *ipmodel.IPExportPreviewRequest) ([]ipmodel.IPPoolEntry, error) {
	program, err := expr.Compile(req.Rule, expr.Env(map[string]interface{}{
		"tags": []string{},
		"cidr": "",
		"ip":   "",
	}))
	if err != nil {
		return nil, fmt.Errorf("%w: compile error: %v", common.ErrBadRequest, err)
	}

	var results []ipmodel.IPPoolEntry
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
				results = append(results, ipmodel.IPPoolEntry{
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
