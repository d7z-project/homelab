package intelligence

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/intelligence"
	"homelab/pkg/services/ip"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

func (s *IntelligenceService) SyncSource(ctx context.Context, id string) error {
	source, err := repo.GetSource(ctx, id)
	if err != nil {
		return err
	}

	source.Status = "Downloading"
	_ = repo.SaveSource(ctx, source)

	go s.runDownload(id)

	commonaudit.FromContext(ctx).Log("SyncIntelligence", source.Name, "Started", true)
	return nil
}

func (s *IntelligenceService) lockSync(ctx context.Context, id string) (func(), error) {
	return common.LockWithTimeout(ctx, "network:intelligence:sync:"+id, 0)
}

func (s *IntelligenceService) runDownload(id string) {
	ctx := context.Background()
	release, err := s.lockSync(ctx, id)
	if err != nil {
		log.Printf("IntelligenceService: failed to acquire lock for %s: %v", id, err)
		return
	}
	defer release()

	source, _ := repo.GetSource(ctx, id)
	if source == nil {
		return
	}

	err = s.downloadFile(source)
	if err != nil {
		source.Status = "Error"
		source.ErrorMessage = err.Error()
	} else {
		source.Status = "Ready"
		source.ErrorMessage = ""
		source.LastUpdatedAt = time.Now()
		common.UpdateGlobalVersion(ctx, "network/intelligence/mmdb")
		common.NotifyCluster(ctx, "mmdb_update", source.Type)
	}

	_ = repo.SaveSource(ctx, source)
}

func (s *IntelligenceService) downloadFile(source *models.IntelligenceSource) error {
	resp, err := http.Get(source.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}

	var targetPath string
	switch source.Type {
	case "asn":
		targetPath = ip.MMDBPathASN
	case "city":
		targetPath = ip.MMDBPathCity
	case "country":
		targetPath = ip.MMDBPathCountry
	default:
		return fmt.Errorf("invalid type")
	}

	_ = common.FS.MkdirAll(filepath.Dir(targetPath), 0755)
	f, err := common.FS.Create(targetPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

