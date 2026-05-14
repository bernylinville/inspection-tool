package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"inspection-tool/internal/client/vm"
	"inspection-tool/internal/config"
)

func TestQueryResultsWithHostFilterFallback_RetriesWithTagsOnly(t *testing.T) {
	requests := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		requests = append(requests, query)

		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(query, "busigroup") {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
			return
		}

		if !strings.Contains(query, `items="重庆传媒数字乡村-电信侧"`) {
			t.Fatalf("fallback query did not keep items tag: %s", query)
		}

		timestamp := time.Now().Unix()
		_, _ = fmt.Fprintf(w, `{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{"metric": {"__name__": "mysql_up", "address": "10.0.0.1:3306"}, "value": [%d, "1"]}
				]
			}
		}`, timestamp)
	}))
	defer server.Close()

	vmClient := vm.NewClient(
		&config.VictoriaMetricsConfig{Endpoint: server.URL},
		&config.RetryConfig{MaxRetries: 0},
		zerolog.Nop(),
	)

	results, err := queryResultsWithHostFilterFallback(
		context.Background(),
		vmClient,
		zerolog.Nop(),
		"mysql_up == 1",
		&vm.HostFilter{
			BusinessGroups: []string{"重数传媒数字乡村电信侧"},
			Tags:           map[string]string{"items": "重庆传媒数字乡村-电信侧"},
		},
	)
	if err != nil {
		t.Fatalf("queryResultsWithHostFilterFallback() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("result count = %d, want 1", len(results))
	}
	if len(requests) != 2 {
		t.Fatalf("request count = %d, want 2 (%#v)", len(requests), requests)
	}
	if !strings.Contains(requests[0], "busigroup") {
		t.Fatalf("first query should contain busigroup: %s", requests[0])
	}
	if strings.Contains(requests[1], "busigroup") {
		t.Fatalf("fallback query should omit busigroup: %s", requests[1])
	}
	if !strings.Contains(requests[1], `items="重庆传媒数字乡村-电信侧"`) {
		t.Fatalf("fallback query should keep items tag: %s", requests[1])
	}
}
