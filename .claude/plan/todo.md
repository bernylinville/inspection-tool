# Fix CMDB Dashboard Alert API Errors

## Steps
- [ ] S1: Add ErrAlertServiceUnavailable error type in alert_proxy.go
- [ ] S2: Map network errors (EOF, timeout, connection refused) to new error type
- [ ] S3: Update alert.go handler to return HTTP 200 with service_unavailable flag
- [ ] S4: Preserve HTTP 500 for genuine internal errors
- [ ] S5: Rebuild backend and verify dashboard loads without errors

## Acceptance
- /api/v1/alerts returns HTTP 200 with service_unavailable:true when FlashDuty is unreachable
- Auth/config errors still return appropriate error codes
- Dashboard displays existing service unavailable message
