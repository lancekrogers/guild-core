// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"sync"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// ComponentRegistry manages registration and retrieval of manager components
type ComponentRegistry interface {
	// RegisterParser registers a response parser
	RegisterParser(name string, parser ResponseParser) error

	// GetParser retrieves a parser by name
	GetParser(name string) (ResponseParser, error)

	// RegisterValidator registers a structure validator
	RegisterValidator(name string, validator StructureValidator) error

	// GetValidator retrieves a validator by name
	GetValidator(name string) (StructureValidator, error)

	// RegisterRefiner registers a commission refiner
	RegisterRefiner(name string, refiner CommissionRefiner) error

	// GetRefiner retrieves a refiner by name
	GetRefiner(name string) (CommissionRefiner, error)

	// RegisterArtisanClient registers an artisan client
	RegisterArtisanClient(name string, client ArtisanClient) error

	// GetArtisanClient retrieves an artisan client by name
	GetArtisanClient(name string) (ArtisanClient, error)
}

// DefaultComponentRegistry is the default implementation of ComponentRegistry
type DefaultComponentRegistry struct {
	parsers        map[string]ResponseParser
	validators     map[string]StructureValidator
	refiners       map[string]CommissionRefiner
	artisanClients map[string]ArtisanClient
	mu             sync.RWMutex
}

// NewComponentRegistry creates a new component registry
func NewComponentRegistry() ComponentRegistry {
	return &DefaultComponentRegistry{
		parsers:        make(map[string]ResponseParser),
		validators:     make(map[string]StructureValidator),
		refiners:       make(map[string]CommissionRefiner),
		artisanClients: make(map[string]ArtisanClient),
	}
}

// RegisterParser registers a response parser
func (r *DefaultComponentRegistry) RegisterParser(name string, parser ResponseParser) error {
	if name == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "parser name cannot be empty", nil).
			WithComponent("manager-registry").
			WithOperation("RegisterParser")
	}
	if parser == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "parser cannot be nil", nil).
			WithComponent("manager-registry").
			WithOperation("RegisterParser")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.parsers[name]; exists {
		return gerror.Newf(gerror.ErrCodeAlreadyExists, "parser '%s' already registered", name).
			WithComponent("manager-registry").
			WithOperation("RegisterParser").
			WithDetails("parser_name", name)
	}

	r.parsers[name] = parser
	return nil
}

// GetParser retrieves a parser by name
func (r *DefaultComponentRegistry) GetParser(name string) (ResponseParser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	parser, exists := r.parsers[name]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "parser '%s' not found", name).
			WithComponent("manager-registry").
			WithOperation("GetParser").
			WithDetails("parser_name", name)
	}

	return parser, nil
}

// RegisterValidator registers a structure validator
func (r *DefaultComponentRegistry) RegisterValidator(name string, validator StructureValidator) error {
	if name == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "validator name cannot be empty", nil).
			WithComponent("manager-registry").
			WithOperation("RegisterValidator")
	}
	if validator == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "validator cannot be nil", nil).
			WithComponent("manager-registry").
			WithOperation("RegisterValidator")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.validators[name]; exists {
		return gerror.Newf(gerror.ErrCodeAlreadyExists, "validator '%s' already registered", name).
			WithComponent("manager-registry").
			WithOperation("RegisterValidator").
			WithDetails("validator_name", name)
	}

	r.validators[name] = validator
	return nil
}

// GetValidator retrieves a validator by name
func (r *DefaultComponentRegistry) GetValidator(name string) (StructureValidator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	validator, exists := r.validators[name]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "validator '%s' not found", name).
			WithComponent("manager-registry").
			WithOperation("GetValidator").
			WithDetails("validator_name", name)
	}

	return validator, nil
}

// RegisterRefiner registers a commission refiner
func (r *DefaultComponentRegistry) RegisterRefiner(name string, refiner CommissionRefiner) error {
	if name == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "refiner name cannot be empty", nil).
			WithComponent("manager-registry").
			WithOperation("RegisterRefiner")
	}
	if refiner == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "refiner cannot be nil", nil).
			WithComponent("manager-registry").
			WithOperation("RegisterRefiner")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.refiners[name]; exists {
		return gerror.Newf(gerror.ErrCodeAlreadyExists, "refiner '%s' already registered", name).
			WithComponent("manager-registry").
			WithOperation("RegisterRefiner").
			WithDetails("refiner_name", name)
	}

	r.refiners[name] = refiner
	return nil
}

// GetRefiner retrieves a refiner by name
func (r *DefaultComponentRegistry) GetRefiner(name string) (CommissionRefiner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	refiner, exists := r.refiners[name]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "refiner '%s' not found", name).
			WithComponent("manager-registry").
			WithOperation("GetRefiner").
			WithDetails("refiner_name", name)
	}

	return refiner, nil
}

// RegisterArtisanClient registers an artisan client
func (r *DefaultComponentRegistry) RegisterArtisanClient(name string, client ArtisanClient) error {
	if name == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "client name cannot be empty", nil).
			WithComponent("manager-registry").
			WithOperation("RegisterArtisanClient")
	}
	if client == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "client cannot be nil", nil).
			WithComponent("manager-registry").
			WithOperation("RegisterArtisanClient")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.artisanClients[name]; exists {
		return gerror.Newf(gerror.ErrCodeAlreadyExists, "client '%s' already registered", name).
			WithComponent("manager-registry").
			WithOperation("RegisterArtisanClient").
			WithDetails("client_name", name)
	}

	r.artisanClients[name] = client
	return nil
}

// GetArtisanClient retrieves an artisan client by name
func (r *DefaultComponentRegistry) GetArtisanClient(name string) (ArtisanClient, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	client, exists := r.artisanClients[name]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "client '%s' not found", name).
			WithComponent("manager-registry").
			WithOperation("GetArtisanClient").
			WithDetails("client_name", name)
	}

	return client, nil
}
