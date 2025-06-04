package orchestrator

import (
	"fmt"
	"sync"
)

// OrchestratorRegistry manages orchestrator components for task planning and assignment
type OrchestratorRegistry interface {
	// RegisterCommissionPlanner registers a commission task planner
	RegisterCommissionPlanner(name string, planner CommissionTaskPlanner) error
	
	// GetCommissionPlanner retrieves a commission planner by name
	GetCommissionPlanner(name string) (CommissionTaskPlanner, error)
	
	// GetDefaultCommissionPlanner returns the default commission planner
	GetDefaultCommissionPlanner() (CommissionTaskPlanner, error)
	
	// SetDefaultCommissionPlanner sets the default commission planner
	SetDefaultCommissionPlanner(name string) error
	
	// RegisterEventBus registers an event bus for orchestrator events
	RegisterEventBus(name string, eventBus EventBus) error
	
	// GetEventBus retrieves an event bus by name
	GetEventBus(name string) (EventBus, error)
	
	// GetDefaultEventBus returns the default event bus
	GetDefaultEventBus() (EventBus, error)
	
	// ListCommissionPlanners returns all registered commission planner names
	ListCommissionPlanners() []string
	
	// HasCommissionPlanner checks if a commission planner is registered
	HasCommissionPlanner(name string) bool
}

// DefaultOrchestratorRegistry implements OrchestratorRegistry
type DefaultOrchestratorRegistry struct {
	commissionPlanners    map[string]CommissionTaskPlanner
	eventBuses           map[string]EventBus
	defaultPlanner       string
	defaultEventBus      string
	mu                   sync.RWMutex
}

// NewOrchestratorRegistry creates a new orchestrator registry
func NewOrchestratorRegistry() OrchestratorRegistry {
	return &DefaultOrchestratorRegistry{
		commissionPlanners: make(map[string]CommissionTaskPlanner),
		eventBuses:        make(map[string]EventBus),
	}
}

// RegisterCommissionPlanner registers a commission task planner
func (r *DefaultOrchestratorRegistry) RegisterCommissionPlanner(name string, planner CommissionTaskPlanner) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.commissionPlanners[name]; exists {
		return fmt.Errorf("commission planner %s already exists", name)
	}

	r.commissionPlanners[name] = planner
	
	// Set as default if it's the first one
	if r.defaultPlanner == "" {
		r.defaultPlanner = name
	}
	
	return nil
}

// GetCommissionPlanner retrieves a commission planner by name
func (r *DefaultOrchestratorRegistry) GetCommissionPlanner(name string) (CommissionTaskPlanner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	planner, exists := r.commissionPlanners[name]
	if !exists {
		return nil, fmt.Errorf("commission planner %s not found", name)
	}

	return planner, nil
}

// GetDefaultCommissionPlanner returns the default commission planner
func (r *DefaultOrchestratorRegistry) GetDefaultCommissionPlanner() (CommissionTaskPlanner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.defaultPlanner == "" {
		return nil, fmt.Errorf("no default commission planner configured")
	}

	return r.GetCommissionPlanner(r.defaultPlanner)
}

// SetDefaultCommissionPlanner sets the default commission planner
func (r *DefaultOrchestratorRegistry) SetDefaultCommissionPlanner(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.commissionPlanners[name]; !exists {
		return fmt.Errorf("commission planner %s not found", name)
	}

	r.defaultPlanner = name
	return nil
}

// RegisterEventBus registers an event bus for orchestrator events
func (r *DefaultOrchestratorRegistry) RegisterEventBus(name string, eventBus EventBus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.eventBuses[name]; exists {
		return fmt.Errorf("event bus %s already exists", name)
	}

	r.eventBuses[name] = eventBus
	
	// Set as default if it's the first one
	if r.defaultEventBus == "" {
		r.defaultEventBus = name
	}
	
	return nil
}

// GetEventBus retrieves an event bus by name
func (r *DefaultOrchestratorRegistry) GetEventBus(name string) (EventBus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	eventBus, exists := r.eventBuses[name]
	if !exists {
		return nil, fmt.Errorf("event bus %s not found", name)
	}

	return eventBus, nil
}

// GetDefaultEventBus returns the default event bus
func (r *DefaultOrchestratorRegistry) GetDefaultEventBus() (EventBus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.defaultEventBus == "" {
		return nil, fmt.Errorf("no default event bus configured")
	}

	return r.GetEventBus(r.defaultEventBus)
}

// ListCommissionPlanners returns all registered commission planner names
func (r *DefaultOrchestratorRegistry) ListCommissionPlanners() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.commissionPlanners))
	for name := range r.commissionPlanners {
		names = append(names, name)
	}

	return names
}

// HasCommissionPlanner checks if a commission planner is registered
func (r *DefaultOrchestratorRegistry) HasCommissionPlanner(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.commissionPlanners[name]
	return exists
}