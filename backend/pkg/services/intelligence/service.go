package intelligence

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/intelligence"
	"homelab/pkg/services/ip"
	"homelab/pkg/services/rbac"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

var (
	ErrSourceNotFound = fmt.Errorf("%w: intelligence source not found", common.ErrNotFound)
)

func init() {
	rbac.RegisterResourceWithVerbs("network/intelligence", func(ctx context.Context, prefix string) ([]models.DiscoverResult, error) {
		return []models.DiscoverResult{}, nil
	}, []string{"list", "create", "update", "delete", "execute", "*"})
}

type IntelligenceService struct {
	mmdb    *ip.MMDBManager
	cron    *cron.Cron
	entries map[string]cron.EntryID
	mu      sync.Mutex
}

func NewIntelligenceService(mmdb *ip.MMDBManager) *IntelligenceService {
	s := &IntelligenceService{
		mmdb:    mmdb,
		cron:    cron.New(),
		entries: make(map[string]cron.EntryID),
	}
	s.cron.Start()
	return s
}

func (s *IntelligenceService) Init(ctx context.Context) error {
	sources, err := repo.ListSources(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range sources {
		src := &sources[i]
		// Reset "Downloading" status if stuck from previous run
		if src.Status == "Downloading" {
			src.Status = "Error"
			src.ErrorMessage = "Interrupted by system restart"
			_ = repo.SaveSource(ctx, src)
		}

		if src.AutoUpdate && src.UpdateCron != "" {
			s.addCronJob(*src)
		}
	}
	log.Printf("IntelligenceService: initialized and cleaned up stuck tasks")
	return nil
}

func (s *IntelligenceService) addCronJob(src models.IntelligenceSource) {
	id := src.ID
	entryID, err := s.cron.AddFunc(src.UpdateCron, func() {
		log.Printf("IntelligenceService: running scheduled update for %s (%s)", src.Name, src.ID)
		s.runDownload(id)
	})
	if err != nil {
		log.Printf("IntelligenceService: failed to schedule job for %s: %v", src.Name, err)
		return
	}
	s.entries[id] = entryID
}

func (s *IntelligenceService) updateCronJob(src models.IntelligenceSource) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing if any
	if entryID, ok := s.entries[src.ID]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, src.ID)
	}

	// Add new if enabled
	if src.AutoUpdate && src.UpdateCron != "" {
		s.addCronJob(src)
	}
}

func (s *IntelligenceService) removeCronJob(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, ok := s.entries[id]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, id)
	}
}

func (s *IntelligenceService) CreateSource(ctx context.Context, source *models.IntelligenceSource) error {
	source.ID = uuid.NewString()
	source.Status = "Ready"
	source.LastUpdatedAt = time.Time{}
	if err := repo.SaveSource(ctx, source); err != nil {
		return err
	}
	s.updateCronJob(*source)
	return nil
}

func (s *IntelligenceService) UpdateSource(ctx context.Context, source *models.IntelligenceSource) error {
	existing, err := repo.GetSource(ctx, source.ID)
	if err != nil {
		return ErrSourceNotFound
	}
	// Preserve immutable or managed fields
	source.Status = existing.Status
	source.LastUpdatedAt = existing.LastUpdatedAt
	source.ErrorMessage = existing.ErrorMessage

	if err := repo.SaveSource(ctx, source); err != nil {
		return err
	}

	s.updateCronJob(*source)
	commonaudit.FromContext(ctx).Log("UpdateIntelligence", source.Name, "Success", true)
	return nil
}

func (s *IntelligenceService) ListSources(ctx context.Context) ([]models.IntelligenceSource, error) {
	return repo.ListSources(ctx)
}

func (s *IntelligenceService) DeleteSource(ctx context.Context, id string) error {
	if err := repo.DeleteSource(ctx, id); err != nil {
		return err
	}
	s.removeCronJob(id)
	return nil
}

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

func (s *IntelligenceService) runDownload(id string) {
	ctx := context.Background()
	source, _ := repo.GetSource(ctx, id)
	if source == nil {
		return
	}

	err := s.downloadFile(source)
	if err != nil {
		source.Status = "Error"
		source.ErrorMessage = err.Error()
	} else {
		source.Status = "Ready"
		source.ErrorMessage = ""
		source.LastUpdatedAt = time.Now()
		_ = s.mmdb.Reload()
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
