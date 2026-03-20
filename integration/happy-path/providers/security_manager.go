// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package providers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/providers"
	"github.com/lancekrogers/guild-core/pkg/registry"
)

// SecurityManager manages security for provider integration
type SecurityManager struct {
	registry        registry.ComponentRegistry
	credentialStore *CredentialStore
	authManager     *AuthenticationManager
	securityMonitor *SecurityMonitor
	keyRotator      *KeyRotator
	auditLogger     *AuditLogger
	running         bool
	mu              sync.RWMutex
}

// CredentialStore manages secure credential storage
type CredentialStore struct {
	credentials map[providers.ProviderType]*ProviderCredentials
	encryption  *EncryptionManager
	mu          sync.RWMutex
}

// ProviderCredentials stores credentials for a provider
type ProviderCredentials struct {
	Provider       providers.ProviderType
	APIKey         string
	SecretKey      string
	TokenType      TokenType
	ExpiresAt      *time.Time
	LastRotated    time.Time
	RotationPolicy RotationPolicy
	Encrypted      bool
	Metadata       map[string]string
}

// TokenType represents different token types
type TokenType int

const (
	TokenTypeAPIKey TokenType = iota
	TokenTypeOAuth
	TokenTypeBearer
	TokenTypeBasic
)

// RotationPolicy defines credential rotation behavior
type RotationPolicy struct {
	Enabled           bool
	RotationInterval  time.Duration
	NotificationDays  int
	AutoRotate        bool
	BackupCredentials bool
}

// AuthenticationManager manages authentication flows
type AuthenticationManager struct {
	providers     map[providers.ProviderType]*ProviderAuthConfig
	tokenCache    map[string]*CachedToken
	refreshTokens map[providers.ProviderType]string
	mu            sync.RWMutex
}

// ProviderAuthConfig configures authentication for a provider
type ProviderAuthConfig struct {
	Provider       providers.ProviderType
	AuthType       AuthenticationType
	Endpoint       string
	Scopes         []string
	TokenExpiry    time.Duration
	RefreshEnabled bool
	RetryConfig    AuthRetryConfig
}

// AuthenticationType represents authentication types
type AuthenticationType int

const (
	AuthenticationTypeAPIKey AuthenticationType = iota
	AuthenticationTypeOAuth2
	AuthenticationTypeJWT
	AuthenticationTypeBasic
)

// AuthRetryConfig configures authentication retry behavior
type AuthRetryConfig struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

// CachedToken represents a cached authentication token
type CachedToken struct {
	Token     string
	ExpiresAt time.Time
	Provider  providers.ProviderType
	Scopes    []string
}

// SecurityMonitor monitors security events and anomalies
type SecurityMonitor struct {
	events          []SecurityEvent
	anomalies       []SecurityAnomaly
	alertThresholds map[SecurityEventType]int
	monitoring      bool
	mu              sync.RWMutex
}

// SecurityEvent represents a security event
type SecurityEvent struct {
	ID          string
	Timestamp   time.Time
	Type        SecurityEventType
	Provider    providers.ProviderType
	Severity    SecuritySeverity
	Description string
	Metadata    map[string]interface{}
	Resolved    bool
}

// SecurityEventType represents types of security events
type SecurityEventType int

const (
	SecurityEventTypeCredentialAccess SecurityEventType = iota
	SecurityEventTypeUnauthorizedRequest
	SecurityEventTypeCredentialRotation
	SecurityEventTypeAnomalousUsage
	SecurityEventTypeAuthenticationFailure
	SecurityEventTypeAPIQuotaExceeded
)

// SecuritySeverity represents security event severity
type SecuritySeverity int

const (
	SecuritySeverityLow SecuritySeverity = iota
	SecuritySeverityMedium
	SecuritySeverityHigh
	SecuritySeverityCritical
)

// SecurityAnomaly represents detected security anomalies
type SecurityAnomaly struct {
	ID          string
	Timestamp   time.Time
	Provider    providers.ProviderType
	AnomalyType AnomalyType
	Confidence  float64
	Description string
	Baseline    float64
	Observed    float64
	ActionTaken string
}

// AnomalyType represents types of security anomalies
type AnomalyType int

const (
	AnomalyTypeUsagePattern AnomalyType = iota
	AnomalyTypeRequestFrequency
	AnomalyTypeErrorRate
	AnomalyTypeUnusualTiming
	AnomalyTypeGeographicLocation
)

// KeyRotator manages automatic key rotation
type KeyRotator struct {
	rotationSchedule map[providers.ProviderType]*RotationSchedule
	credentialStore  *CredentialStore
	notifications    chan RotationNotification
	mu               sync.RWMutex
}

// RotationSchedule tracks rotation schedule for a provider
type RotationSchedule struct {
	Provider       providers.ProviderType
	NextRotation   time.Time
	LastRotation   time.Time
	RotationPolicy RotationPolicy
	Enabled        bool
}

// RotationNotification represents a rotation notification
type RotationNotification struct {
	Provider  providers.ProviderType
	Type      NotificationType
	DaysUntil int
	Message   string
	Timestamp time.Time
}

// NotificationType represents notification types
type NotificationType int

const (
	NotificationTypeWarning NotificationType = iota
	NotificationTypeRotated
	NotificationTypeFailed
)

// AuditLogger logs security audit events
type AuditLogger struct {
	entries []AuditEntry
	mu      sync.RWMutex
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	ID        string
	Timestamp time.Time
	Action    AuditAction
	Provider  providers.ProviderType
	UserID    string
	IPAddress string
	UserAgent string
	Success   bool
	Details   map[string]interface{}
}

// AuditAction represents auditable actions
type AuditAction int

const (
	AuditActionCredentialAccess AuditAction = iota
	AuditActionCredentialUpdate
	AuditActionCredentialRotation
	AuditActionAuthentication
	AuditActionProviderRequest
	AuditActionSecurityEvent
)

// EncryptionManager manages encryption for sensitive data
type EncryptionManager struct {
	key []byte
	mu  sync.RWMutex
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(registry registry.ComponentRegistry) (*SecurityManager, error) {
	sm := &SecurityManager{
		registry: registry,
	}

	// Initialize credential store
	credentialStore, err := NewCredentialStore()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create credential store")
	}
	sm.credentialStore = credentialStore

	// Initialize authentication manager
	authManager, err := NewAuthenticationManager()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create authentication manager")
	}
	sm.authManager = authManager

	// Initialize security monitor
	securityMonitor, err := NewSecurityMonitor()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create security monitor")
	}
	sm.securityMonitor = securityMonitor

	// Initialize key rotator
	keyRotator, err := NewKeyRotator(credentialStore)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create key rotator")
	}
	sm.keyRotator = keyRotator

	// Initialize audit logger
	auditLogger, err := NewAuditLogger()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create audit logger")
	}
	sm.auditLogger = auditLogger

	return sm, nil
}

// NewCredentialStore creates a new credential store
func NewCredentialStore() (*CredentialStore, error) {
	encryption, err := NewEncryptionManager()
	if err != nil {
		return nil, err
	}

	store := &CredentialStore{
		credentials: make(map[providers.ProviderType]*ProviderCredentials),
		encryption:  encryption,
	}

	// Initialize with mock credentials for testing
	err = store.initializeTestCredentials()
	if err != nil {
		return nil, err
	}

	return store, nil
}

// initializeTestCredentials initializes mock credentials for testing
func (cs *CredentialStore) initializeTestCredentials() error {
	testCredentials := []struct {
		provider providers.ProviderType
		apiKey   string
		policy   RotationPolicy
	}{
		{
			provider: providers.ProviderOpenAI,
			apiKey:   "sk-test-openai-key-" + generateRandomString(32),
			policy: RotationPolicy{
				Enabled:          true,
				RotationInterval: 30 * 24 * time.Hour, // 30 days
				AutoRotate:       false,
			},
		},
		{
			provider: providers.ProviderAnthropic,
			apiKey:   "sk-ant-test-" + generateRandomString(48),
			policy: RotationPolicy{
				Enabled:          true,
				RotationInterval: 60 * 24 * time.Hour, // 60 days
				AutoRotate:       false,
			},
		},
		{
			provider: providers.ProviderDeepSeek,
			apiKey:   "sk-deepseek-" + generateRandomString(40),
			policy: RotationPolicy{
				Enabled:          true,
				RotationInterval: 90 * 24 * time.Hour, // 90 days
				AutoRotate:       true,
			},
		},
		{
			provider: providers.ProviderOra,
			apiKey:   "ora-api-" + generateRandomString(36),
			policy: RotationPolicy{
				Enabled:          false,
				RotationInterval: 0,
				AutoRotate:       false,
			},
		},
	}

	for _, cred := range testCredentials {
		err := cs.StoreCredentials(cred.provider, cred.apiKey, "", TokenTypeAPIKey, nil, cred.policy)
		if err != nil {
			return err
		}
	}

	return nil
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:length]
}

// NewEncryptionManager creates a new encryption manager
func NewEncryptionManager() (*EncryptionManager, error) {
	// Generate a random encryption key for testing
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to generate encryption key")
	}

	return &EncryptionManager{
		key: key,
	}, nil
}

// NewAuthenticationManager creates a new authentication manager
func NewAuthenticationManager() (*AuthenticationManager, error) {
	am := &AuthenticationManager{
		providers:     make(map[providers.ProviderType]*ProviderAuthConfig),
		tokenCache:    make(map[string]*CachedToken),
		refreshTokens: make(map[providers.ProviderType]string),
	}

	// Initialize provider auth configs
	am.initializeProviderConfigs()

	return am, nil
}

// initializeProviderConfigs initializes authentication configs for providers
func (am *AuthenticationManager) initializeProviderConfigs() {
	configs := map[providers.ProviderType]*ProviderAuthConfig{
		providers.ProviderOpenAI: {
			Provider:       providers.ProviderOpenAI,
			AuthType:       AuthenticationTypeAPIKey,
			Endpoint:       "https://api.openai.com/v1",
			TokenExpiry:    24 * time.Hour,
			RefreshEnabled: false,
			RetryConfig: AuthRetryConfig{
				MaxRetries:    3,
				InitialDelay:  time.Second,
				MaxDelay:      10 * time.Second,
				BackoffFactor: 2.0,
			},
		},
		providers.ProviderAnthropic: {
			Provider:       providers.ProviderAnthropic,
			AuthType:       AuthenticationTypeAPIKey,
			Endpoint:       "https://api.anthropic.com/v1",
			TokenExpiry:    24 * time.Hour,
			RefreshEnabled: false,
			RetryConfig: AuthRetryConfig{
				MaxRetries:    3,
				InitialDelay:  time.Second,
				MaxDelay:      10 * time.Second,
				BackoffFactor: 2.0,
			},
		},
		providers.ProviderOllama: {
			Provider:       providers.ProviderOllama,
			AuthType:       AuthenticationTypeAPIKey,
			Endpoint:       "http://localhost:11434/api",
			TokenExpiry:    0, // No expiry for local
			RefreshEnabled: false,
			RetryConfig: AuthRetryConfig{
				MaxRetries:    2,
				InitialDelay:  500 * time.Millisecond,
				MaxDelay:      5 * time.Second,
				BackoffFactor: 1.5,
			},
		},
	}

	am.mu.Lock()
	defer am.mu.Unlock()
	am.providers = configs
}

// NewSecurityMonitor creates a new security monitor
func NewSecurityMonitor() (*SecurityMonitor, error) {
	monitor := &SecurityMonitor{
		events:    make([]SecurityEvent, 0),
		anomalies: make([]SecurityAnomaly, 0),
		alertThresholds: map[SecurityEventType]int{
			SecurityEventTypeCredentialAccess:      10, // Alert after 10 accesses in short time
			SecurityEventTypeUnauthorizedRequest:   5,  // Alert after 5 unauthorized requests
			SecurityEventTypeAuthenticationFailure: 3,  // Alert after 3 auth failures
			SecurityEventTypeAnomalousUsage:        1,  // Alert immediately on anomalous usage
			SecurityEventTypeAPIQuotaExceeded:      1,  // Alert immediately on quota exceeded
		},
	}

	return monitor, nil
}

// NewKeyRotator creates a new key rotator
func NewKeyRotator(credentialStore *CredentialStore) (*KeyRotator, error) {
	rotator := &KeyRotator{
		rotationSchedule: make(map[providers.ProviderType]*RotationSchedule),
		credentialStore:  credentialStore,
		notifications:    make(chan RotationNotification, 100),
	}

	// Initialize rotation schedules
	rotator.initializeRotationSchedules()

	return rotator, nil
}

// initializeRotationSchedules initializes rotation schedules for providers
func (kr *KeyRotator) initializeRotationSchedules() {
	kr.mu.Lock()
	defer kr.mu.Unlock()

	providerTypes := []providers.ProviderType{
		providers.ProviderOpenAI,
		providers.ProviderAnthropic,
		providers.ProviderDeepSeek,
		providers.ProviderOra,
	}

	for _, provider := range providerTypes {
		credentials, err := kr.credentialStore.GetCredentials(provider)
		if err != nil {
			continue
		}

		schedule := &RotationSchedule{
			Provider:       provider,
			LastRotation:   credentials.LastRotated,
			RotationPolicy: credentials.RotationPolicy,
			Enabled:        credentials.RotationPolicy.Enabled,
		}

		if schedule.Enabled {
			schedule.NextRotation = credentials.LastRotated.Add(credentials.RotationPolicy.RotationInterval)
		}

		kr.rotationSchedule[provider] = schedule
	}
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() (*AuditLogger, error) {
	return &AuditLogger{
		entries: make([]AuditEntry, 0),
	}, nil
}

// Start starts the security manager
func (sm *SecurityManager) Start(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.running {
		return gerror.New(gerror.ErrCodeConflict, "security manager already running", nil)
	}

	// Start security monitoring
	go sm.securityMonitor.Start(ctx)

	// Start key rotation monitoring
	go sm.keyRotator.Start(ctx)

	sm.running = true
	return nil
}

// Stop stops the security manager
func (sm *SecurityManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.running = false
	return nil
}

// StoreCredentials stores credentials for a provider
func (cs *CredentialStore) StoreCredentials(provider providers.ProviderType, apiKey, secretKey string, tokenType TokenType, expiresAt *time.Time, policy RotationPolicy) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Encrypt sensitive data
	encryptedAPIKey, err := cs.encryption.Encrypt(apiKey)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to encrypt API key")
	}

	encryptedSecretKey := ""
	if secretKey != "" {
		encryptedSecretKey, err = cs.encryption.Encrypt(secretKey)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to encrypt secret key")
		}
	}

	credentials := &ProviderCredentials{
		Provider:       provider,
		APIKey:         encryptedAPIKey,
		SecretKey:      encryptedSecretKey,
		TokenType:      tokenType,
		ExpiresAt:      expiresAt,
		LastRotated:    time.Now(),
		RotationPolicy: policy,
		Encrypted:      true,
		Metadata:       make(map[string]string),
	}

	cs.credentials[provider] = credentials
	return nil
}

// GetCredentials retrieves credentials for a provider
func (cs *CredentialStore) GetCredentials(provider providers.ProviderType) (*ProviderCredentials, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	credentials, exists := cs.credentials[provider]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, fmt.Sprintf("credentials for %s not found", provider), nil)
	}

	// Decrypt sensitive data
	decryptedAPIKey, err := cs.encryption.Decrypt(credentials.APIKey)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to decrypt API key")
	}

	decryptedSecretKey := ""
	if credentials.SecretKey != "" {
		decryptedSecretKey, err = cs.encryption.Decrypt(credentials.SecretKey)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to decrypt secret key")
		}
	}

	// Return decrypted copy
	return &ProviderCredentials{
		Provider:       credentials.Provider,
		APIKey:         decryptedAPIKey,
		SecretKey:      decryptedSecretKey,
		TokenType:      credentials.TokenType,
		ExpiresAt:      credentials.ExpiresAt,
		LastRotated:    credentials.LastRotated,
		RotationPolicy: credentials.RotationPolicy,
		Encrypted:      false,
		Metadata:       credentials.Metadata,
	}, nil
}

// Authenticate authenticates with a provider
func (am *AuthenticationManager) Authenticate(ctx context.Context, provider providers.ProviderType, credentials *ProviderCredentials) (string, error) {
	am.mu.RLock()
	config, exists := am.providers[provider]
	am.mu.RUnlock()

	if !exists {
		return "", gerror.New(gerror.ErrCodeNotFound, fmt.Sprintf("auth config for %s not found", provider), nil)
	}

	// Check token cache first
	cacheKey := string(provider)
	am.mu.RLock()
	cached, exists := am.tokenCache[cacheKey]
	am.mu.RUnlock()

	if exists && time.Now().Before(cached.ExpiresAt) {
		return cached.Token, nil
	}

	// Perform authentication based on type
	switch config.AuthType {
	case AuthenticationTypeAPIKey:
		return credentials.APIKey, nil
	case AuthenticationTypeOAuth2:
		return am.authenticateOAuth2(ctx, provider, config, credentials)
	case AuthenticationTypeJWT:
		return am.authenticateJWT(ctx, provider, config, credentials)
	default:
		return "", gerror.New(gerror.ErrCodeInternal, fmt.Sprintf("unsupported auth type for %s", provider), nil)
	}
}

// authenticateOAuth2 performs OAuth2 authentication
func (am *AuthenticationManager) authenticateOAuth2(ctx context.Context, provider providers.ProviderType, config *ProviderAuthConfig, credentials *ProviderCredentials) (string, error) {
	// Mock OAuth2 flow for testing
	token := "oauth2-token-" + generateRandomString(32)
	expiresAt := time.Now().Add(config.TokenExpiry)

	// Cache token
	am.mu.Lock()
	am.tokenCache[string(provider)] = &CachedToken{
		Token:     token,
		ExpiresAt: expiresAt,
		Provider:  provider,
		Scopes:    config.Scopes,
	}
	am.mu.Unlock()

	return token, nil
}

// authenticateJWT performs JWT authentication
func (am *AuthenticationManager) authenticateJWT(ctx context.Context, provider providers.ProviderType, config *ProviderAuthConfig, credentials *ProviderCredentials) (string, error) {
	// Mock JWT creation for testing
	token := "jwt-token-" + generateRandomString(48)
	expiresAt := time.Now().Add(config.TokenExpiry)

	// Cache token
	am.mu.Lock()
	am.tokenCache[string(provider)] = &CachedToken{
		Token:     token,
		ExpiresAt: expiresAt,
		Provider:  provider,
		Scopes:    config.Scopes,
	}
	am.mu.Unlock()

	return token, nil
}

// Encrypt encrypts data
func (em *EncryptionManager) Encrypt(data string) (string, error) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	// Simple encryption for testing - in production use proper encryption
	hash := sha256.Sum256([]byte(data + string(em.key)))
	return base64.StdEncoding.EncodeToString(hash[:]), nil
}

// Decrypt decrypts data
func (em *EncryptionManager) Decrypt(encryptedData string) (string, error) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	// For testing, return a mock decrypted value
	// In production, implement proper decryption
	decoded, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", err
	}

	// Return a deterministic "decrypted" value for testing
	return fmt.Sprintf("decrypted-%x", decoded[:8]), nil
}

// RecordSecurityEvent records a security event
func (sm *SecurityManager) RecordSecurityEvent(eventType SecurityEventType, provider providers.ProviderType, severity SecuritySeverity, description string, metadata map[string]interface{}) {
	sm.securityMonitor.RecordEvent(eventType, provider, severity, description, metadata)

	// Log to audit trail
	sm.auditLogger.LogAction(AuditActionSecurityEvent, provider, "", "", "", true, map[string]interface{}{
		"event_type":  eventType,
		"severity":    severity,
		"description": description,
	})
}

// RecordEvent records a security event
func (sm *SecurityMonitor) RecordEvent(eventType SecurityEventType, provider providers.ProviderType, severity SecuritySeverity, description string, metadata map[string]interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	event := SecurityEvent{
		ID:          fmt.Sprintf("event-%d", time.Now().UnixNano()),
		Timestamp:   time.Now(),
		Type:        eventType,
		Provider:    provider,
		Severity:    severity,
		Description: description,
		Metadata:    metadata,
		Resolved:    false,
	}

	sm.events = append(sm.events, event)

	// Keep only last 1000 events
	if len(sm.events) > 1000 {
		sm.events = sm.events[len(sm.events)-1000:]
	}

	// Check if alert threshold is reached
	if threshold, exists := sm.alertThresholds[eventType]; exists {
		recentEvents := sm.countRecentEvents(eventType, time.Hour)
		if recentEvents >= threshold {
			sm.triggerAlert(eventType, provider, recentEvents)
		}
	}
}

// countRecentEvents counts recent events of a specific type
func (sm *SecurityMonitor) countRecentEvents(eventType SecurityEventType, window time.Duration) int {
	cutoff := time.Now().Add(-window)
	count := 0

	for _, event := range sm.events {
		if event.Type == eventType && event.Timestamp.After(cutoff) {
			count++
		}
	}

	return count
}

// triggerAlert triggers a security alert
func (sm *SecurityMonitor) triggerAlert(eventType SecurityEventType, provider providers.ProviderType, count int) {
	// In production, this would send alerts to monitoring systems
	fmt.Printf("🚨 SECURITY ALERT: %d %v events for %s in the last hour\n", count, eventType, provider)
}

// LogAction logs an audit action
func (al *AuditLogger) LogAction(action AuditAction, provider providers.ProviderType, userID, ipAddress, userAgent string, success bool, details map[string]interface{}) {
	al.mu.Lock()
	defer al.mu.Unlock()

	entry := AuditEntry{
		ID:        fmt.Sprintf("audit-%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Action:    action,
		Provider:  provider,
		UserID:    userID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   success,
		Details:   details,
	}

	al.entries = append(al.entries, entry)

	// Keep only last 10000 entries
	if len(al.entries) > 10000 {
		al.entries = al.entries[len(al.entries)-10000:]
	}
}

// Start starts security monitoring
func (sm *SecurityMonitor) Start(ctx context.Context) {
	sm.mu.Lock()
	sm.monitoring = true
	sm.mu.Unlock()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			sm.mu.Lock()
			sm.monitoring = false
			sm.mu.Unlock()
			return
		case <-ticker.C:
			sm.detectAnomalies()
		}
	}
}

// detectAnomalies detects security anomalies
func (sm *SecurityMonitor) detectAnomalies() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Simple anomaly detection - in production use ML models
	now := time.Now()
	hourAgo := now.Add(-time.Hour)

	// Detect unusual request patterns
	eventCounts := make(map[SecurityEventType]int)
	for _, event := range sm.events {
		if event.Timestamp.After(hourAgo) {
			eventCounts[event.Type]++
		}
	}

	for eventType, count := range eventCounts {
		if count > 50 { // Threshold for anomaly
			anomaly := SecurityAnomaly{
				ID:          fmt.Sprintf("anomaly-%d", time.Now().UnixNano()),
				Timestamp:   now,
				AnomalyType: AnomalyTypeRequestFrequency,
				Confidence:  0.8,
				Description: fmt.Sprintf("Unusual frequency of %v events: %d in last hour", eventType, count),
				Baseline:    10.0,
				Observed:    float64(count),
				ActionTaken: "alert_triggered",
			}

			sm.anomalies = append(sm.anomalies, anomaly)
		}
	}
}

// Start starts key rotation monitoring
func (kr *KeyRotator) Start(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour) // Check daily
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			kr.checkRotationSchedules()
		}
	}
}

// checkRotationSchedules checks if any keys need rotation
func (kr *KeyRotator) checkRotationSchedules() {
	kr.mu.RLock()
	schedules := make(map[providers.ProviderType]*RotationSchedule)
	for k, v := range kr.rotationSchedule {
		schedules[k] = v
	}
	kr.mu.RUnlock()

	now := time.Now()

	for provider, schedule := range schedules {
		if !schedule.Enabled {
			continue
		}

		// Check if rotation is due
		if now.After(schedule.NextRotation) {
			if schedule.RotationPolicy.AutoRotate {
				err := kr.rotateCredentials(provider)
				if err != nil {
					kr.notifications <- RotationNotification{
						Provider:  provider,
						Type:      NotificationTypeFailed,
						Message:   fmt.Sprintf("Automatic rotation failed for %s: %v", provider, err),
						Timestamp: now,
					}
				} else {
					kr.notifications <- RotationNotification{
						Provider:  provider,
						Type:      NotificationTypeRotated,
						Message:   fmt.Sprintf("Credentials rotated for %s", provider),
						Timestamp: now,
					}
				}
			} else {
				kr.notifications <- RotationNotification{
					Provider:  provider,
					Type:      NotificationTypeWarning,
					Message:   fmt.Sprintf("Manual rotation required for %s", provider),
					Timestamp: now,
				}
			}
		} else {
			// Check for advance warning
			daysUntil := int(schedule.NextRotation.Sub(now).Hours() / 24)
			if daysUntil <= schedule.RotationPolicy.NotificationDays && daysUntil > 0 {
				kr.notifications <- RotationNotification{
					Provider:  provider,
					Type:      NotificationTypeWarning,
					DaysUntil: daysUntil,
					Message:   fmt.Sprintf("Credentials for %s expire in %d days", provider, daysUntil),
					Timestamp: now,
				}
			}
		}
	}
}

// rotateCredentials rotates credentials for a provider
func (kr *KeyRotator) rotateCredentials(provider providers.ProviderType) error {
	// Mock credential rotation - in production this would call provider APIs
	newAPIKey := "rotated-key-" + generateRandomString(32)

	credentials, err := kr.credentialStore.GetCredentials(provider)
	if err != nil {
		return err
	}

	// Update with new credentials
	err = kr.credentialStore.StoreCredentials(
		provider,
		newAPIKey,
		credentials.SecretKey,
		credentials.TokenType,
		credentials.ExpiresAt,
		credentials.RotationPolicy,
	)
	if err != nil {
		return err
	}

	// Update rotation schedule
	kr.mu.Lock()
	if schedule, exists := kr.rotationSchedule[provider]; exists {
		schedule.LastRotation = time.Now()
		schedule.NextRotation = time.Now().Add(schedule.RotationPolicy.RotationInterval)
	}
	kr.mu.Unlock()

	return nil
}

// GetSecurityEvents returns security events
func (sm *SecurityManager) GetSecurityEvents() []SecurityEvent {
	return sm.securityMonitor.GetEvents()
}

// GetEvents returns security events
func (sm *SecurityMonitor) GetEvents() []SecurityEvent {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	events := make([]SecurityEvent, len(sm.events))
	copy(events, sm.events)
	return events
}

// GetAuditLog returns audit log entries
func (sm *SecurityManager) GetAuditLog() []AuditEntry {
	return sm.auditLogger.GetEntries()
}

// GetEntries returns audit log entries
func (al *AuditLogger) GetEntries() []AuditEntry {
	al.mu.RLock()
	defer al.mu.RUnlock()

	entries := make([]AuditEntry, len(al.entries))
	copy(entries, al.entries)
	return entries
}

// GetCredentials gets credentials for a provider
func (sm *SecurityManager) GetCredentials(provider providers.ProviderType) (*ProviderCredentials, error) {
	// Record access in audit log
	sm.auditLogger.LogAction(AuditActionCredentialAccess, provider, "", "", "", true, map[string]interface{}{
		"access_time": time.Now(),
	})

	// Record security event
	sm.RecordSecurityEvent(SecurityEventTypeCredentialAccess, provider, SecuritySeverityLow, "Credentials accessed", map[string]interface{}{
		"access_method": "api",
	})

	return sm.credentialStore.GetCredentials(provider)
}

// AuthenticateProvider authenticates with a provider
func (sm *SecurityManager) AuthenticateProvider(ctx context.Context, provider providers.ProviderType) (string, error) {
	credentials, err := sm.GetCredentials(provider)
	if err != nil {
		sm.RecordSecurityEvent(SecurityEventTypeAuthenticationFailure, provider, SecuritySeverityMedium, "Failed to get credentials", map[string]interface{}{
			"error": err.Error(),
		})
		return "", err
	}

	token, err := sm.authManager.Authenticate(ctx, provider, credentials)
	if err != nil {
		sm.RecordSecurityEvent(SecurityEventTypeAuthenticationFailure, provider, SecuritySeverityMedium, "Authentication failed", map[string]interface{}{
			"error": err.Error(),
		})
		return "", err
	}

	// Log successful authentication
	sm.auditLogger.LogAction(AuditActionAuthentication, provider, "", "", "", true, map[string]interface{}{
		"auth_time":  time.Now(),
		"token_type": credentials.TokenType,
	})

	return token, nil
}

// GetSecurityStats returns security statistics
func (sm *SecurityManager) GetSecurityStats() map[string]interface{} {
	events := sm.securityMonitor.GetEvents()

	eventCounts := make(map[string]int)
	severityCounts := make(map[string]int)

	for _, event := range events {
		eventCounts[fmt.Sprintf("%d", event.Type)]++
		severityCounts[fmt.Sprintf("%d", event.Severity)]++
	}

	return map[string]interface{}{
		"total_events":     len(events),
		"event_counts":     eventCounts,
		"severity_counts":  severityCounts,
		"monitoring":       sm.securityMonitor.monitoring,
		"credential_store": len(sm.credentialStore.credentials),
	}
}
