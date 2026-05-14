package service

import (
	"context"

	"github.com/rs/zerolog"

	"inspection-tool/internal/client/vm"
)

// queryResultsWithHostFilterFallback executes a VM query with the requested
// filter and retries with tags-only when a business-group + tag filter matches
// no series. Some deployments tag VM metrics with items but do not expose the
// busigroup label, while N9E still uses business groups for inventory.
func queryResultsWithHostFilterFallback(
	ctx context.Context,
	vmClient *vm.Client,
	logger zerolog.Logger,
	query string,
	filter *vm.HostFilter,
) ([]vm.QueryResult, error) {
	results, err := vmClient.QueryResultsWithFilter(ctx, query, filter)
	if err != nil {
		return nil, err
	}
	if !shouldRetryWithTagsOnly(filter, results) {
		return results, nil
	}

	logger.Debug().
		Strs("business_groups", filter.BusinessGroups).
		Msg("VM filter matched no results with business groups and tags; retrying with tags only")

	return vmClient.QueryResultsWithFilter(ctx, query, tagsOnlyHostFilter(filter))
}

func shouldRetryWithTagsOnly(filter *vm.HostFilter, results []vm.QueryResult) bool {
	return filter != nil &&
		len(filter.BusinessGroups) > 0 &&
		len(filter.Tags) > 0 &&
		len(results) == 0
}
