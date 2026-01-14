package sync

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"inspection-tool/apps/cmdb-server/internal/model"
	"inspection-tool/apps/cmdb-server/internal/repository"
	"inspection-tool/pkg/vm"
)

// DiscoveryResult 发现结果统计
type DiscoveryResult struct {
	MySQL         int
	Redis         int
	Nginx         int
	Tomcat        int
	Elasticsearch int
	Duration      time.Duration
}

// InstanceDiscoveryService 中间件实例发现服务
type InstanceDiscoveryService struct {
	vmClient   *vm.Client
	hostRepo   repository.HostRepository
	mysqlRepo  repository.MySQLInstanceRepository
	redisRepo  repository.RedisInstanceRepository
	nginxRepo  repository.NginxInstanceRepository
	tomcatRepo repository.TomcatInstanceRepository
	esRepo     repository.ElasticsearchClusterRepository
	logger     zerolog.Logger
}

// NewInstanceDiscoveryService 构造函数
func NewInstanceDiscoveryService(
	vmClient *vm.Client,
	hostRepo repository.HostRepository,
	mysqlRepo repository.MySQLInstanceRepository,
	redisRepo repository.RedisInstanceRepository,
	nginxRepo repository.NginxInstanceRepository,
	tomcatRepo repository.TomcatInstanceRepository,
	esRepo repository.ElasticsearchClusterRepository,
	logger zerolog.Logger,
) *InstanceDiscoveryService {
	return &InstanceDiscoveryService{
		vmClient:   vmClient,
		hostRepo:   hostRepo,
		mysqlRepo:  mysqlRepo,
		redisRepo:  redisRepo,
		nginxRepo:  nginxRepo,
		tomcatRepo: tomcatRepo,
		esRepo:     esRepo,
		logger:     logger,
	}
}

// DiscoverAll 发现所有中间件实例
func (s *InstanceDiscoveryService) DiscoverAll(ctx context.Context) (*DiscoveryResult, error) {
	startTime := time.Now()
	result := &DiscoveryResult{}

	// 发现各类中间件
	if count, err := s.DiscoverMySQL(ctx); err == nil {
		result.MySQL = count
	}
	if count, err := s.DiscoverRedis(ctx); err == nil {
		result.Redis = count
	}
	if count, err := s.DiscoverNginx(ctx); err == nil {
		result.Nginx = count
	}
	if count, err := s.DiscoverTomcat(ctx); err == nil {
		result.Tomcat = count
	}
	if count, err := s.DiscoverElasticsearch(ctx); err == nil {
		result.Elasticsearch = count
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// DiscoverMySQL 发现 MySQL 实例
func (s *InstanceDiscoveryService) DiscoverMySQL(ctx context.Context) (int, error) {
	results, err := s.vmClient.QueryResults(ctx, "mysql_up")
	if err != nil {
		return 0, err
	}

	count := 0
	for _, r := range results {
		address := r.Labels["address"]
		if address == "" {
			continue
		}

		// 查找关联主机
		var hostID *int64
		if ident := r.Ident; ident != "" {
			if host, err := s.hostRepo.FindByIdent(ctx, ident); err == nil && host != nil {
				hostID = &host.ID
			}
		}

		// 查找或创建实例
		existing, _ := s.mysqlRepo.FindByAddress(ctx, address)
		now := time.Now()
		if existing == nil {
			instance := &model.MySQLInstance{
				Address:    address,
				HostID:     hostID,
				Version:    r.Labels["version"],
				LastSyncAt: &now,
			}
			if err := s.mysqlRepo.Create(ctx, instance); err == nil {
				count++
			}
		} else {
			existing.HostID = hostID
			existing.Version = r.Labels["version"]
			existing.LastSyncAt = &now
			s.mysqlRepo.Update(ctx, existing)
			count++
		}
	}
	return count, nil
}

// DiscoverRedis 发现 Redis 实例
func (s *InstanceDiscoveryService) DiscoverRedis(ctx context.Context) (int, error) {
	results, err := s.vmClient.QueryResults(ctx, "redis_up")
	if err != nil {
		return 0, err
	}

	count := 0
	for _, r := range results {
		address := r.Labels["address"]
		if address == "" {
			continue
		}

		var hostID *int64
		if ident := r.Ident; ident != "" {
			if host, err := s.hostRepo.FindByIdent(ctx, ident); err == nil && host != nil {
				hostID = &host.ID
			}
		}

		existing, _ := s.redisRepo.FindByAddress(ctx, address)
		now := time.Now()
		if existing == nil {
			instance := &model.RedisInstance{
				Address:    address,
				HostID:     hostID,
				Version:    r.Labels["version"],
				LastSyncAt: &now,
			}
			if err := s.redisRepo.Create(ctx, instance); err == nil {
				count++
			}
		} else {
			existing.HostID = hostID
			existing.Version = r.Labels["version"]
			existing.LastSyncAt = &now
			s.redisRepo.Update(ctx, existing)
			count++
		}
	}
	return count, nil
}

// DiscoverNginx 发现 Nginx 实例
func (s *InstanceDiscoveryService) DiscoverNginx(ctx context.Context) (int, error) {
	results, err := s.vmClient.QueryResults(ctx, "nginx_up")
	if err != nil {
		return 0, err
	}

	count := 0
	for _, r := range results {
		address := r.Labels["address"]
		if address == "" {
			continue
		}

		var hostID *int64
		if ident := r.Ident; ident != "" {
			if host, err := s.hostRepo.FindByIdent(ctx, ident); err == nil && host != nil {
				hostID = &host.ID
			}
		}

		existing, _ := s.nginxRepo.FindByAddress(ctx, address)
		now := time.Now()
		if existing == nil {
			instance := &model.NginxInstance{
				Address:    address,
				HostID:     hostID,
				Version:    r.Labels["version"],
				LastSyncAt: &now,
			}
			if err := s.nginxRepo.Create(ctx, instance); err == nil {
				count++
			}
		} else {
			existing.HostID = hostID
			existing.Version = r.Labels["version"]
			existing.LastSyncAt = &now
			s.nginxRepo.Update(ctx, existing)
			count++
		}
	}
	return count, nil
}

// DiscoverTomcat 发现 Tomcat 实例
func (s *InstanceDiscoveryService) DiscoverTomcat(ctx context.Context) (int, error) {
	results, err := s.vmClient.QueryResults(ctx, "tomcat_up")
	if err != nil {
		return 0, err
	}

	count := 0
	for _, r := range results {
		address := r.Labels["address"]
		if address == "" {
			continue
		}

		var hostID *int64
		if ident := r.Ident; ident != "" {
			if host, err := s.hostRepo.FindByIdent(ctx, ident); err == nil && host != nil {
				hostID = &host.ID
			}
		}

		existing, _ := s.tomcatRepo.FindByAddress(ctx, address)
		now := time.Now()
		if existing == nil {
			instance := &model.TomcatInstance{
				Address:    address,
				HostID:     hostID,
				Version:    r.Labels["version"],
				LastSyncAt: &now,
			}
			if err := s.tomcatRepo.Create(ctx, instance); err == nil {
				count++
			}
		} else {
			existing.HostID = hostID
			existing.Version = r.Labels["version"]
			existing.LastSyncAt = &now
			s.tomcatRepo.Update(ctx, existing)
			count++
		}
	}
	return count, nil
}

// DiscoverElasticsearch 发现 Elasticsearch 集群
func (s *InstanceDiscoveryService) DiscoverElasticsearch(ctx context.Context) (int, error) {
	results, err := s.vmClient.QueryResults(ctx, "elasticsearch_cluster_health_status")
	if err != nil {
		return 0, err
	}

	count := 0
	for _, r := range results {
		clusterName := r.Labels["cluster"]
		if clusterName == "" {
			continue
		}

		existing, _ := s.esRepo.FindByClusterName(ctx, clusterName)
		now := time.Now()
		if existing == nil {
			cluster := &model.ElasticsearchCluster{
				ClusterName: clusterName,
				Version:     r.Labels["version"],
				LastSyncAt:  &now,
			}
			if err := s.esRepo.Create(ctx, cluster); err == nil {
				count++
			}
		} else {
			existing.Version = r.Labels["version"]
			existing.LastSyncAt = &now
			s.esRepo.Update(ctx, existing)
			count++
		}
	}
	return count, nil
}
