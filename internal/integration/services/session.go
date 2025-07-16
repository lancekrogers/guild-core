package services

import (
	"context"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/session"
)

// SessionService provides session management as a service
type SessionService struct {
	manager session.SessionManagerInterface
	logger  observability.Logger

	// Configuration
	config SessionServiceConfig

	// State
	started bool
	mu      sync.RWMutex

	// Metrics
	activeSessions   int
	totalSessions    uint64
	sessionsRestored uint64
}

// SessionServiceConfig configures the session service
type SessionServiceConfig struct {
	// MaxActiveSessions limits concurrent sessions
	MaxActiveSessions int

	// SessionTimeout for inactive sessions
	SessionTimeout time.Duration

	// PersistSessions enables session persistence
	PersistSessions bool

	// RestoreOnStartup attempts to restore previous sessions
	RestoreOnStartup bool

	// CleanupInterval for expired sessions
	CleanupInterval time.Duration
}

// DefaultSessionServiceConfig returns default configuration
func DefaultSessionServiceConfig() SessionServiceConfig {
	return SessionServiceConfig{
		MaxActiveSessions: 100,
		SessionTimeout:    24 * time.Hour,
		PersistSessions:   true,
		RestoreOnStartup:  true,
		CleanupInterval:   1 * time.Hour,
	}
}

// NewSessionService creates a new session service
func NewSessionService(manager session.SessionManagerInterface, logger observability.Logger, config SessionServiceConfig) *SessionService {
	return &SessionService{
		manager: manager,
		logger:  logger,
		config:  config,
	}
}

// Name returns the service name
func (s *SessionService) Name() string {
	return "session-service"
}

// Start initializes and starts the service
func (s *SessionService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return gerror.New(gerror.ErrCodeAlreadyExists, "service already started", nil).
			WithComponent("session_service")
	}

	// Restore sessions if configured
	if s.config.RestoreOnStartup && s.config.PersistSessions {
		if resumer, ok := s.manager.(session.SessionResumerInterface); ok {
			// TODO: Need a way to get the user ID for session restoration
			// For now, we'll skip automatic restoration
			s.logger.WarnContext(ctx, "Session restoration not implemented - need user ID")
			_ = resumer // Suppress unused variable warning
		}
	}

	// Start cleanup goroutine
	go s.cleanupLoop(ctx)

	// Update metrics
	sessions, err := s.manager.ListSessions(ctx, session.ListOptions{})
	if err != nil {
		s.logger.WarnContext(ctx, "Failed to list sessions for metrics", "error", err)
		s.activeSessions = 0
	} else {
		s.activeSessions = len(sessions)
	}
	s.totalSessions = uint64(s.activeSessions)

	s.started = true
	s.logger.InfoContext(ctx, "Session service started",
		"active_sessions", s.activeSessions,
		"sessions_restored", s.sessionsRestored)

	return nil
}

// Stop gracefully shuts down the service
func (s *SessionService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("session_service")
	}

	// Persist all active sessions if configured
	if s.config.PersistSessions {
		sessions, err := s.manager.ListSessions(ctx, session.ListOptions{})
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to list sessions for persistence", "error", err)
		} else {
			persisted := 0
			for _, sess := range sessions {
				if err := s.manager.SaveSession(ctx, sess); err != nil {
					s.logger.ErrorContext(ctx, "Failed to persist session",
						"session_id", sess.ID,
						"error", err)
				} else {
					persisted++
				}
			}
			s.logger.InfoContext(ctx, "Persisted sessions", "count", persisted)
		}
	}

	s.started = false
	s.logger.InfoContext(ctx, "Session service stopped",
		"total_sessions", s.totalSessions)

	return nil
}

// Health checks if the service is healthy
func (s *SessionService) Health(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeResourceExhausted, "service not started", nil).
			WithComponent("session_service")
	}

	// Check if we're at capacity
	if s.config.MaxActiveSessions > 0 && s.activeSessions >= s.config.MaxActiveSessions {
		return gerror.New(gerror.ErrCodeResourceExhausted, "max sessions reached", nil).
			WithComponent("session_service").
			WithDetails("active", s.activeSessions).
			WithDetails("max", s.config.MaxActiveSessions)
	}

	return nil
}

// Ready checks if the service is ready
func (s *SessionService) Ready(ctx context.Context) error {
	return s.Health(ctx)
}

// CreateSession creates a new managed session
func (s *SessionService) CreateSession(ctx context.Context, userID, campaignID string) (*session.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil, gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("session_service")
	}

	// Check capacity
	if s.config.MaxActiveSessions > 0 && s.activeSessions >= s.config.MaxActiveSessions {
		return nil, gerror.New(gerror.ErrCodeResourceExhausted, "max sessions reached", nil).
			WithComponent("session_service")
	}

	// Create session
	sess, err := s.manager.CreateSession(ctx, userID, campaignID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create session").
			WithComponent("session_service")
	}

	s.activeSessions++
	s.totalSessions++

	s.logger.InfoContext(ctx, "Session created",
		"session_id", sess.ID,
		"user_id", userID,
		"campaign_id", campaignID,
		"active_sessions", s.activeSessions)

	return sess, nil
}

// LoadSession retrieves a session
func (s *SessionService) LoadSession(ctx context.Context, sessionID string) (*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return nil, gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("session_service")
	}

	return s.manager.LoadSession(ctx, sessionID)
}

// ListSessions returns all active sessions
func (s *SessionService) ListSessions(ctx context.Context) ([]*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return nil, gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("session_service")
	}

	return s.manager.ListSessions(ctx, session.ListOptions{})
}

// DeleteSession deletes a session
func (s *SessionService) DeleteSession(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("session_service")
	}

	// Get session for persistence before deleting
	sess, _ := s.manager.LoadSession(ctx, sessionID)

	// Delete session
	if err := s.manager.DeleteSession(ctx, sessionID); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to delete session").
			WithComponent("session_service")
	}

	// Persist if configured (as a backup before deletion)
	if s.config.PersistSessions && sess != nil {
		if err := s.manager.SaveSession(ctx, sess); err != nil {
			s.logger.WarnContext(ctx, "Failed to persist session before deletion",
				"session_id", sessionID,
				"error", err)
		}
	}

	s.activeSessions--
	s.logger.InfoContext(ctx, "Session deleted",
		"session_id", sessionID,
		"active_sessions", s.activeSessions)

	return nil
}

// cleanupLoop periodically cleans up expired sessions
func (s *SessionService) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanupExpiredSessions(ctx)
		}
	}
}

// cleanupExpiredSessions removes expired sessions
func (s *SessionService) cleanupExpiredSessions(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions, err := s.manager.ListSessions(ctx, session.ListOptions{})
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to list sessions for cleanup", "error", err)
		return
	}

	expired := 0

	for _, sess := range sessions {
		if sess.LastActiveTime.Add(s.config.SessionTimeout).Before(time.Now()) {
			if err := s.manager.DeleteSession(ctx, sess.ID); err != nil {
				s.logger.ErrorContext(ctx, "Failed to cleanup expired session",
					"session_id", sess.ID,
					"error", err)
			} else {
				expired++
				s.activeSessions--
			}
		}
	}

	if expired > 0 {
		s.logger.InfoContext(ctx, "Cleaned up expired sessions",
			"count", expired,
			"active_sessions", s.activeSessions)
	}
}

// persistSession persists a session for later restoration
func (s *SessionService) persistSession(ctx context.Context, sess *session.Session) error {
	// In a real implementation, this would save to storage
	// For now, we'll rely on the session manager's built-in persistence
	return nil
}

// GetMetrics returns service metrics
func (s *SessionService) GetMetrics() SessionServiceMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return SessionServiceMetrics{
		ActiveSessions:   s.activeSessions,
		TotalSessions:    s.totalSessions,
		SessionsRestored: s.sessionsRestored,
		Running:          s.started,
	}
}

// SessionServiceMetrics contains service metrics
type SessionServiceMetrics struct {
	ActiveSessions   int
	TotalSessions    uint64
	SessionsRestored uint64
	Running          bool
}
