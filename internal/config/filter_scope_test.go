package config

import (
	"os"
	"reflect"
	"testing"
)

func TestLoad_AppliesHostFilterScopeToMiddlewareInstanceFilters(t *testing.T) {
	content := `
datasources:
  n9e:
    endpoint: "http://localhost:17000"
    token: "test-token"
  victoriametrics:
    endpoint: "http://localhost:8428"
inspection:
  host_filter:
    business_groups:
      - "重庆传媒数字乡村"
    tags:
      items: "重庆传媒数字乡村-电信侧"
mysql:
  enabled: true
  cluster_mode: "mgr"
  instance_filter:
    address_patterns:
      - "10.0.0.*"
redis:
  enabled: true
nginx:
  enabled: true
tomcat:
  enabled: true
elasticsearch:
  enabled: true
`

	cfg := loadConfigFromContent(t, content)

	wantGroups := []string{"重庆传媒数字乡村"}
	wantTags := map[string]string{"items": "重庆传媒数字乡村-电信侧"}

	if !reflect.DeepEqual(cfg.MySQL.InstanceFilter.BusinessGroups, wantGroups) {
		t.Fatalf("MySQL groups = %#v, want %#v", cfg.MySQL.InstanceFilter.BusinessGroups, wantGroups)
	}
	if !reflect.DeepEqual(cfg.MySQL.InstanceFilter.Tags, wantTags) {
		t.Fatalf("MySQL tags = %#v, want %#v", cfg.MySQL.InstanceFilter.Tags, wantTags)
	}
	if !reflect.DeepEqual(cfg.Redis.InstanceFilter.BusinessGroups, wantGroups) {
		t.Fatalf("Redis groups = %#v, want %#v", cfg.Redis.InstanceFilter.BusinessGroups, wantGroups)
	}
	if !reflect.DeepEqual(cfg.Redis.InstanceFilter.Tags, wantTags) {
		t.Fatalf("Redis tags = %#v, want %#v", cfg.Redis.InstanceFilter.Tags, wantTags)
	}
	if !reflect.DeepEqual(cfg.Nginx.InstanceFilter.BusinessGroups, wantGroups) {
		t.Fatalf("Nginx groups = %#v, want %#v", cfg.Nginx.InstanceFilter.BusinessGroups, wantGroups)
	}
	if !reflect.DeepEqual(cfg.Nginx.InstanceFilter.Tags, wantTags) {
		t.Fatalf("Nginx tags = %#v, want %#v", cfg.Nginx.InstanceFilter.Tags, wantTags)
	}
	if !reflect.DeepEqual(cfg.Tomcat.InstanceFilter.BusinessGroups, wantGroups) {
		t.Fatalf("Tomcat groups = %#v, want %#v", cfg.Tomcat.InstanceFilter.BusinessGroups, wantGroups)
	}
	if !reflect.DeepEqual(cfg.Tomcat.InstanceFilter.Tags, wantTags) {
		t.Fatalf("Tomcat tags = %#v, want %#v", cfg.Tomcat.InstanceFilter.Tags, wantTags)
	}
	if !reflect.DeepEqual(cfg.Elasticsearch.InstanceFilter.BusinessGroups, wantGroups) {
		t.Fatalf("Elasticsearch groups = %#v, want %#v", cfg.Elasticsearch.InstanceFilter.BusinessGroups, wantGroups)
	}
	if !reflect.DeepEqual(cfg.Elasticsearch.InstanceFilter.Tags, wantTags) {
		t.Fatalf("Elasticsearch tags = %#v, want %#v", cfg.Elasticsearch.InstanceFilter.Tags, wantTags)
	}

	if !reflect.DeepEqual(cfg.MySQL.InstanceFilter.AddressPatterns, []string{"10.0.0.*"}) {
		t.Fatalf("MySQL address patterns were not preserved: %#v", cfg.MySQL.InstanceFilter.AddressPatterns)
	}
}

func TestLoad_HostFilterScopePreservesExplicitDomainLabels(t *testing.T) {
	content := `
datasources:
  n9e:
    endpoint: "http://localhost:17000"
    token: "test-token"
  victoriametrics:
    endpoint: "http://localhost:8428"
inspection:
  host_filter:
    business_groups:
      - "global-scope"
    tags:
      items: "global-items"
      env: "prod"
mysql:
  enabled: true
  cluster_mode: "mgr"
  instance_filter:
    business_groups:
      - "mysql-scope"
    tags:
      items: "mysql-items"
      app: "mysql"
`

	cfg := loadConfigFromContent(t, content)

	if !reflect.DeepEqual(cfg.MySQL.InstanceFilter.BusinessGroups, []string{"mysql-scope"}) {
		t.Fatalf("explicit MySQL groups were overwritten: %#v", cfg.MySQL.InstanceFilter.BusinessGroups)
	}
	wantTags := map[string]string{
		"items": "mysql-items",
		"app":   "mysql",
		"env":   "prod",
	}
	if !reflect.DeepEqual(cfg.MySQL.InstanceFilter.Tags, wantTags) {
		t.Fatalf("MySQL tags = %#v, want %#v", cfg.MySQL.InstanceFilter.Tags, wantTags)
	}
}

func loadConfigFromContent(t *testing.T, content string) *Config {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	return cfg
}
