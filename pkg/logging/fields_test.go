package logging

import (
	"errors"
	"testing"
	"time"
)

func TestFieldFunctions(t *testing.T) {
	// Test request fields
	t.Run("RequestFields", func(t *testing.T) {
		field := RequestIDField("req-123")
		if field.Key != "request_id" || field.Value.String() != "req-123" {
			t.Errorf("RequestIDField failed: %+v", field)
		}

		field = UserIDField("user-456")
		if field.Key != "user_id" || field.Value.String() != "user-456" {
			t.Errorf("UserIDField failed: %+v", field)
		}

		field = SessionIDField("sess-789")
		if field.Key != "session_id" || field.Value.String() != "sess-789" {
			t.Errorf("SessionIDField failed: %+v", field)
		}

		field = MethodField("POST")
		if field.Key != "method" || field.Value.String() != "POST" {
			t.Errorf("MethodField failed: %+v", field)
		}

		field = PathField("/api/v1/users")
		if field.Key != "path" || field.Value.String() != "/api/v1/users" {
			t.Errorf("PathField failed: %+v", field)
		}

		field = StatusField(200)
		if field.Key != "status" || field.Value.Int64() != 200 {
			t.Errorf("StatusField failed: %+v", field)
		}

		duration := 100 * time.Millisecond
		field = DurationField(duration)
		if field.Key != "duration" || field.Value.Duration() != duration {
			t.Errorf("DurationField failed: %+v", field)
		}

		field = RemoteAddrField("192.168.1.1:8080")
		if field.Key != "remote_addr" || field.Value.String() != "192.168.1.1:8080" {
			t.Errorf("RemoteAddrField failed: %+v", field)
		}

		field = BytesField(1024)
		if field.Key != "bytes" || field.Value.Int64() != int64(1024) {
			t.Errorf("BytesField failed: %+v", field)
		}
	})

	// Test commission fields
	t.Run("CommissionFields", func(t *testing.T) {
		field := CommissionIDField("comm-123")
		if field.Key != "commission_id" || field.Value.String() != "comm-123" {
			t.Errorf("CommissionIDField failed: %+v", field)
		}

		field = AgentIDField("agent-456")
		if field.Key != "agent_id" || field.Value.String() != "agent-456" {
			t.Errorf("AgentIDField failed: %+v", field)
		}

		field = AgentNameField("analyzer")
		if field.Key != "agent_name" || field.Value.String() != "analyzer" {
			t.Errorf("AgentNameField failed: %+v", field)
		}

		field = TaskIDField("task-789")
		if field.Key != "task_id" || field.Value.String() != "task-789" {
			t.Errorf("TaskIDField failed: %+v", field)
		}

		field = TaskTypeField("code-review")
		if field.Key != "task_type" || field.Value.String() != "code-review" {
			t.Errorf("TaskTypeField failed: %+v", field)
		}
	})

	// Test context fields
	t.Run("ContextFields", func(t *testing.T) {
		field := GuildIDField("guild-123")
		if field.Key != "guild_id" || field.Value.String() != "guild-123" {
			t.Errorf("GuildIDField failed: %+v", field)
		}

		field = WorkspaceField("workspace-456")
		if field.Key != "workspace" || field.Value.String() != "workspace-456" {
			t.Errorf("WorkspaceField failed: %+v", field)
		}
	})

	// Test application fields
	t.Run("ApplicationFields", func(t *testing.T) {
		field := ServiceField("guild-agent")
		if field.Key != "service" || field.Value.String() != "guild-agent" {
			t.Errorf("ServiceField failed: %+v", field)
		}

		field = VersionField("1.0.0")
		if field.Key != "version" || field.Value.String() != "1.0.0" {
			t.Errorf("VersionField failed: %+v", field)
		}

		field = EnvironmentField("production")
		if field.Key != "environment" || field.Value.String() != "production" {
			t.Errorf("EnvironmentField failed: %+v", field)
		}

		field = ComponentField("terminal")
		if field.Key != "component" || field.Value.String() != "terminal" {
			t.Errorf("ComponentField failed: %+v", field)
		}

		field = PackageField("logging")
		if field.Key != "package" || field.Value.String() != "logging" {
			t.Errorf("PackageField failed: %+v", field)
		}
	})

	// Test error fields
	t.Run("ErrorFields", func(t *testing.T) {
		field := ErrorCodeField("ERR_VALIDATION")
		if field.Key != "error_code" || field.Value.String() != "ERR_VALIDATION" {
			t.Errorf("ErrorCodeField failed: %+v", field)
		}

		field = ErrorTypeField("ValidationError")
		if field.Key != "error_type" || field.Value.String() != "ValidationError" {
			t.Errorf("ErrorTypeField failed: %+v", field)
		}

		field = RetryCountField(3)
		if field.Key != "retry_count" || field.Value.Int64() != 3 {
			t.Errorf("RetryCountField failed: %+v", field)
		}

		field = RetryableField(true)
		if field.Key != "retryable" || field.Value.Bool() != true {
			t.Errorf("RetryableField failed: %+v", field)
		}
	})

	// Test performance fields
	t.Run("PerformanceFields", func(t *testing.T) {
		field := LatencyField(50 * time.Millisecond)
		if field.Key != "latency" || field.Value.Duration() != 50*time.Millisecond {
			t.Errorf("LatencyField failed: %+v", field)
		}

		field = CPUUsageField(75.5)
		if field.Key != "cpu_usage" {
			t.Errorf("CPUUsageField failed: %+v", field)
		}
		// Check the value as Any type
		if val, ok := field.Value.Any().(float64); !ok || val != 75.5 {
			t.Errorf("CPUUsageField value failed: %+v", field)
		}

		field = MemoryUsageField(1024 * 1024)
		if field.Key != "memory_usage" || field.Value.Int64() != 1024*1024 {
			t.Errorf("MemoryUsageField failed: %+v", field)
		}

		field = GoroutinesField(100)
		if field.Key != "goroutines" || field.Value.Int64() != 100 {
			t.Errorf("GoroutinesField failed: %+v", field)
		}

		field = QueueSizeField(50)
		if field.Key != "queue_size" || field.Value.Int64() != 50 {
			t.Errorf("QueueSizeField failed: %+v", field)
		}

		field = CacheHitField(true)
		if field.Key != "cache_hit" || field.Value.Bool() != true {
			t.Errorf("CacheHitField failed: %+v", field)
		}
	})

	// Test database fields
	t.Run("DatabaseFields", func(t *testing.T) {
		field := QueryField("SELECT * FROM users")
		if field.Key != "query" || field.Value.String() != "SELECT * FROM users" {
			t.Errorf("QueryField failed: %+v", field)
		}

		field = QueryDurationField(25 * time.Millisecond)
		if field.Key != "query_duration" || field.Value.Duration() != 25*time.Millisecond {
			t.Errorf("QueryDurationField failed: %+v", field)
		}

		field = RowsAffectedField(10)
		if field.Key != "rows_affected" || field.Value.Int64() != 10 {
			t.Errorf("RowsAffectedField failed: %+v", field)
		}

		field = DatabaseField("guild_db")
		if field.Key != "database" || field.Value.String() != "guild_db" {
			t.Errorf("DatabaseField failed: %+v", field)
		}

		field = TableField("users")
		if field.Key != "table" || field.Value.String() != "users" {
			t.Errorf("TableField failed: %+v", field)
		}
	})

	// Test event fields
	t.Run("EventFields", func(t *testing.T) {
		field := EventTypeField("user.created")
		if field.Key != "event_type" || field.Value.String() != "user.created" {
			t.Errorf("EventTypeField failed: %+v", field)
		}

		field = EventIDField("evt-123")
		if field.Key != "event_id" || field.Value.String() != "evt-123" {
			t.Errorf("EventIDField failed: %+v", field)
		}

		field = MessageIDField("msg-456")
		if field.Key != "message_id" || field.Value.String() != "msg-456" {
			t.Errorf("MessageIDField failed: %+v", field)
		}

		field = CorrelationIDField("corr-789")
		if field.Key != "correlation_id" || field.Value.String() != "corr-789" {
			t.Errorf("CorrelationIDField failed: %+v", field)
		}

		field = PayloadSizeField(2048)
		if field.Key != "payload_size" || field.Value.Int64() != 2048 {
			t.Errorf("PayloadSizeField failed: %+v", field)
		}
	})

	// Test AI fields
	t.Run("AIFields", func(t *testing.T) {
		field := ProviderField("openai")
		if field.Key != "provider" || field.Value.String() != "openai" {
			t.Errorf("ProviderField failed: %+v", field)
		}

		field = ModelField("gpt-4")
		if field.Key != "model" || field.Value.String() != "gpt-4" {
			t.Errorf("ModelField failed: %+v", field)
		}

		field = TokensUsedField(150)
		if field.Key != "tokens_used" || field.Value.Int64() != 150 {
			t.Errorf("TokensUsedField failed: %+v", field)
		}

		field = CostField(0.05)
		if field.Key != "cost" {
			t.Errorf("CostField failed: %+v", field)
		}
		// Check the value as Any type
		if val, ok := field.Value.Any().(float64); !ok || val != 0.05 {
			t.Errorf("CostField value failed: %+v", field)
		}

		field = RateLimitField(100)
		if field.Key != "rate_limit" || field.Value.Int64() != 100 {
			t.Errorf("RateLimitField failed: %+v", field)
		}
	})
}

func TestFieldSetOperations(t *testing.T) {
	t.Run("BasicOperations", func(t *testing.T) {
		fs := NewFieldSet()

		// Test Add
		fs.Add(String("key1", "value1"))
		fields := fs.Fields()
		if len(fields) != 1 || fields[0].Key != "key1" || fields[0].Value.String() != "value1" {
			t.Errorf("Add failed: %+v", fields)
		}

		// Test AddIf
		fs.AddIf(true, String("key2", "value2"))
		fs.AddIf(false, String("key3", "value3"))
		fields = fs.Fields()
		if len(fields) != 2 {
			t.Errorf("AddIf failed: expected 2 fields, got %d", len(fields))
		}

		// Test AddString
		fs.AddString("key4", "value4")
		fields = fs.Fields()
		if len(fields) != 3 {
			t.Errorf("AddString failed: expected 3 fields, got %d", len(fields))
		}

		// Test AddInt
		fs.AddInt("count", 42)
		fields = fs.Fields()
		found := false
		for _, f := range fields {
			if f.Key == "count" && f.Value.Int64() == 42 {
				found = true
				break
			}
		}
		if !found {
			t.Error("AddInt failed")
		}

		// Test AddDuration
		duration := 100 * time.Millisecond
		fs.AddDuration("duration", duration)
		fields = fs.Fields()
		found = false
		for _, f := range fields {
			if f.Key == "duration" && f.Value.Duration() == duration {
				found = true
				break
			}
		}
		if !found {
			t.Error("AddDuration failed")
		}

		// Test AddError
		testErr := errors.New("test error")
		fs.AddError(testErr)
		fields = fs.Fields()
		found = false
		for _, f := range fields {
			if f.Key == "error" {
				// ErrorField stores errors as their native type
				if err, ok := f.Value.Any().(error); ok && err.Error() == testErr.Error() {
					found = true
					break
				}
			}
		}
		if !found {
			t.Error("AddError failed")
		}
	})

	t.Run("NilError", func(t *testing.T) {
		fs := NewFieldSet()
		fs.AddError(nil)
		fields := fs.Fields()

		// Should not add nil errors
		for _, f := range fields {
			if f.Key == "error" {
				t.Error("Should not add nil error")
			}
		}
	})
}
