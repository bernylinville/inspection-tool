package config

// applyHostFilterScope makes inspection.host_filter the default VM label scope
// for middleware domains. Domain-level instance_filter values remain the place
// to further narrow by address, hostname, container, name, or explicit labels.
func applyHostFilterScope(cfg *Config) {
	if cfg == nil {
		return
	}

	hostFilter := cfg.Inspection.HostFilter
	if len(hostFilter.BusinessGroups) == 0 && len(hostFilter.Tags) == 0 {
		return
	}

	inheritBusinessGroups(&cfg.MySQL.InstanceFilter.BusinessGroups, hostFilter.BusinessGroups)
	inheritTags(&cfg.MySQL.InstanceFilter.Tags, hostFilter.Tags)

	inheritBusinessGroups(&cfg.Redis.InstanceFilter.BusinessGroups, hostFilter.BusinessGroups)
	inheritTags(&cfg.Redis.InstanceFilter.Tags, hostFilter.Tags)

	inheritBusinessGroups(&cfg.Nginx.InstanceFilter.BusinessGroups, hostFilter.BusinessGroups)
	inheritTags(&cfg.Nginx.InstanceFilter.Tags, hostFilter.Tags)

	inheritBusinessGroups(&cfg.Tomcat.InstanceFilter.BusinessGroups, hostFilter.BusinessGroups)
	inheritTags(&cfg.Tomcat.InstanceFilter.Tags, hostFilter.Tags)

	inheritBusinessGroups(&cfg.Elasticsearch.InstanceFilter.BusinessGroups, hostFilter.BusinessGroups)
	inheritTags(&cfg.Elasticsearch.InstanceFilter.Tags, hostFilter.Tags)
}

func inheritBusinessGroups(target *[]string, defaults []string) {
	if target == nil || len(*target) > 0 || len(defaults) == 0 {
		return
	}
	*target = append([]string(nil), defaults...)
}

func inheritTags(target *map[string]string, defaults map[string]string) {
	if target == nil || len(defaults) == 0 {
		return
	}

	if *target == nil {
		*target = make(map[string]string, len(defaults))
	}

	for key, value := range defaults {
		if _, exists := (*target)[key]; !exists {
			(*target)[key] = value
		}
	}
}
