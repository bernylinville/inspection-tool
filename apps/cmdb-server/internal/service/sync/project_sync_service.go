package sync

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"inspection-tool/apps/cmdb-server/internal/model"
	"inspection-tool/apps/cmdb-server/internal/repository"
	"inspection-tool/pkg/n9e"
)

// ProjectSyncResult represents the summary of a project sync operation.
type ProjectSyncResult struct {
	TotalProjects   int
	NewProjects     int
	UpdatedProjects int
	FailedProjects  int
	Duration        time.Duration
}

// ProjectSyncService syncs projects from N9E business groups into the CMDB.
type ProjectSyncService struct {
	n9eClient   *n9e.Client
	projectRepo repository.ProjectRepository
	hostRepo    repository.HostRepository
	db          *gorm.DB
	logger      zerolog.Logger
}

// NewProjectSyncService creates a ProjectSyncService.
func NewProjectSyncService(n9eClient *n9e.Client, projectRepo repository.ProjectRepository, hostRepo repository.HostRepository, db *gorm.DB, logger zerolog.Logger) *ProjectSyncService {
	return &ProjectSyncService{
		n9eClient:   n9eClient,
		projectRepo: projectRepo,
		hostRepo:    hostRepo,
		db:          db,
		logger:      logger,
	}
}

// SyncProjects syncs all projects from N9E business groups and returns a summary.
func (s *ProjectSyncService) SyncProjects(ctx context.Context) (*ProjectSyncResult, error) {
	startTime := time.Now()

	groups, err := s.n9eClient.GetBusiGroups(ctx)
	if err != nil {
		return nil, err
	}

	result := &ProjectSyncResult{
		TotalProjects: len(groups),
	}

	for _, group := range groups {
		isNew := false
		existing, err := s.projectRepo.FindByCode(ctx, group.Name)
		if err != nil {
			if isRecordNotFound(err) {
				isNew = true
			} else {
				result.FailedProjects++
				s.logger.Warn().Err(err).Str("group_name", group.Name).Msg("failed to check existing project")
				continue
			}
		} else if existing == nil {
			isNew = true
		}

		if err := s.upsertProject(ctx, group); err != nil {
			result.FailedProjects++
			s.logger.Warn().Err(err).Str("group_name", group.Name).Msg("failed to sync project")
			continue
		}

		if isNew {
			result.NewProjects++
		} else {
			result.UpdatedProjects++
		}
	}

	// Update host_count for all projects
	if err := s.updateProjectHostCounts(ctx); err != nil {
		s.logger.Warn().Err(err).Msg("failed to update project host counts")
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (s *ProjectSyncService) upsertProject(ctx context.Context, group n9e.BusiGroup) error {
	existing, err := s.projectRepo.FindByCode(ctx, group.Name)
	if err != nil && !isRecordNotFound(err) {
		return err
	}

	if existing == nil {
		project := &model.Project{
			Name:       group.Name,
			Code:       group.Name,
			N9EGroupID: group.ID,
			Status:     "active",
		}
		return s.projectRepo.Create(ctx, project)
	}

	existing.N9EGroupID = group.ID
	return s.projectRepo.Update(ctx, existing)
}

func (s *ProjectSyncService) updateProjectHostCounts(ctx context.Context) error {
	sql := `
		UPDATE projects p
		SET host_count = (
			SELECT COUNT(*) FROM hosts h WHERE h.business_group = p.code
		)
	`
	return s.db.WithContext(ctx).Exec(sql).Error
}
