package sync

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/rs/zerolog"
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

	for _, target := range targets {
		isNew := false
		existing, err := s.hostRepo.FindByIdent(ctx, target.Ident)
		if err != nil {
			if isRecordNotFound(err) {
				isNew = true
			} else {
				result.FailedHosts++
				s.logger.Warn().Err(err).Str("ident", target.Ident).Msg("failed to check existing host")
				continue
			}
		} else if existing == nil {
			isNew = true
		}

		if err := s.syncHost(ctx, target); err != nil {
			result.FailedHosts++
			s.logger.Warn().Err(err).Str("ident", target.Ident).Msg("failed to sync host")
			continue
		}

		if isNew {
			result.NewHosts++
		} else {
			result.UpdatedHosts++
		}
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
