package logging

import (
	"time"
)

// Common field keys used throughout the system
const (
	// Request/Response fields
	FieldRequestID  = "request_id"
	FieldUserID     = "user_id"
	FieldSessionID  = "session_id"
	FieldMethod     = "method"
	FieldPath       = "path"
	FieldStatus     = "status"
	FieldDuration   = "duration"
	FieldRemoteAddr = "remote_addr"
	FieldUserAgent  = "user_agent"
	FieldReferer    = "referer"
	FieldBytes      = "bytes"
	FieldProtocol   = "protocol"

	// Guild-specific fields
	FieldCommissionID = "commission_id"
	FieldAgentID      = "agent_id"
	FieldAgentName    = "agent_name"
	FieldTaskID       = "task_id"
	FieldTaskType     = "task_type"
	FieldGuildID      = "guild_id"
	FieldWorkspace    = "workspace"
	FieldArtifact     = "artifact"

	// System fields
	FieldService     = "service"
	FieldVersion     = "version"
	FieldEnvironment = "environment"
	FieldHostname    = "hostname"
	FieldPID         = "pid"
	FieldComponent   = "component"
	FieldPackage     = "package"

	// Error fields
	FieldError      = "error"
	FieldErrorCode  = "error_code"
	FieldErrorType  = "error_type"
	FieldStackTrace = "stack_trace"
	FieldRetryCount = "retry_count"
	FieldRetryable  = "retryable"

	// Performance fields
	FieldLatency     = "latency"
	FieldCPUUsage    = "cpu_usage"
	FieldMemoryUsage = "memory_usage"
	FieldGoroutines  = "goroutines"
	FieldQueueSize   = "queue_size"
	FieldCacheHit    = "cache_hit"
	FieldCacheMiss   = "cache_miss"

	// Database fields
	FieldQuery         = "query"
	FieldQueryDuration = "query_duration"
	FieldRowsAffected  = "rows_affected"
	FieldDatabase      = "database"
	FieldTable         = "table"

	// Message/Event fields
	FieldEventType     = "event_type"
	FieldEventID       = "event_id"
	FieldMessageID     = "message_id"
	FieldCorrelationID = "correlation_id"
	FieldPayloadSize   = "payload_size"

	// Provider fields
	FieldProvider   = "provider"
	FieldModel      = "model"
	FieldTokensUsed = "tokens_used"
	FieldCost       = "cost"
	FieldRateLimit  = "rate_limit"
)

// Common field constructors for consistency

// Request fields
func RequestIDField(id string) Field      { return String(FieldRequestID, id) }
func UserIDField(id string) Field         { return String(FieldUserID, id) }
func SessionIDField(id string) Field      { return String(FieldSessionID, id) }
func MethodField(method string) Field     { return String(FieldMethod, method) }
func PathField(path string) Field         { return String(FieldPath, path) }
func StatusField(status int) Field        { return Int(FieldStatus, status) }
func DurationField(d time.Duration) Field { return Duration(FieldDuration, d) }
func RemoteAddrField(addr string) Field   { return String(FieldRemoteAddr, addr) }
func BytesField(bytes int) Field          { return Int(FieldBytes, bytes) }

// Guild-specific fields
func CommissionIDField(id string) Field     { return String(FieldCommissionID, id) }
func AgentIDField(id string) Field          { return String(FieldAgentID, id) }
func AgentNameField(name string) Field      { return String(FieldAgentName, name) }
func TaskIDField(id string) Field           { return String(FieldTaskID, id) }
func TaskTypeField(taskType string) Field   { return String(FieldTaskType, taskType) }
func GuildIDField(id string) Field          { return String(FieldGuildID, id) }
func WorkspaceField(workspace string) Field { return String(FieldWorkspace, workspace) }

// System fields
func ServiceField(service string) Field     { return String(FieldService, service) }
func VersionField(version string) Field     { return String(FieldVersion, version) }
func EnvironmentField(env string) Field     { return String(FieldEnvironment, env) }
func ComponentField(component string) Field { return String(FieldComponent, component) }
func PackageField(pkg string) Field         { return String(FieldPackage, pkg) }

// Error fields
func ErrorCodeField(code string) Field    { return String(FieldErrorCode, code) }
func ErrorTypeField(errType string) Field { return String(FieldErrorType, errType) }
func RetryCountField(count int) Field     { return Int(FieldRetryCount, count) }
func RetryableField(retryable bool) Field { return Bool(FieldRetryable, retryable) }

// Performance fields
func LatencyField(latency time.Duration) Field { return Duration(FieldLatency, latency) }
func CPUUsageField(usage float64) Field        { return Any(FieldCPUUsage, usage) }
func MemoryUsageField(usage int64) Field       { return Int64(FieldMemoryUsage, usage) }
func GoroutinesField(count int) Field          { return Int(FieldGoroutines, count) }
func QueueSizeField(size int) Field            { return Int(FieldQueueSize, size) }
func CacheHitField(hit bool) Field             { return Bool(FieldCacheHit, hit) }

// Database fields
func QueryField(query string) Field            { return String(FieldQuery, query) }
func QueryDurationField(d time.Duration) Field { return Duration(FieldQueryDuration, d) }
func RowsAffectedField(rows int64) Field       { return Int64(FieldRowsAffected, rows) }
func DatabaseField(db string) Field            { return String(FieldDatabase, db) }
func TableField(table string) Field            { return String(FieldTable, table) }

// Event fields
func EventTypeField(eventType string) Field { return String(FieldEventType, eventType) }
func EventIDField(id string) Field          { return String(FieldEventID, id) }
func MessageIDField(id string) Field        { return String(FieldMessageID, id) }
func CorrelationIDField(id string) Field    { return String(FieldCorrelationID, id) }
func PayloadSizeField(size int) Field       { return Int(FieldPayloadSize, size) }

// Provider fields
func ProviderField(provider string) Field { return String(FieldProvider, provider) }
func ModelField(model string) Field       { return String(FieldModel, model) }
func TokensUsedField(tokens int) Field    { return Int(FieldTokensUsed, tokens) }
func CostField(cost float64) Field        { return Any(FieldCost, cost) }
func RateLimitField(limit int) Field      { return Int(FieldRateLimit, limit) }

// FieldSet provides a builder pattern for accumulating fields
type FieldSet struct {
	fields []Field
}

// NewFieldSet creates a new field set
func NewFieldSet() *FieldSet {
	return &FieldSet{
		fields: make([]Field, 0, 8),
	}
}

// Add adds a field to the set
func (fs *FieldSet) Add(field Field) *FieldSet {
	fs.fields = append(fs.fields, field)
	return fs
}

// AddIf conditionally adds a field
func (fs *FieldSet) AddIf(condition bool, field Field) *FieldSet {
	if condition {
		fs.fields = append(fs.fields, field)
	}
	return fs
}

// AddString adds a string field if non-empty
func (fs *FieldSet) AddString(key, value string) *FieldSet {
	if value != "" {
		fs.fields = append(fs.fields, String(key, value))
	}
	return fs
}

// AddInt adds an int field if non-zero
func (fs *FieldSet) AddInt(key string, value int) *FieldSet {
	if value != 0 {
		fs.fields = append(fs.fields, Int(key, value))
	}
	return fs
}

// AddDuration adds a duration field if non-zero
func (fs *FieldSet) AddDuration(key string, value time.Duration) *FieldSet {
	if value != 0 {
		fs.fields = append(fs.fields, Duration(key, value))
	}
	return fs
}

// AddError adds an error field if non-nil
func (fs *FieldSet) AddError(err error) *FieldSet {
	if err != nil {
		fs.fields = append(fs.fields, ErrorField(err))
	}
	return fs
}

// Fields returns the accumulated fields
func (fs *FieldSet) Fields() []Field {
	return fs.fields
}
