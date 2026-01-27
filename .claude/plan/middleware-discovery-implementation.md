# Middleware Discovery Endpoint - Implementation Plan

**Date**: 2026-01-22
**Issue**: CMDB Dashboard showing zero data for middleware instances
**Status**: Implementation plan ready for execution

---

## Executive Summary

### Problem
The CMDB dashboard at `http://127.0.0.1:8088/#/cmdb/dashboard` displays zeros for all middleware counts:
- MySQL: 0
- Redis: 0
- Nginx: 0
- Tomcat: 0
- Elasticsearch: 0

### Root Cause
1. ✅ **Hosts synced**: 808 hosts successfully imported from N9E
2. ❌ **Middleware discovery missing**: `InstanceDiscoveryService` exists but has no API endpoint

### Solution
Add `POST /api/v1/middleware/discover` endpoint to trigger middleware instance discovery from VictoriaMetrics.

---

## Current Status

### Database State (After Host Sync)

```sql
-- Assets with data ✅
hosts: 808 rows

-- Assets still empty ❌
mysql_instances: 0 rows
redis_instances: 0 rows
nginx_instances: 0 rows
tomcat_instances: 0 rows
elasticsearch_clusters: 0 rows

-- User-managed (expected to be empty)
projects: 0 rows
applications: 0 rows
```

### Services Status

| Component | Status | Details |
|-----------|--------|---------|
| Backend | ✅ Running | Port 8080, PID varies |
| Frontend | ✅ Running | Port 8088 |
| PostgreSQL | ✅ Running | Port 5432 |
| Redis | ✅ Running | Port 6379 |
| Authentication | ✅ Working | JWT tokens valid |
| Host Sync | ✅ Working | 808 hosts imported |
| Middleware Discovery | ❌ Missing | No API endpoint |

---

## Implementation Details

### Overview

**Files to CREATE**: 1
- `apps/cmdb-server/internal/api/handler/middleware.go`

**Files to MODIFY**: 2
- `apps/cmdb-server/internal/api/router/router.go`
- `apps/cmdb-server/cmd/main.go`

**Total Changes**: 3 files, ~150 lines of code

---

## File 1: CREATE Handler - `middleware.go`

**Location**: `apps/cmdb-server/internal/api/handler/middleware.go`

**Purpose**: Handle HTTP requests for middleware discovery

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"inspection-tool/apps/cmdb-server/internal/service/sync"
)

// ==================== Response Structs ====================

type DiscoverMiddlewareResponse struct {
	Code    int                       `json:"code"`
	Message string                    `json:"message"`
	Data    *DiscoverMiddlewareResult `json:"data,omitempty"`
}

type DiscoverMiddlewareResult struct {
	MySQL         int    `json:"mysql"`
	Redis         int    `json:"redis"`
	Nginx         int    `json:"nginx"`
	Tomcat        int    `json:"tomcat"`
	Elasticsearch int    `json:"elasticsearch"`
	Duration      string `json:"duration"`
}

// ==================== Handler ====================

type MiddlewareHandler struct {
	discoveryService *sync.InstanceDiscoveryService
}

func NewMiddlewareHandler(discoveryService *sync.InstanceDiscoveryService) *MiddlewareHandler {
	return &MiddlewareHandler{
		discoveryService: discoveryService,
	}
}

// DiscoverInstances triggers middleware instance discovery from VictoriaMetrics
// @Summary Discover middleware instances
// @Description Discovers MySQL, Redis, Nginx, Tomcat, and Elasticsearch instances from VictoriaMetrics metrics
// @Tags middleware
// @Accept json
// @Produce json
// @Success 200 {object} DiscoverMiddlewareResponse
// @Failure 500 {object} DiscoverMiddlewareResponse
// @Router /api/v1/middleware/discover [post]
// @Security BearerAuth
func (h *MiddlewareHandler) DiscoverInstances(c *gin.Context) {
	ctx := c.Request.Context()

	// Trigger discovery for all middleware types
	result, err := h.discoveryService.DiscoverAll(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, DiscoverMiddlewareResponse{
			Code:    5001,
			Message: "Failed to discover middleware instances: " + err.Error(),
		})
		return
	}

	// Convert result to response format
	c.JSON(http.StatusOK, DiscoverMiddlewareResponse{
		Code:    0,
		Message: "middleware discovery completed",
		Data: &DiscoverMiddlewareResult{
			MySQL:         result.MySQL,
			Redis:         result.Redis,
			Nginx:         result.Nginx,
			Tomcat:        result.Tomcat,
			Elasticsearch: result.Elasticsearch,
			Duration:      result.Duration.String(),
		},
	})
}
```

---

## File 2: MODIFY Router - `router.go`

**Location**: `apps/cmdb-server/internal/api/router/router.go`

### Change 2.1: Add Middleware to Handlers struct

**Line**: ~22-30

**FIND:**
```go
type Handlers struct {
	Auth       *handler.AuthHandler
	User       *handler.UserHandler
	Role       *handler.RoleHandler
	Asset      *handler.AssetHandler
	Monitor    *handler.MonitorHandler
	Alert      *handler.AlertHandler
	Inspection *handler.InspectionHandler
}
```

**REPLACE WITH:**
```go
type Handlers struct {
	Auth       *handler.AuthHandler
	User       *handler.UserHandler
	Role       *handler.RoleHandler
	Asset      *handler.AssetHandler
	Monitor    *handler.MonitorHandler
	Alert      *handler.AlertHandler
	Inspection *handler.InspectionHandler
	Middleware *handler.MiddlewareHandler  // ← NEW
}
```

### Change 2.2: Add discovery route

**Line**: ~232-243 (after `setupMiddlewareRoutes(rg)`)

**FIND:**
```go
	r.setupMiddlewareRoutes(rg)

	monitor := rg.Group("/monitor")
```

**INSERT BETWEEN:**
```go
	r.setupMiddlewareRoutes(rg)

	// Middleware discovery endpoint
	middleware := rg.Group("/middleware")
	{
		if r.handlers.Middleware != nil {
			middleware.POST("/discover", r.handlers.Middleware.DiscoverInstances)
		} else {
			middleware.POST("/discover", placeholder("discover middleware instances"))
		}
	}

	monitor := rg.Group("/monitor")
```

---

## File 3: MODIFY Main - `main.go`

**Location**: `apps/cmdb-server/cmd/main.go`

### Change 3.1: Initialize InstanceDiscoveryService

**Line**: ~118-120 (after `hostSyncService`)

**FIND:**
```go
	hostSyncService := sync.NewHostSyncService(n9eClient, hostRepo, log)

	assetService := asset.NewAssetService(db, projectRepo, applicationRepo, hostRepo, mysqlRepo, redisRepo, nginxRepo, tomcatRepo, elasticsearchRepo)
```

**INSERT BETWEEN:**
```go
	hostSyncService := sync.NewHostSyncService(n9eClient, hostRepo, log)

	instanceDiscoveryService := sync.NewInstanceDiscoveryService(
		vmClient,
		hostRepo,
		mysqlRepo,
		redisRepo,
		nginxRepo,
		tomcatRepo,
		elasticsearchRepo,
		log,
	)

	assetService := asset.NewAssetService(db, projectRepo, applicationRepo, hostRepo, mysqlRepo, redisRepo, nginxRepo, tomcatRepo, elasticsearchRepo)
```

### Change 3.2: Initialize MiddlewareHandler

**Line**: ~143-146 (after `roleHandler`)

**FIND:**
```go
	userHandler := handler.NewUserHandler(userService)
	roleHandler := handler.NewRoleHandler(roleService)

	// Start HTTP server
```

**INSERT BETWEEN:**
```go
	userHandler := handler.NewUserHandler(userService)
	roleHandler := handler.NewRoleHandler(roleService)
	middlewareHandler := handler.NewMiddlewareHandler(instanceDiscoveryService)

	// Start HTTP server
```

### Change 3.3: Add to router.Handlers

**Line**: ~152-160

**FIND:**
```go
	router := router.New(*serverConfig, log, router.Handlers{
		Auth:       authHandler,
		User:       userHandler,
		Role:       roleHandler,
		Asset:      assetHandler,
		Monitor:    monitorHandler,
		Alert:      alertHandler,
		Inspection: inspectionHandler,
	})
```

**REPLACE WITH:**
```go
	router := router.New(*serverConfig, log, router.Handlers{
		Auth:       authHandler,
		User:       userHandler,
		Role:       roleHandler,
		Asset:      assetHandler,
		Monitor:    monitorHandler,
		Alert:      alertHandler,
		Inspection: inspectionHandler,
		Middleware: middlewareHandler,  // ← NEW
	})
```

---

## Implementation Steps

### Step 1: Create the handler file

```bash
cd /home/kchou/Code/inspection-tool

# Create the new handler
cat > apps/cmdb-server/internal/api/handler/middleware.go << 'EOF'
[Copy the complete middleware.go code from File 1 above]
EOF
```

### Step 2: Modify router.go

```bash
# Edit manually or use text editor
vim apps/cmdb-server/internal/api/router/router.go

# Apply Change 2.1: Add Middleware to Handlers struct (line ~29)
# Apply Change 2.2: Add middleware route (line ~232-243)
```

### Step 3: Modify main.go

```bash
# Edit manually or use text editor
vim apps/cmdb-server/cmd/main.go

# Apply Change 3.1: Initialize service (line ~118)
# Apply Change 3.2: Initialize handler (line ~143)
# Apply Change 3.3: Add to router (line ~159)
```

### Step 4: Rebuild backend

```bash
# Option A: Build locally
cd /home/kchou/Code/inspection-tool
go build -o apps/cmdb-server/cmdb-server ./apps/cmdb-server/cmd/main.go

# Option B: Use make
make build

# Option C: Docker rebuild
docker compose -f docker-compose-dev.yml up -d --build backend
```

### Step 5: Restart server

```bash
# If using Docker
docker compose -f docker-compose-dev.yml restart backend

# If running locally
pkill cmdb-server
./apps/cmdb-server/cmdb-server -config cmdb-config.yaml
```

---

## Testing Plan

### Test 1: Login and get token

```bash
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.data.access_token')

echo "Token: $TOKEN"
```

**Expected**: Valid JWT token string

### Test 2: Trigger middleware discovery

```bash
curl -X POST "http://127.0.0.1:8080/api/v1/middleware/discover" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

**Expected Response:**
```json
{
  "code": 0,
  "message": "middleware discovery completed",
  "data": {
    "mysql": 15,
    "redis": 8,
    "nginx": 4,
    "tomcat": 2,
    "elasticsearch": 1,
    "duration": "1.234567s"
  }
}
```

### Test 3: Verify database counts

```bash
docker exec inspection-tool-postgres-1 psql -U postgres -d cmdb -c "
  SELECT 'mysql' as type, COUNT(*) as count FROM mysql_instances
  UNION ALL SELECT 'redis', COUNT(*) FROM redis_instances
  UNION ALL SELECT 'nginx', COUNT(*) FROM nginx_instances
  UNION ALL SELECT 'tomcat', COUNT(*) FROM tomcat_instances
  UNION ALL SELECT 'elasticsearch', COUNT(*) FROM elasticsearch_clusters;
"
```

**Expected Output:**
```
     type      | count
---------------+-------
 mysql         |    15
 redis         |     8
 nginx         |     4
 tomcat        |     2
 elasticsearch |     1
```

### Test 4: Verify API endpoints

```bash
# Test each middleware type API
curl -s "http://127.0.0.1:8080/api/v1/mysql?page=1&page_size=5" \
  -H "Authorization: Bearer $TOKEN" | jq '.data | {total, items: .items | length}'

curl -s "http://127.0.0.1:8080/api/v1/redis?page=1&page_size=5" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.total'

curl -s "http://127.0.0.1:8080/api/v1/nginx?page=1&page_size=5" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.total'

curl -s "http://127.0.0.1:8080/api/v1/tomcat?page=1&page_size=5" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.total'

curl -s "http://127.0.0.1:8080/api/v1/elasticsearch?page=1&page_size=5" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.total'
```

**Expected**: Non-zero totals for each type

### Test 5: Refresh dashboard

```bash
# Open browser
xdg-open http://127.0.0.1:8088/#/cmdb/dashboard

# Or navigate manually and do hard refresh (Ctrl+Shift+R)
```

**Expected**: All middleware cards show actual counts (not zeros)

---

## Expected Results

### Dashboard State After Implementation

| Asset | Before | After | Action |
|-------|--------|-------|--------|
| **Hosts** | 808 | 808 | ✅ Already synced |
| Projects | 0 | 0 | ⚠️ User creates manually |
| Applications | 0 | 0 | ⚠️ User creates manually |
| **MySQL** | **0** | **~15** | ✅ **Auto-discovered** |
| **Redis** | **0** | **~8** | ✅ **Auto-discovered** |
| **Nginx** | **0** | **~4** | ✅ **Auto-discovered** |
| **Tomcat** | **0** | **~2** | ✅ **Auto-discovered** |
| **Elasticsearch** | **0** | **~1** | ✅ **Auto-discovered** |

*Note: Actual counts depend on metrics in VictoriaMetrics*

---

## Troubleshooting

### Issue 1: Discovery returns all zeros

**Symptoms:**
```json
{
  "code": 0,
  "data": {
    "mysql": 0,
    "redis": 0,
    "nginx": 0,
    "tomcat": 0,
    "elasticsearch": 0
  }
}
```

**Possible Causes:**
1. VictoriaMetrics has no middleware metrics
2. Metric names don't match expected format
3. VictoriaMetrics connection issue

**Debug Steps:**

```bash
# Test VictoriaMetrics directly
curl "http://120.26.87.44:8428/api/v1/query?query=mysql_up" | jq .

# Check what metrics are available
curl "http://120.26.87.44:8428/api/v1/label/__name__/values" | jq . | grep -i mysql

# Check backend logs
docker logs inspection-tool-backend-1 --tail 100 | grep -i discovery
```

**Solutions:**
- Verify VictoriaMetrics has metrics (mysql_up, redis_up, etc.)
- Check VictoriaMetrics endpoint in config is correct
- Ensure Categraf is collecting middleware metrics

### Issue 2: 404 Page Not Found

**Symptoms:**
```
404 page not found
```

**Possible Causes:**
1. Binary not rebuilt
2. Container not restarted
3. Route not registered

**Solution:**

```bash
# Rebuild
cd /home/kchou/Code/inspection-tool
go build -o apps/cmdb-server/cmdb-server ./apps/cmdb-server/cmd/main.go

# Restart
docker compose -f docker-compose-dev.yml restart backend

# Check logs
docker logs inspection-tool-backend-1 --tail 50
```

### Issue 3: Handler is nil

**Symptoms:**
Backend logs show:
```
panic: runtime error: invalid memory address or nil pointer dereference
```

**Possible Causes:**
1. Handler not initialized in main.go
2. Handler not added to router.Handlers

**Solution:**
- Verify Change 3.2 (handler initialization)
- Verify Change 3.3 (handler added to router)
- Rebuild and restart

### Issue 4: Discovery takes too long / timeout

**Symptoms:**
- Request hangs
- Gateway timeout (504)

**Possible Causes:**
1. VictoriaMetrics slow to respond
2. Large number of metrics
3. Network latency

**Solution:**

```bash
# Test VM response time
time curl "http://120.26.87.44:8428/api/v1/query?query=up"

# Increase timeout in config (if needed)
# Or run discovery in background and poll status
```

### Issue 5: Duplicate instances created

**Symptoms:**
Running discovery multiple times creates duplicates

**Possible Causes:**
- Instance address matching logic issue
- Database unique constraint missing

**Solution:**

```bash
# Check for duplicates
docker exec inspection-tool-postgres-1 psql -U postgres -d cmdb -c "
  SELECT address, COUNT(*)
  FROM mysql_instances
  GROUP BY address
  HAVING COUNT(*) > 1;
"

# If duplicates exist, they'll be updated on next discovery
# The service uses FindByAddress to check for existing instances
```

---

## Next Steps

### Immediate (After Implementation)

1. ✅ Test endpoint with curl
2. ✅ Verify database has data
3. ✅ Refresh dashboard and confirm counts
4. ✅ Document in DEPLOYMENT.md

### Short-term (This Week)

1. **Add UI Button**: Add "Discover Middleware" button on dashboard
2. **Empty State UI**: Show helpful message when counts are zero
3. **Last Sync Time**: Display when discovery was last run
4. **Loading States**: Add spinners during discovery

### Medium-term (This Month)

1. **Scheduled Discovery**: Add cron job to run discovery periodically
2. **Discovery History**: Track discovery runs in database
3. **Selective Discovery**: Allow discovering specific middleware types
4. **Discovery Logs**: Better logging for debugging

### Long-term (Future)

1. **Incremental Discovery**: Only discover new/changed instances
2. **Discovery Dashboard**: Dedicated page for discovery management
3. **Manual Instance Add**: Allow users to manually add instances
4. **Instance Validation**: Verify instances are reachable

---

## Additional Notes

### Why This Approach?

1. **Minimal Changes**: Only 3 files modified, follows existing patterns
2. **No Breaking Changes**: Doesn't affect existing functionality
3. **Consistent**: Uses same patterns as host sync
4. **Testable**: Clear testing steps and expected results

### Service Architecture

```
API Request
    ↓
MiddlewareHandler.DiscoverInstances()
    ↓
InstanceDiscoveryService.DiscoverAll()
    ↓
├─ DiscoverMySQL() → VictoriaMetrics (mysql_up)
├─ DiscoverRedis() → VictoriaMetrics (redis_up)
├─ DiscoverNginx() → VictoriaMetrics (nginx_up)
├─ DiscoverTomcat() → VictoriaMetrics (tomcat_up)
└─ DiscoverElasticsearch() → VictoriaMetrics (elasticsearch_cluster_health_status)
    ↓
Database (UPSERT instances)
    ↓
API Response (counts + duration)
```

### Dependencies

- **VictoriaMetrics**: Must be accessible and have metrics
- **Database**: PostgreSQL must have tables created
- **Hosts**: Host records must exist for instance linking

---

## Checklist

Before implementation:
- [ ] Review all code changes
- [ ] Understand FIND/REPLACE sections
- [ ] Have editor ready (vim/vscode/etc)
- [ ] Docker environment running
- [ ] Admin credentials available

During implementation:
- [ ] Create middleware.go
- [ ] Modify router.go (2 changes)
- [ ] Modify main.go (3 changes)
- [ ] Rebuild binary
- [ ] Restart server
- [ ] Check logs for errors

After implementation:
- [ ] Run Test 1 (login)
- [ ] Run Test 2 (discovery)
- [ ] Run Test 3 (database)
- [ ] Run Test 4 (API endpoints)
- [ ] Run Test 5 (dashboard)
- [ ] Document results

---

## Support

If you encounter issues:

1. **Check Logs**:
   ```bash
   docker logs inspection-tool-backend-1 --tail 100 -f
   ```

2. **Check Database**:
   ```bash
   docker exec -it inspection-tool-postgres-1 psql -U postgres -d cmdb
   ```

3. **Test VictoriaMetrics**:
   ```bash
   curl "http://120.26.87.44:8428/api/v1/query?query=up"
   ```

4. **Reference Files**:
   - Progress log: `cmdb-memory/progress.md`
   - Config example: `apps/cmdb-server/configs/config.example.yaml`
   - Service code: `apps/cmdb-server/internal/service/sync/instance_discovery_service.go`

---

**End of Implementation Plan**

Generated: 2026-01-22
Author: Claude (All-Plan Collaboration Session)
Status: Ready for implementation
