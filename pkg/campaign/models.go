// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package campaign

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// CampaignStatus represents the state of a campaign
type CampaignStatus string

const (
	CampaignStatusDream     CampaignStatus = "dream"     // Initial idea/concept stage
	CampaignStatusPlanning  CampaignStatus = "planning"  // Commissions being defined and iterated
	CampaignStatusReady     CampaignStatus = "ready"     // Ready to execute
	CampaignStatusActive    CampaignStatus = "active"    // Currently being executed
	CampaignStatusPaused    CampaignStatus = "paused"    // Temporarily halted
	CampaignStatusCompleted CampaignStatus = "completed" // Successfully finished
	CampaignStatusCancelled CampaignStatus = "cancelled" // Terminated before completion
)

// Campaign represents a strategic goal with multiple commissions
type Campaign struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Status      CampaignStatus         `json:"status"`
	Commissions []string               `json:"commissions"` // Commission IDs
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`

	// Progress tracking
	TotalCommissions     int     `json:"total_commissions"`
	CompletedCommissions int     `json:"completed_commissions"`
	Progress             float64 `json:"progress"` // 0.0 to 1.0
}

// CampaignEvent represents state changes in a campaign
type CampaignEvent struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	CampaignID string                 `json:"campaign_id"`
	Campaign   *Campaign              `json:"campaign,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// CampaignProgress represents detailed progress information
type CampaignProgress struct {
	CampaignID           string    `json:"campaign_id"`
	TotalCommissions     int       `json:"total_commissions"`
	CompletedCommissions int       `json:"completed_commissions"`
	ActiveCommissions    int       `json:"active_commissions"`
	PendingCommissions   int       `json:"pending_commissions"`
	Progress             float64   `json:"progress"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// Event types for campaign state changes
const (
	EventCampaignCreated         = "campaign.created"
	EventCampaignPlanningStarted = "campaign.planning.started"
	EventCampaignMarkedReady     = "campaign.marked.ready"
	EventCampaignStarted         = "campaign.started"
	EventCampaignPaused          = "campaign.paused"
	EventCampaignResumed         = "campaign.resumed"
	EventCampaignCompleted       = "campaign.completed"
	EventCampaignCancelled       = "campaign.cancelled"
	EventCampaignProgress        = "campaign.progress"
	EventCommissionAdded         = "campaign.commission.added"
	EventCommissionRemoved       = "campaign.commission.removed"
)

// IsTerminal returns true if the campaign is in a terminal state
func (s CampaignStatus) IsTerminal() bool {
	return s == CampaignStatusCompleted || s == CampaignStatusCancelled
}

// String returns the string representation of the status
func (s CampaignStatus) String() string {
	return string(s)
}

// IsValid checks if the status is a valid campaign status
func (s CampaignStatus) IsValid() bool {
	switch s {
	case CampaignStatusDream, CampaignStatusPlanning, CampaignStatusReady,
		CampaignStatusActive, CampaignStatusPaused, CampaignStatusCompleted,
		CampaignStatusCancelled:
		return true
	default:
		return false
	}
}

// NewCampaign creates a new campaign with default values
func NewCampaign(name, description string) *Campaign {
	now := time.Now()
	return &Campaign{
		ID:          generateID(), // Will implement this helper
		Name:        name,
		Description: description,
		Status:      CampaignStatusDream, // Start in dream/idea stage
		Commissions: []string{},
		Tags:        []string{},
		Metadata:    make(map[string]interface{}),
		CreatedAt:   now,
		UpdatedAt:   now,
		Progress:    0.0,
	}
}

// generateID creates a unique campaign ID
func generateID() string {
	// Simple implementation for now, can be enhanced later
	return "campaign-" + time.Now().Format("20060102-150405-") + randomString(6)
}

// randomString generates a random string of given length
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
