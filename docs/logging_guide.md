# Logging Guide

This document outlines the logging best practices and architecture for the One-Click AKS Server application.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Best Practices by Layer](#best-practices-by-layer)
- [Context-Based Logging](#context-based-logging)
- [Log Levels and Filtering](#log-levels-and-filtering)
- [Implementation Examples](#implementation-examples)
- [Configuration](#configuration)
- [Troubleshooting](#troubleshooting)

## Overview

Our logging system is built around these core principles:

1. **Context-Aware**: Every log entry includes trace ID and user context for correlation
2. **Structured**: JSON format with consistent field naming
3. **Layered Responsibility**: Each architectural layer has specific logging responsibilities
4. **Production-Ready**: Path-based filtering to reduce noise from health checks
5. **Traceable**: Request flows can be tracked end-to-end using trace IDs

## Architecture

### Logging Components

```text
internal/
├── logging/
│   └── logger.go          # Core logging functionality
├── middleware/
│   ├── context.go         # Trace ID and user context injection
│   └── logging.go         # HTTP request logging with filtering
```

### Key Features

- **Custom Handler**: Project-relative source paths in logs
- **String-Based Levels**: DEBUG, INFO, WARN, ERROR (user-friendly)
- **Trace Correlation**: UUID v4 trace IDs for request tracking
- **User Isolation**: User ID in context for multi-tenant logging
- **Path Filtering**: Configurable log levels per endpoint pattern

## Best Practices by Layer

### Handler Layer - Minimal Logging

**Responsibility**: HTTP-specific concerns only

```go
func (h *UserHandler) CreateUser(c *gin.Context) {
    // ✅ Log entry points
    logging.LogInfo(c, "Creating user", map[string]interface{}{
        "endpoint": "POST /users",
    })
    
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        // ✅ Log validation errors
        logging.LogError(c, "Invalid request payload", err, map[string]interface{}{
            "validation_error": err.Error(),
        })
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }
    
    // Call service - let it handle detailed logging
    user, err := h.userService.CreateUser(c.Request.Context(), req)
    if err != nil {
        // ❌ Don't log here - service already logged the details
        c.JSON(500, gin.H{"error": "Failed to create user"})
        return
    }
    
    c.JSON(201, user)
}
```

**What to Log in Handlers:**

- ✅ Entry points (endpoint accessed)
- ✅ Validation errors (malformed requests)
- ✅ Authentication/authorization failures
- ❌ Business logic errors (let service handle)
- ❌ Database errors (let repository handle)

### Service Layer - Primary Logging

**Responsibility**: Business logic outcomes and orchestration

```go
func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
    // ✅ Log business operations start
    logging.LogInfo(ctx, "Starting user creation", map[string]interface{}{
        "email": req.Email,
        "operation": "create_user",
    })
    
    // ✅ Log validation failures
    if err := s.validateUser(req); err != nil {
        logging.LogError(ctx, "User validation failed", err, map[string]interface{}{
            "email": req.Email,
            "validation_rule": "email_format",
        })
        return nil, fmt.Errorf("invalid user data: %w", err)
    }
    
    // Call repository
    user, err := s.userRepo.Create(ctx, req)
    if err != nil {
        // ✅ Log business context of infrastructure failures
        logging.LogError(ctx, "Failed to create user in database", err, map[string]interface{}{
            "email": req.Email,
            "operation": "db_create",
        })
        return nil, fmt.Errorf("database error: %w", err)
    }
    
    // ✅ Log successful outcomes
    logging.LogInfo(ctx, "User created successfully", map[string]interface{}{
        "user_id": user.ID,
        "email": user.Email,
    })
    
    return user, nil
}
```

**What to Log in Services:**

- ✅ Business operation start/completion
- ✅ Business rule validation failures
- ✅ External service interactions
- ✅ Critical business decisions
- ✅ Performance metrics (when needed)

### Repository Layer - Infrastructure Only

**Responsibility**: Database and infrastructure errors only

```go
func (r *UserRepository) Create(ctx context.Context, req CreateUserRequest) (*User, error) {
    query := `INSERT INTO users (email, name) VALUES ($1, $2) RETURNING id, created_at`
    
    var user User
    err := r.db.QueryRowContext(ctx, query, req.Email, req.Name).Scan(&user.ID, &user.CreatedAt)
    if err != nil {
        // ✅ Only log infrastructure/database errors
        logging.LogError(ctx, "Database query failed", err, map[string]interface{}{
            "query": "insert_user",
            "table": "users",
            "error_type": "database",
        })
        return nil, err
    }
    
    // ❌ No success logging - let service layer handle business outcomes
    return &user, nil
}
```

**What to Log in Repositories:**

- ✅ Database connection failures
- ✅ Query execution errors
- ✅ Transaction rollbacks
- ✅ Infrastructure timeouts
- ❌ Successful operations (service handles this)
- ❌ Business logic (not repository's concern)

## Context-Based Logging

### Trace ID Generation

Every request gets a unique trace ID (UUID v4 format) for end-to-end tracking:

```go
// Automatic trace ID injection via middleware
func ContextMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        traceID := generateTraceID() // UUID v4: e.g., "550e8400-e29b-41d4-a716-446655440000"
        
        ctx := context.WithValue(c.Request.Context(), traceIDKey, traceID)
        c.Request = c.Request.WithContext(ctx)
        c.Header("X-Trace-ID", traceID)
        
        c.Next()
    }
}
```

### User Context

User information is automatically injected for multi-tenant logging:

```go
// In authentication middleware
func SetUserIDInGin(c *gin.Context, userID string) {
    ctx := context.WithValue(c.Request.Context(), userIDKey, userID)
    c.Request = c.Request.WithContext(ctx)
}
```

### Using Context in Logs

Always pass the context to logging functions:

```go
// ✅ Correct - context provides trace ID and user ID
logging.LogInfo(ctx, "Processing request", map[string]interface{}{
    "operation": "create_lab",
    "lab_type": "kubernetes",
})

// ❌ Wrong - missing trace correlation
log.Info("Processing request")
```

## Log Levels and Filtering

### Standard Log Levels

- **DEBUG**: Detailed diagnostic information
- **INFO**: General information about application flow
- **WARN**: Warning conditions that don't stop execution
- **ERROR**: Error conditions that require attention

### Path-Based Filtering

Configure different log levels for different endpoints:

```go
config := middleware.LoggingConfig{
    // Health checks at DEBUG level to reduce noise
    QuietPaths: []string{"/status", "/health", "/healthz", "/ping", "/metrics"},
    
    // Skip logging entirely for these paths
    SkipPaths: []string{"/favicon.ico"},
    
    // Only log errors for high-frequency endpoints
    ErrorOnlyPaths: []string{"/api/internal/*"},
}
```

### Configuration Options

```go
type LoggingConfig struct {
    QuietPaths     []string  // Log at DEBUG level only
    SkipPaths      []string  // No logging at all
    ErrorOnlyPaths []string  // Only log errors
}
```

## Implementation Examples

### Complete Handler Example

```go
func (h *LabHandler) CreateLab(c *gin.Context) {
    // Entry point logging
    logging.LogInfo(c, "Lab creation request received", map[string]interface{}{
        "endpoint": "POST /labs",
    })
    
    var req CreateLabRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        logging.LogError(c, "Invalid request payload", err, map[string]interface{}{
            "validation_error": err.Error(),
            "endpoint": "POST /labs",
        })
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }
    
    // Delegate to service
    result, err := h.labService.CreateLab(c, req)
    if err != nil {
        // Service already logged error details
        c.JSON(500, gin.H{"error": "Failed to create lab"})
        return
    }
    
    logging.LogInfo(c, "Lab creation completed", map[string]interface{}{
        "lab_id": result.ID,
        "response_status": 201,
    })
    
    c.JSON(201, result)
}
```

### Complete Service Example

```go
func (s *LabService) CreateLab(ctx context.Context, req CreateLabRequest) (*Lab, error) {
    logging.LogInfo(ctx, "Starting lab creation", map[string]interface{}{
        "lab_type": req.Type,
        "user_id": getUserID(ctx),
        "operation": "create_lab",
    })
    
    // Validate business rules
    if err := s.validateLabRequest(req); err != nil {
        logging.LogError(ctx, "Lab validation failed", err, map[string]interface{}{
            "lab_type": req.Type,
            "validation_rule": err.Error(),
        })
        return nil, fmt.Errorf("invalid lab configuration: %w", err)
    }
    
    // Check resource availability
    if !s.checkResourceAvailability(ctx, req) {
        logging.LogWarn(ctx, "Insufficient resources for lab creation", map[string]interface{}{
            "lab_type": req.Type,
            "requested_resources": req.Resources,
        })
        return nil, errors.New("insufficient resources")
    }
    
    // Create lab
    lab, err := s.labRepo.Create(ctx, req)
    if err != nil {
        logging.LogError(ctx, "Failed to create lab in database", err, map[string]interface{}{
            "lab_type": req.Type,
            "operation": "db_create",
        })
        return nil, fmt.Errorf("database error: %w", err)
    }
    
    // Initialize resources
    if err := s.initializeLabResources(ctx, lab); err != nil {
        logging.LogError(ctx, "Failed to initialize lab resources", err, map[string]interface{}{
            "lab_id": lab.ID,
            "operation": "resource_init",
        })
        // Cleanup lab record
        s.labRepo.Delete(ctx, lab.ID)
        return nil, fmt.Errorf("resource initialization failed: %w", err)
    }
    
    logging.LogInfo(ctx, "Lab created successfully", map[string]interface{}{
        "lab_id": lab.ID,
        "lab_type": lab.Type,
        "duration_ms": time.Since(startTime).Milliseconds(),
    })
    
    return lab, nil
}
```

### Complete Repository Example

```go
func (r *LabRepository) Create(ctx context.Context, req CreateLabRequest) (*Lab, error) {
    query := `
        INSERT INTO labs (name, type, config, user_id, created_at) 
        VALUES ($1, $2, $3, $4, NOW()) 
        RETURNING id, created_at
    `
    
    var lab Lab
    lab.Name = req.Name
    lab.Type = req.Type
    lab.Config = req.Config
    lab.UserID = getUserID(ctx)
    
    err := r.db.QueryRowContext(ctx, query, 
        lab.Name, lab.Type, lab.Config, lab.UserID,
    ).Scan(&lab.ID, &lab.CreatedAt)
    
    if err != nil {
        logging.LogError(ctx, "Database query failed", err, map[string]interface{}{
            "query": "insert_lab",
            "table": "labs",
            "error_type": "database",
        })
        return nil, err
    }
    
    return &lab, nil
}
```

## Configuration

### Setting Up Logging

```go
// In main.go
func main() {
    // Setup logging
    logging.SetupLogger("INFO", true) // level, enableJSON
    
    // Setup Gin with logging middleware
    r := gin.New()
    
    // Add context middleware (adds trace ID)
    r.Use(middleware.ContextMiddleware())
    
    // Add logging middleware with custom config
    config := middleware.DefaultLoggingConfig()
    config.QuietPaths = append(config.QuietPaths, "/api/status")
    r.Use(middleware.GinLoggerWithConfig(config))
    
    // Your routes...
}
```

### Environment Variables

```bash
# Log level
LOG_LEVEL=INFO

# Enable JSON logging in production
LOG_JSON=true

# Custom quiet paths (comma-separated)
LOG_QUIET_PATHS="/status,/health,/metrics"
```

## Troubleshooting

### Common Issues

#### 1. Missing Trace IDs

**Problem**: Logs don't show trace IDs
**Solution**: Ensure `ContextMiddleware()` is registered before other middleware

#### 2. Duplicate Logging

**Problem**: Same error logged multiple times
**Solution**: Follow the layer responsibility pattern - only service layer logs business errors

#### 3. Too Much Noise

**Problem**: Health check endpoints filling logs
**Solution**: Use `QuietPaths` configuration to set DEBUG level for health checks

#### 4. Missing Context

**Problem**: Context is nil or missing user/trace info
**Solution**: Always pass the original request context down through all layers

### Debugging Tips

1. **Search by Trace ID**: Use trace ID to find all related log entries
2. **Filter by User ID**: Isolate logs for specific users
3. **Use Structured Fields**: Search by operation, table, error_type, etc.
4. **Check Log Levels**: Verify appropriate level is set for your environment

### Log Analysis Queries

```bash
# Find all logs for a trace ID
grep "550e8400-e29b-41d4-a716-446655440000" app.log

# Find all errors for a user
jq 'select(.user_id == "user123" and .level == "ERROR")' app.log

# Find database errors
jq 'select(.error_type == "database")' app.log
```

## Anti-Patterns to Avoid

### ❌ Don't Do This

```go
// Double logging
func (s *Service) CreateUser(ctx context.Context) error {
    err := s.repo.Create(ctx, user)
    if err != nil {
        log.Error("Service failed", err) // ❌ Don't log here
        return err
    }
}

func (r *Repository) Create(ctx context.Context, user User) error {
    if err := r.db.Create(user); err != nil {
        log.Error("DB failed", err) // ❌ Already logged above
        return err
    }
}

// Logging without context
log.Info("User created") // ❌ No trace ID, no user context

// Inconsistent field names
log.Info("msg", "userId", "123") // ❌ Should be "user_id"
log.Info("msg", "user_id", "456") // ✅ Consistent naming
```

### ✅ Do This Instead

```go
// Single point of responsibility
func (s *Service) CreateUser(ctx context.Context) error {
    err := s.repo.Create(ctx, user)
    if err != nil {
        logging.LogError(ctx, "Failed to create user", err, map[string]interface{}{
            "operation": "create_user",
        })
        return fmt.Errorf("service error: %w", err)
    }
}

func (r *Repository) Create(ctx context.Context, user User) error {
    // Just return error, let service decide what to log
    return r.db.Create(user)
}
```

---

## Summary

This logging architecture provides:

- **Traceability**: Every request can be tracked end-to-end
- **Maintainability**: Clear responsibilities per layer
- **Production-Ready**: Noise reduction for health checks
- **Debuggability**: Rich context and structured data
- **Performance**: Minimal overhead with smart filtering

Follow these patterns to maintain consistent, useful logging across the entire application.
