package inspection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"

	"inspection-tool/apps/cmdb-server/internal/model"
	"inspection-tool/apps/cmdb-server/internal/repository"
)

var (
	ErrJobNotFound      = errors.New("inspection job not found")
	ErrInvalidJobType   = errors.New("invalid inspection job type")
	ErrJobAlreadyExists = errors.New("inspection job already exists")
	ErrCLINotFound      = errors.New("inspect-cli binary not found")
	ErrJobRunning       = errors.New("job is currently running")
)

type CreateJobRequest struct {
	Type        string `json:"type" binding:"required"`
	TriggerType string `json:"trigger_type"`
	ProjectID   int64  `json:"project_id"`
	ProjectCode string `json:"project_code"`
	CreatedBy   string `json:"created_by"`
}

type JobSummary struct {
	TotalHosts    int `json:"total_hosts,omitempty"`
	NormalHosts   int `json:"normal_hosts,omitempty"`
	WarningHosts  int `json:"warning_hosts,omitempty"`
	CriticalHosts int `json:"critical_hosts,omitempty"`
	FailedHosts   int `json:"failed_hosts,omitempty"`

	TotalInstances    int `json:"total_instances,omitempty"`
	NormalInstances   int `json:"normal_instances,omitempty"`
	WarningInstances  int `json:"warning_instances,omitempty"`
	CriticalInstances int `json:"critical_instances,omitempty"`
	FailedInstances   int `json:"failed_instances,omitempty"`
}

type InspectService struct {
	jobRepo    repository.InspectionJobRepository
	cliPath    string
	configPath string
	outputDir  string
	logger     zerolog.Logger
}

func NewInspectService(
	jobRepo repository.InspectionJobRepository,
	cliPath string,
	configPath string,
	outputDir string,
	logger zerolog.Logger,
) *InspectService {
	return &InspectService{
		jobRepo:    jobRepo,
		cliPath:    cliPath,
		configPath: configPath,
		outputDir:  outputDir,
		logger:     logger,
	}
}

func (s *InspectService) CreateJob(ctx context.Context, req *CreateJobRequest) (*model.InspectionJob, error) {
	if !isValidJobType(req.Type) {
		return nil, ErrInvalidJobType
	}

	triggerType := req.TriggerType
	if triggerType == "" {
		triggerType = model.TriggerTypeManual
	}

	job := &model.InspectionJob{
		Type:        req.Type,
		TriggerType: triggerType,
		Status:      model.JobStatusPending,
		ProjectID:   req.ProjectID,
		ProjectCode: req.ProjectCode,
		CreatedBy:   req.CreatedBy,
		CreatedAt:   time.Now(),
	}

	if err := s.jobRepo.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	go s.executeJob(job.ID)

	return job, nil
}

func (s *InspectService) GetJob(ctx context.Context, id int64) (*model.InspectionJob, error) {
	job, err := s.jobRepo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrJobNotFound
	}
	return job, nil
}

func (s *InspectService) ListJobs(ctx context.Context, opts repository.ListOptions) ([]model.InspectionJob, int64, error) {
	return s.jobRepo.List(ctx, opts)
}

func (s *InspectService) ListJobsByStatus(ctx context.Context, status string) ([]model.InspectionJob, error) {
	return s.jobRepo.ListByStatus(ctx, status)
}

func (s *InspectService) ListJobsByType(ctx context.Context, jobType string) ([]model.InspectionJob, error) {
	return s.jobRepo.ListByType(ctx, jobType)
}

func (s *InspectService) DeleteJob(ctx context.Context, id int64) error {
	job, err := s.jobRepo.FindByID(ctx, id)
	if err != nil {
		return ErrJobNotFound
	}

	if job.Status == model.JobStatusRunning {
		return ErrJobRunning
	}

	return s.jobRepo.Delete(ctx, id)
}

func (s *InspectService) UpdateStatus(ctx context.Context, id int64, status string, errMsg string) error {
	job, err := s.jobRepo.FindByID(ctx, id)
	if err != nil {
		return ErrJobNotFound
	}

	job.Status = status
	if errMsg != "" {
		job.ErrorMessage = errMsg
	}

	if status == model.JobStatusRunning {
		now := time.Now()
		job.StartTime = &now
	}

	if status == model.JobStatusSuccess || status == model.JobStatusFailed {
		now := time.Now()
		job.EndTime = &now
		if job.StartTime != nil {
			job.DurationSeconds = int(now.Sub(*job.StartTime).Seconds())
		}
	}

	return s.jobRepo.Update(ctx, job)
}

func (s *InspectService) executeJob(jobID int64) {
	ctx := context.Background()

	if err := s.UpdateStatus(ctx, jobID, model.JobStatusRunning, ""); err != nil {
		s.logger.Error().Err(err).Int64("job_id", jobID).Msg("failed to update job status to running")
		return
	}

	job, err := s.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to load job: %v", err)
		s.logger.Error().Err(err).Int64("job_id", jobID).Msg("failed to load job before execution")
		if updateErr := s.UpdateStatus(ctx, jobID, model.JobStatusFailed, errMsg); updateErr != nil {
			s.logger.Error().Err(updateErr).Int64("job_id", jobID).Msg("failed to update job status to failed")
		}
		return
	}

	var configOverride string
	if job.ProjectCode != "" {
		configOverride, err = s.generateJobConfig(jobID, job.ProjectCode)
		if err != nil {
			errMsg := fmt.Sprintf("failed to generate project config: %v", err)
			s.logger.Error().
				Err(err).
				Int64("job_id", jobID).
				Str("project_code", job.ProjectCode).
				Msg("failed to generate temporary config")
			if updateErr := s.UpdateStatus(ctx, jobID, model.JobStatusFailed, errMsg); updateErr != nil {
				s.logger.Error().Err(updateErr).Int64("job_id", jobID).Msg("failed to update job status to failed")
			}
			return
		}
		defer func() {
			if removeErr := os.Remove(configOverride); removeErr != nil {
				s.logger.Warn().
					Err(removeErr).
					Int64("job_id", jobID).
					Str("path", configOverride).
					Msg("failed to remove temporary config file")
			}
		}()
	}

	args := s.buildCLIArgs(jobID, configOverride)

	s.logger.Info().
		Int64("job_id", jobID).
		Str("cli_path", s.cliPath).
		Strs("args", args).
		Msg("starting inspection job")

	cmd := exec.Command(s.cliPath, args...)
	cmd.Dir = "/home/kchou/Code/inspection-tool"
	output, err := cmd.CombinedOutput()

	if err != nil {
		errMsg := fmt.Sprintf("CLI execution failed: %v, output: %s", err, string(output))
		s.logger.Error().
			Err(err).
			Int64("job_id", jobID).
			Str("output", string(output)).
			Msg("inspection job failed")

		if updateErr := s.UpdateStatus(ctx, jobID, model.JobStatusFailed, errMsg); updateErr != nil {
			s.logger.Error().Err(updateErr).Int64("job_id", jobID).Msg("failed to update job status to failed")
		}
		return
	}

	if err := s.updateJobResults(ctx, jobID); err != nil {
		s.logger.Error().Err(err).Int64("job_id", jobID).Msg("failed to update job results")
		return
	}

	s.logger.Info().Int64("job_id", jobID).Msg("inspection job completed successfully")
}

func (s *InspectService) buildCLIArgs(jobID int64, configOverride string) []string {
	job, _ := s.jobRepo.FindByID(context.Background(), jobID)

	args := []string{"run"}

	configPath := s.configPath
	if configOverride != "" {
		configPath = configOverride
	}
	if configPath != "" {
		args = append(args, "-c", configPath)
	}

	if s.outputDir != "" {
		args = append(args, "-o", s.outputDir)
	}

	if job != nil {
		switch job.Type {
		case model.InspectionTypeHost:
			args = append(args, "--skip-mysql", "--skip-redis")
		case model.InspectionTypeMySQL:
			args = append(args, "--mysql-only")
		case model.InspectionTypeRedis:
			args = append(args, "--redis-only")
		}
	}

	return args
}

func (s *InspectService) generateJobConfig(jobID int64, projectCode string) (string, error) {
	if s.configPath == "" {
		return "", fmt.Errorf("inspection config path is empty")
	}

	content, err := os.ReadFile(s.configPath)
	if err != nil {
		return "", fmt.Errorf("read base config: %w", err)
	}

	cfg := make(map[string]interface{})
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return "", fmt.Errorf("unmarshal base config: %w", err)
	}

	businessGroups := []string{projectCode}

	inspectionCfg := ensureMap(cfg, "inspection")
	hostFilterCfg := ensureMap(inspectionCfg, "host_filter")
	hostFilterCfg["business_groups"] = businessGroups

	for _, section := range []string{"mysql", "redis", "nginx", "tomcat", "elasticsearch"} {
		sectionCfg := ensureMap(cfg, section)
		instanceFilterCfg := ensureMap(sectionCfg, "instance_filter")
		instanceFilterCfg["business_groups"] = businessGroups
	}

	updatedContent, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal updated config: %w", err)
	}

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("inspection_job_%d_*.yaml", jobID))
	if err != nil {
		return "", fmt.Errorf("create temp config file: %w", err)
	}

	if _, err := tmpFile.Write(updatedContent); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("write temp config file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("close temp config file: %w", err)
	}

	return tmpFile.Name(), nil
}

func ensureMap(root map[string]interface{}, key string) map[string]interface{} {
	if root == nil {
		return map[string]interface{}{}
	}

	value, ok := root[key]
	if !ok {
		next := make(map[string]interface{})
		root[key] = next
		return next
	}

	if valueMap, ok := value.(map[string]interface{}); ok {
		return valueMap
	}

	next := make(map[string]interface{})
	root[key] = next
	return next
}

func (s *InspectService) updateJobResults(ctx context.Context, jobID int64) error {
	job, err := s.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102_150405")
	baseFilename := fmt.Sprintf("inspection_report_%s_%d", timestamp, jobID)

	job.ReportExcelPath = filepath.Join(s.outputDir, baseFilename+".xlsx")
	job.ReportHTMLPath = filepath.Join(s.outputDir, baseFilename+".html")
	job.Status = model.JobStatusSuccess

	now := time.Now()
	job.EndTime = &now
	if job.StartTime != nil {
		job.DurationSeconds = int(now.Sub(*job.StartTime).Seconds())
	}

	summary := &JobSummary{}
	summaryJSON, _ := json.Marshal(summary)
	job.Summary = summaryJSON

	return s.jobRepo.Update(ctx, job)
}

func isValidJobType(jobType string) bool {
	validTypes := []string{
		model.InspectionTypeHost,
		model.InspectionTypeMySQL,
		model.InspectionTypeRedis,
		model.InspectionTypeNginx,
		model.InspectionTypeTomcat,
		model.InspectionTypeElasticsearch,
	}

	for _, t := range validTypes {
		if jobType == t {
			return true
		}
	}
	return false
}
