package sync

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"gorm.io/datatypes"

	"inspection-tool/apps/cmdb-server/internal/model"
	"inspection-tool/apps/cmdb-server/internal/repository"
	"inspection-tool/pkg/n9e"
)

// SyncResult represents the summary of a sync operation.
type SyncResult struct {
	TotalHosts   int
	NewHosts     int
	UpdatedHosts int
	FailedHosts  int
	Duration     time.Duration
}

// HostSyncService syncs hosts from N9E into the CMDB.
type HostSyncService struct {
	n9eClient *n9e.Client
	hostRepo  repository.HostRepository
	logger    zerolog.Logger
}

// NewHostSyncService creates a HostSyncService.
func NewHostSyncService(n9eClient *n9e.Client, hostRepo repository.HostRepository, logger zerolog.Logger) *HostSyncService {
	return &HostSyncService{
		n9eClient: n9eClient,
		hostRepo:  hostRepo,
		logger:    logger,
	}
}

// SyncHosts syncs all hosts from N9E and returns a summary.
func (s *HostSyncService) SyncHosts(ctx context.Context) (*SyncResult, error) {
	startTime := time.Now()

	targets, err := s.n9eClient.GetTargets(ctx)
	if err != nil {
		return nil, err
	}

	result := &SyncResult{
		TotalHosts: len(targets),
	}

	const maxConcurrency = int64(10)
	sem := semaphore.NewWeighted(maxConcurrency)
	group, groupCtx := errgroup.WithContext(ctx)
	var mu sync.Mutex

	for _, target := range targets {
		target := target
		group.Go(func() error {
			if err := sem.Acquire(groupCtx, 1); err != nil {
				return err
			}
			defer sem.Release(1)

			detail := target
			detailTarget, err := s.n9eClient.GetTarget(groupCtx, target.Ident)
			if err != nil {
				s.logger.Warn().Err(err).Str("ident", target.Ident).Msg("failed to fetch target detail, using basic info")
			} else if detailTarget != nil {
				detail = *detailTarget
			}

			isNew := false
			existing, err := s.hostRepo.FindByIdent(groupCtx, detail.Ident)
			if err != nil {
				if isRecordNotFound(err) {
					isNew = true
				} else {
					mu.Lock()
					result.FailedHosts++
					mu.Unlock()
					s.logger.Warn().Err(err).Str("ident", detail.Ident).Msg("failed to check existing host")
					return nil
				}
			} else if existing == nil {
				isNew = true
			}

			if err := s.syncHost(groupCtx, detail); err != nil {
				mu.Lock()
				result.FailedHosts++
				mu.Unlock()
				s.logger.Warn().Err(err).Str("ident", detail.Ident).Msg("failed to sync host")
				return nil
			}

			mu.Lock()
			if isNew {
				result.NewHosts++
			} else {
				result.UpdatedHosts++
			}
			mu.Unlock()

			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (s *HostSyncService) syncHost(ctx context.Context, target n9e.TargetData) error {
	hostMeta, err := target.ToHostMeta()
	if err != nil {
		return err
	}
	if hostMeta == nil {
		return errors.New("host meta is nil")
	}

	businessGroup := ""
	if len(target.TagsMaps) > 0 {
		if items, ok := target.TagsMaps["items"]; ok {
			businessGroup = items
		}
	}

	var tagsJSON datatypes.JSON
	if len(target.TagsMaps) > 0 {
		data, err := json.Marshal(target.TagsMaps)
		if err == nil {
			tagsJSON = datatypes.JSON(data)
		}
	}

	now := time.Now()
	existing, err := s.hostRepo.FindByIdent(ctx, hostMeta.Ident)
	if err != nil {
		if !isRecordNotFound(err) {
			return err
		}

		host := &model.Host{
			Ident:         hostMeta.Ident,
			Hostname:      hostMeta.Hostname,
			IP:            hostMeta.IP,
			OS:            hostMeta.OS,
			OSVersion:     hostMeta.OSVersion,
			KernelVersion: hostMeta.KernelVersion,
			CPUCores:      hostMeta.CPUCores,
			CPUModel:      hostMeta.CPUModel,
			MemoryTotal:   hostMeta.MemoryTotal,
			BusinessGroup: businessGroup,
			Tags:          tagsJSON,
			LastSyncAt:    &now,
		}

		return s.hostRepo.Create(ctx, host)
	}

	if existing == nil {
		host := &model.Host{
			Ident:         hostMeta.Ident,
			Hostname:      hostMeta.Hostname,
			IP:            hostMeta.IP,
			OS:            hostMeta.OS,
			OSVersion:     hostMeta.OSVersion,
			KernelVersion: hostMeta.KernelVersion,
			CPUCores:      hostMeta.CPUCores,
			CPUModel:      hostMeta.CPUModel,
			MemoryTotal:   hostMeta.MemoryTotal,
			BusinessGroup: businessGroup,
			Tags:          tagsJSON,
			LastSyncAt:    &now,
		}

		return s.hostRepo.Create(ctx, host)
	}

	existing.Ident = hostMeta.Ident
	existing.Hostname = hostMeta.Hostname
	existing.IP = hostMeta.IP
	existing.OS = hostMeta.OS
	existing.OSVersion = hostMeta.OSVersion
	existing.KernelVersion = hostMeta.KernelVersion
	existing.CPUCores = hostMeta.CPUCores
	existing.CPUModel = hostMeta.CPUModel
	existing.MemoryTotal = hostMeta.MemoryTotal
	existing.BusinessGroup = businessGroup
	existing.Tags = tagsJSON
	existing.LastSyncAt = &now

	return s.hostRepo.Update(ctx, existing)
}

func isRecordNotFound(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "record not found"
}
