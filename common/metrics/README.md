# Metrics Package

Comprehensive OpenTelemetry-based metrics package for GRUD services.

## Overview

This package provides pre-configured metric collectors for:
- **Runtime Metrics**: Go process health (goroutines, memory, GC)
- **Database Metrics**: Connection pool stats and query performance
- **Messaging Metrics**: NATS publish/consume performance
- **Health Metrics**: Dependency availability and service info

**Note**: Business metrics are service-specific and should be implemented in each service's own metrics package.

All metrics are exported via OpenTelemetry to the OTel Collector, then to Prometheus.

## Usage

### 1. Initialize Metrics

```go
import (
    "grud/common/telemetry"
    "grud/common/metrics"
)

// In your app initialization
ctx := context.Background()
telem, err := telemetry.Init(ctx, "student-service", "1.0.0", "production", logger)
if err != nil {
    log.Fatal(err)
}
defer telemetry.Shutdown(ctx, telem.MeterProvider, logger)

// Access metrics collectors
m := telem.Metrics
```

### 2. Runtime Metrics (Automatic)

Runtime metrics are collected automatically every 10 seconds:
- `runtime.go.goroutines` - Number of goroutines
- `runtime.go.mem.heap_alloc` - Heap memory allocated
- `runtime.go.mem.heap_inuse` - Heap memory in use
- `runtime.go.mem.heap_objects` - Heap objects count
- `runtime.go.mem.stack_inuse` - Stack memory
- `runtime.go.gc.count` - GC cycles count
- `service.uptime` - Service uptime in seconds

No code needed - these are collected automatically!

### 3. Database Metrics

#### Register Database Connection Pool

```go
import "grud/common/metrics"

// After creating database connection
db := db.New(cfg.Database)

// Register for automatic connection pool monitoring
meter := otel.Meter("student-service")
if err := m.Database.RegisterDB(db.DB(), meter); err != nil {
    logger.Warn("failed to register database metrics", "error", err)
}
```

#### Record Query Performance

```go
// Example: Track query execution
func (r *Repository) GetStudentByID(ctx context.Context, id int64) (*Student, error) {
    start := time.Now()

    var student Student
    err := r.db.GetContext(ctx, &student, "SELECT * FROM students WHERE id = $1", id)

    // Record query metrics
    r.metrics.Database.RecordQuery(ctx, "SELECT", "students", time.Since(start), err)

    return &student, err
}
```

Metrics exposed:
- `db.connections.open` - Open connections
- `db.connections.idle` - Idle connections
- `db.connections.in_use` - Active connections
- `db.connections.max_open` - Max connections limit
- `db.query.duration{operation, table}` - Query latency
- `db.query.errors{operation, table}` - Query errors

### 4. Messaging Metrics (NATS)

#### Record Message Publish

```go
func (p *Producer) Publish(ctx context.Context, subject string, data []byte) error {
    start := time.Now()

    err := p.nc.Publish(subject, data)

    // Record publish metrics
    p.metrics.Messaging.RecordPublish(ctx, subject, time.Since(start), err)

    return err
}
```

#### Record Message Consumption

```go
func (c *Consumer) handleMessage(msg *nats.Msg) {
    start := time.Now()
    ctx := context.Background()

    err := c.processMessage(msg)

    // Record consume metrics
    c.metrics.Messaging.RecordConsume(ctx, msg.Subject, time.Since(start), err)
}
```

#### Track Connections

```go
// When connecting
nc, err := nats.Connect(url)
if err == nil {
    metrics.Messaging.RecordConnectionChange(ctx, 1) // +1 connection
}

// When disconnecting
defer func() {
    nc.Close()
    metrics.Messaging.RecordConnectionChange(ctx, -1) // -1 connection
}()
```

Metrics exposed:
- `messaging.messages.published{subject}` - Messages published
- `messaging.messages.consumed{subject}` - Messages consumed
- `messaging.message.processing_duration{subject}` - Processing time
- `messaging.message.publish_duration{subject}` - Publish time
- `messaging.message.errors{subject}` - Message errors
- `messaging.connections.active` - Active connections

### 5. Health & Dependency Metrics

#### Register Dependencies

```go
// In app initialization
meter := otel.Meter("student-service")
dependencies := []string{"postgres", "nats", "project-service"}
if err := m.Health.RegisterDependencies(ctx, meter, dependencies); err != nil {
    logger.Warn("failed to register dependencies", "error", err)
}
```

#### Health Checks

```go
func (h *HealthHandler) checkDatabase(ctx context.Context) error {
    start := time.Now()

    err := h.db.PingContext(ctx)

    // Record dependency check
    h.metrics.Health.RecordDependencyCheck(ctx, "postgres", time.Since(start), err)

    return err
}

func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    ctx := r.Context()

    err := h.checkDatabase(ctx)

    // Record overall health check
    h.metrics.Health.RecordHealthCheck(ctx, "readiness", time.Since(start), err)

    if err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        return
    }
    w.WriteHeader(http.StatusOK)
}
```

Metrics exposed:
- `dependency.up{dependency}` - 1=up, 0=down
- `dependency.response_time{dependency}` - Check latency
- `health.check.duration{check_type}` - Health check duration
- `health.check.errors{check_type}` - Health check errors
- `service.info{service_name, version, environment}` - Service metadata

## Integration Example

See `examples/metrics_integration.go` for complete service integration example.

## Prometheus Queries

### Golden Signals (RED Method)

#### Request Rate
```promql
# HTTP request rate
rate(http_server_duration_seconds_count{service_name="student-service"}[5m])

# Database query rate
rate(grud_db_query_duration_count[5m])

# Message processing rate
rate(grud_messaging_message_processing_duration_count[5m])
```

#### Error Rate
```promql
# Database query errors
rate(grud_db_query_errors_total[5m])

# Messaging errors
rate(grud_messaging_message_errors_total[5m])

# HTTP errors (5xx)
rate(http_server_duration_seconds_count{http_status_code=~"5.."}[5m])
```

#### Duration (Latency) - p50, p95, p99, p99.9

**Database Query Latency:**
```promql
# p50 (median) - 50% of queries are faster than this
histogram_quantile(0.50,
  rate(grud_db_query_duration_bucket{operation="SELECT"}[5m])
)

# p95 - 95% of queries are faster than this (SLA target)
histogram_quantile(0.95,
  rate(grud_db_query_duration_bucket{operation="SELECT"}[5m])
)

# p99 - 99% of queries are faster than this
histogram_quantile(0.99,
  rate(grud_db_query_duration_bucket{operation="SELECT"}[5m])
)

# p99.9 - 99.9% of queries are faster than this (worst case)
histogram_quantile(0.999,
  rate(grud_db_query_duration_bucket{operation="SELECT"}[5m])
)
```

**HTTP Request Latency:**
```promql
# p95 HTTP latency per endpoint
histogram_quantile(0.95,
  sum by (le, http_route) (
    rate(http_server_duration_seconds_bucket{service_name="student-service"}[5m])
  )
)

# p99 HTTP latency
histogram_quantile(0.99,
  rate(http_server_duration_seconds_bucket{service_name="student-service"}[5m])
)
```

**Messaging Latency:**
```promql
# p95 message processing time
histogram_quantile(0.95,
  rate(grud_messaging_message_processing_duration_bucket[5m])
)

# p99 message processing time
histogram_quantile(0.99,
  rate(grud_messaging_message_processing_duration_bucket[5m])
)
```

**gRPC Request Latency:**
```promql
# p95 gRPC latency
histogram_quantile(0.95,
  rate(rpc_server_duration_seconds_bucket{service_name="project-service"}[5m])
)

# p99 gRPC latency
histogram_quantile(0.99,
  rate(rpc_server_duration_seconds_bucket{service_name="project-service"}[5m])
)
```

### USE Method (Utilization, Saturation, Errors)

#### Utilization
```promql
# Database connection pool utilization (%)
(grud_db_connections_in_use / grud_db_connections_max_open) * 100

# Memory utilization
grud_runtime_go_mem_heap_inuse

# Goroutine count
grud_runtime_go_goroutines
```

#### Saturation
```promql
# Connection pool saturation - p95 wait time
histogram_quantile(0.95,
  rate(grud_db_connections_wait_duration_bucket[5m])
)

# Idle connections (should be > 0, if 0 = saturated)
grud_db_connections_idle
```

#### Errors
```promql
# Error rate by type
rate(grud_db_query_errors_total[5m])
rate(grud_messaging_message_errors_total[5m])
rate(grud_health_check_errors_total[5m])
```

### SLI/SLO Monitoring

#### Availability SLI (99.9% uptime target)
```promql
# Service uptime percentage over 30 days
(
  count_over_time(grud_service_info[30d])
  /
  (30 * 24 * 60 * 60 / 10)  # expected samples at 10s interval
) * 100
```

#### Latency SLI (95% of requests < 100ms)
```promql
# Percentage of requests faster than 100ms
(
  sum(rate(grud_db_query_duration_bucket{le="0.1"}[5m]))
  /
  sum(rate(grud_db_query_duration_count[5m]))
) * 100
```

#### Error Budget
```promql
# Error budget remaining (target: < 0.1% errors)
(1 - (
  sum(rate(grud_db_query_errors_total[30d]))
  /
  sum(rate(grud_db_query_duration_count[30d]))
)) * 100
```

### Advanced Queries

#### Apdex Score (Application Performance Index)
```promql
# Apdex: satisfied (< 100ms) + tolerated (< 500ms) / 2
(
  sum(rate(grud_db_query_duration_bucket{le="0.1"}[5m]))
  + sum(rate(grud_db_query_duration_bucket{le="0.5"}[5m])) / 2
) / sum(rate(grud_db_query_duration_count[5m]))
```

#### Heatmap of Query Duration Distribution
```promql
# For Grafana heatmap panel
sum by (le) (
  increase(grud_db_query_duration_bucket[1m])
)
```

#### Memory Leak Detection
```promql
# Heap growth over 1 hour
deriv(grud_runtime_go_mem_heap_inuse[1h])
```

## Best Practices

1. **Always use context**: Pass context to all metric recording functions
2. **Use labels sparingly**: High cardinality labels (like user IDs) cause memory issues
3. **Record errors**: Always record errors in metrics for observability
4. **Use timers**: Always measure duration for operations
5. **Service-specific metrics**: Implement business metrics in each service's own metrics package
