# CMDB Server

Configuration Management Database server for inspection-tool.

## Configuration Setup

The CMDB Server requires a configuration file to run. Follow these steps to set it up:

1. **Copy the example configuration:**
   ```bash
   cp config.example.yaml ../../cmdb-config.yaml
   ```

2. **Edit the configuration:**
   Open `cmdb-config.yaml` and update it with your real credentials:
   - Update `flashduty.app_key` with your FlashDuty API key
   - Update database credentials if needed
   - Update external system URLs (N9E, VictoriaMetrics) if needed

3. **Important: Never commit `cmdb-config.yaml`**
   The file is already in `.gitignore` to prevent accidental commits.

## Running the Server

### Development
```bash
cd apps/cmdb-server
go run ./cmd/main.go
```

### Production
```bash
cd apps/cmdb-server
go build -o cmdb-server ./cmd/main.go
./cmdb-server -config ../../cmdb-config.yaml
```

### Command Line Options

- `-config <path>`: Path to configuration file (default: `../cmdb-config.yaml`)
- `-migrate`: Run database migrations only, then exit

## API Endpoints

### Health Check
- `GET /health` - Server health status

### Authentication
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/logout` - User logout
- `POST /api/v1/auth/refresh` - Refresh access token

### Assets
- `GET /api/v1/projects` - List projects
- `GET /api/v1/applications` - List applications
- `GET /api/v1/hosts` - List hosts
- `POST /api/v1/hosts/sync` - Sync hosts from N9E

### Middleware
- `GET /api/v1/mysql` - List MySQL instances
- `GET /api/v1/redis` - List Redis instances
- `GET /api/v1/nginx` - List Nginx instances
- `GET /api/v1/tomcat` - List Tomcat instances
- `GET /api/v1/elasticsearch` - List Elasticsearch clusters

### Monitoring
- `GET /api/v1/monitor/query` - Instant metric query
- `GET /api/v1/monitor/query_range` - Range metric query

### Alerts
- `GET /api/v1/alerts` - List alerts
- `GET /api/v1/incidents` - List incidents

### Inspection
- `GET /api/v1/inspection/jobs` - List inspection jobs
- `POST /api/v1/inspection/jobs` - Create inspection job

## Configuration Reference

| Section | Key | Description |
|---------|-----|-------------|
| `server` | `port` | HTTP server port |
| `server` | `mode` | Server mode (debug/release/test) |
| `server` | `static_path` | Path to frontend static files |
| `database` | - | PostgreSQL connection settings |
| `redis` | - | Redis connection settings |
| `jwt` | - | JWT token settings |
| `n9e` | - | N9E API configuration |
| `victoriametrics` | - | VictoriaMetrics configuration |
| `flashduty` | - | FlashDuty API configuration |
| `inspection` | - | Inspection CLI configuration |
