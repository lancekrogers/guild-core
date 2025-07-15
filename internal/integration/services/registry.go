package services

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// serviceEntry holds a service and its metadata
type serviceEntry struct {
	service      Service
	info         ServiceInfo
	options      ServiceOptions
	healthCancel context.CancelFunc
	readyCancel  context.CancelFunc
}

// DefaultServiceRegistry implements ServiceRegistry
type DefaultServiceRegistry struct {
	services map[string]*serviceEntry
	mu       sync.RWMutex

	// Dependency graph
	dependencies map[string][]string // service -> dependencies
	dependents   map[string][]string // service -> dependents

	// Lifecycle hooks
	hooks []ServiceHook

	// Context for background operations
	ctx    context.Context
	cancel context.CancelFunc
	logger observability.Logger
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(ctx context.Context) *DefaultServiceRegistry {
	ctx, cancel := context.WithCancel(ctx)
	return &DefaultServiceRegistry{
		services:     make(map[string]*serviceEntry),
		dependencies: make(map[string][]string),
		dependents:   make(map[string][]string),
		hooks:        make([]ServiceHook, 0),
		ctx:          ctx,
		cancel:       cancel,
		logger:       observability.GetLogger(ctx),
	}
}

// Register adds a service to the registry
func (r *DefaultServiceRegistry) Register(service Service) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := service.Name()
	if name == "" {
		return gerror.New(gerror.ErrCodeValidation, "service name cannot be empty", nil).
			WithComponent("service_registry")
	}

	if _, exists := r.services[name]; exists {
		return gerror.New(gerror.ErrCodeAlreadyExists, "service already registered", nil).
			WithComponent("service_registry").
			WithDetails("service", name)
	}

	entry := &serviceEntry{
		service: service,
		info: ServiceInfo{
			Name:  name,
			State: StateUnknown,
		},
		options: DefaultServiceOptions(),
	}

	r.services[name] = entry
	r.dependencies[name] = []string{}
	r.dependents[name] = []string{}

	r.logger.InfoContext(r.ctx, "Service registered", "service", name)
	return nil
}

// Unregister removes a service from the registry
func (r *DefaultServiceRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.services[name]
	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "service not found", nil).
			WithComponent("service_registry").
			WithDetails("service", name)
	}

	// Stop health checks
	if entry.healthCancel != nil {
		entry.healthCancel()
	}
	if entry.readyCancel != nil {
		entry.readyCancel()
	}

	// Remove from maps
	delete(r.services, name)
	delete(r.dependencies, name)

	// Remove as dependent from other services
	for svc, deps := range r.dependents {
		filtered := make([]string, 0, len(deps))
		for _, dep := range deps {
			if dep != name {
				filtered = append(filtered, dep)
			}
		}
		r.dependents[svc] = filtered
	}
	delete(r.dependents, name)

	r.logger.InfoContext(r.ctx, "Service unregistered", "service", name)
	return nil
}

// Get retrieves a service by name
func (r *DefaultServiceRegistry) Get(name string) (Service, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.services[name]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "service not found", nil).
			WithComponent("service_registry").
			WithDetails("service", name)
	}

	return entry.service, nil
}

// List returns all registered services
func (r *DefaultServiceRegistry) List() []ServiceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]ServiceInfo, 0, len(r.services))
	for _, entry := range r.services {
		info := entry.info
		info.Dependencies = r.dependencies[info.Name]
		infos = append(infos, info)
	}

	// Sort by name for consistent output
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})

	return infos
}

// SetDependency declares that serviceA depends on serviceB
func (r *DefaultServiceRegistry) SetDependency(serviceA, serviceB string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate services exist
	if _, exists := r.services[serviceA]; !exists {
		return gerror.New(gerror.ErrCodeNotFound, "service not found", nil).
			WithComponent("service_registry").
			WithDetails("service", serviceA)
	}
	if _, exists := r.services[serviceB]; !exists {
		return gerror.New(gerror.ErrCodeNotFound, "service not found", nil).
			WithComponent("service_registry").
			WithDetails("service", serviceB)
	}

	// Check for circular dependency
	if r.wouldCreateCycle(serviceA, serviceB) {
		return gerror.New(gerror.ErrCodeValidation, "circular dependency detected", nil).
			WithComponent("service_registry").
			WithDetails("serviceA", serviceA).
			WithDetails("serviceB", serviceB)
	}

	// Add dependency
	deps := r.dependencies[serviceA]
	for _, dep := range deps {
		if dep == serviceB {
			return nil // Already exists
		}
	}
	r.dependencies[serviceA] = append(deps, serviceB)
	r.dependents[serviceB] = append(r.dependents[serviceB], serviceA)

	r.logger.InfoContext(r.ctx, "Service dependency added",
		"service", serviceA,
		"depends_on", serviceB)

	return nil
}

// Start starts all services in dependency order
func (r *DefaultServiceRegistry) Start(ctx context.Context) error {
	order, err := r.getStartOrder()
	if err != nil {
		return err
	}

	r.logger.InfoContext(ctx, "Starting services", "count", len(order), "order", order)

	for _, name := range order {
		if err := r.startService(ctx, name); err != nil {
			// Stop already started services on error
			r.stopStartedServices(ctx, order, name)
			return err
		}
	}

	r.logger.InfoContext(ctx, "All services started successfully")
	return nil
}

// Stop stops all services in reverse dependency order
func (r *DefaultServiceRegistry) Stop(ctx context.Context) error {
	order, err := r.getStartOrder()
	if err != nil {
		return err
	}

	// Reverse order for stopping
	for i := len(order)/2 - 1; i >= 0; i-- {
		opp := len(order) - 1 - i
		order[i], order[opp] = order[opp], order[i]
	}

	r.logger.InfoContext(ctx, "Stopping services", "count", len(order), "order", order)

	var errors []error
	for _, name := range order {
		if err := r.stopService(ctx, name); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return gerror.New(gerror.ErrCodeInternal, "failed to stop some services", nil).
			WithComponent("service_registry").
			WithDetails("errors", errors)
	}

	r.logger.InfoContext(ctx, "All services stopped successfully")
	return nil
}

// Health checks health of all services
func (r *DefaultServiceRegistry) Health(ctx context.Context) map[string]error {
	r.mu.RLock()
	services := make(map[string]Service)
	for name, entry := range r.services {
		if entry.info.State == StateRunning {
			services[name] = entry.service
		}
	}
	r.mu.RUnlock()

	results := make(map[string]error)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, service := range services {
		wg.Add(1)
		go func(n string, s Service) {
			defer wg.Done()

			err := s.Health(ctx)
			mu.Lock()
			results[n] = err
			mu.Unlock()
		}(name, service)
	}

	wg.Wait()
	return results
}

// AddHook adds a lifecycle hook
func (r *DefaultServiceRegistry) AddHook(hook ServiceHook) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks = append(r.hooks, hook)
}

// startService starts a single service
func (r *DefaultServiceRegistry) startService(ctx context.Context, name string) error {
	r.mu.Lock()
	entry, exists := r.services[name]
	if !exists {
		r.mu.Unlock()
		return gerror.New(gerror.ErrCodeNotFound, "service not found", nil).
			WithComponent("service_registry").
			WithDetails("service", name)
	}

	// Update state
	entry.info.State = StateStarting
	entry.info.StartedAt = time.Now()
	r.mu.Unlock()

	// Call pre-start hooks
	for _, hook := range r.hooks {
		if err := hook.OnStart(ctx, entry.service); err != nil {
			r.mu.Lock()
			entry.info.State = StateError
			entry.info.Error = err
			r.mu.Unlock()
			return err
		}
	}

	// Start with timeout
	startCtx, cancel := context.WithTimeout(ctx, entry.options.StartTimeout)
	defer cancel()

	r.logger.InfoContext(ctx, "Starting service", "service", name)

	if err := entry.service.Start(startCtx); err != nil {
		r.mu.Lock()
		entry.info.State = StateError
		entry.info.Error = gerror.Wrap(err, gerror.ErrCodeInternal, "service start failed").
			WithComponent("service_registry").
			WithDetails("service", name)
		r.mu.Unlock()

		// Call error hooks
		for _, hook := range r.hooks {
			hook.OnError(ctx, entry.service, err)
		}

		return entry.info.Error
	}

	// Update state
	r.mu.Lock()
	entry.info.State = StateRunning
	entry.info.Error = nil
	r.mu.Unlock()

	// Start health checks
	r.startHealthChecks(entry)

	// Call post-start hooks
	for _, hook := range r.hooks {
		hook.OnStarted(ctx, entry.service)
	}

	r.logger.InfoContext(ctx, "Service started successfully",
		"service", name,
		"duration", time.Since(entry.info.StartedAt))

	return nil
}

// stopService stops a single service
func (r *DefaultServiceRegistry) stopService(ctx context.Context, name string) error {
	r.mu.Lock()
	entry, exists := r.services[name]
	if !exists {
		r.mu.Unlock()
		return gerror.New(gerror.ErrCodeNotFound, "service not found", nil).
			WithComponent("service_registry").
			WithDetails("service", name)
	}

	// Stop health checks
	if entry.healthCancel != nil {
		entry.healthCancel()
	}
	if entry.readyCancel != nil {
		entry.readyCancel()
	}

	// Update state
	entry.info.State = StateStopping
	entry.info.StoppedAt = time.Now()
	r.mu.Unlock()

	// Call pre-stop hooks
	for _, hook := range r.hooks {
		if err := hook.OnStop(ctx, entry.service); err != nil {
			return err
		}
	}

	// Stop with timeout
	stopCtx, cancel := context.WithTimeout(ctx, entry.options.StopTimeout)
	defer cancel()

	r.logger.InfoContext(ctx, "Stopping service", "service", name)

	if err := entry.service.Stop(stopCtx); err != nil {
		r.mu.Lock()
		entry.info.State = StateError
		entry.info.Error = gerror.Wrap(err, gerror.ErrCodeInternal, "service stop failed").
			WithComponent("service_registry").
			WithDetails("service", name)
		r.mu.Unlock()

		// Call error hooks
		for _, hook := range r.hooks {
			hook.OnError(ctx, entry.service, err)
		}

		return entry.info.Error
	}

	// Update state
	r.mu.Lock()
	entry.info.State = StateStopped
	entry.info.Error = nil
	r.mu.Unlock()

	// Call post-stop hooks
	for _, hook := range r.hooks {
		hook.OnStopped(ctx, entry.service)
	}

	r.logger.InfoContext(ctx, "Service stopped successfully",
		"service", name,
		"uptime", entry.info.StoppedAt.Sub(entry.info.StartedAt))

	return nil
}

// startHealthChecks starts background health and readiness checks
func (r *DefaultServiceRegistry) startHealthChecks(entry *serviceEntry) {
	// Health check
	healthCtx, healthCancel := context.WithCancel(r.ctx)
	entry.healthCancel = healthCancel

	go func() {
		ticker := time.NewTicker(entry.options.HealthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-healthCtx.Done():
				return
			case <-ticker.C:
				err := entry.service.Health(healthCtx)

				r.mu.Lock()
				entry.info.LastHealthAt = time.Now()
				entry.info.Healthy = err == nil
				if err != nil {
					entry.info.Error = err
				}
				r.mu.Unlock()

				// Call health check hooks
				for _, hook := range r.hooks {
					hook.OnHealthCheck(healthCtx, entry.service, err == nil, err)
				}
			}
		}
	}()

	// Readiness check
	readyCtx, readyCancel := context.WithCancel(r.ctx)
	entry.readyCancel = readyCancel

	go func() {
		ticker := time.NewTicker(entry.options.ReadinessCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-readyCtx.Done():
				return
			case <-ticker.C:
				err := entry.service.Ready(readyCtx)

				r.mu.Lock()
				entry.info.Ready = err == nil
				r.mu.Unlock()
			}
		}
	}()
}

// getStartOrder returns services in dependency order
func (r *DefaultServiceRegistry) getStartOrder() ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Topological sort using Kahn's algorithm
	inDegree := make(map[string]int)
	for name := range r.services {
		inDegree[name] = len(r.dependencies[name])
	}

	queue := make([]string, 0)
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	var order []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		order = append(order, current)

		for _, dependent := range r.dependents[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(order) != len(r.services) {
		return nil, gerror.New(gerror.ErrCodeValidation, "circular dependency detected", nil).
			WithComponent("service_registry")
	}

	return order, nil
}

// wouldCreateCycle checks if adding a dependency would create a cycle
func (r *DefaultServiceRegistry) wouldCreateCycle(from, to string) bool {
	// DFS to check if we can reach 'from' starting from 'to'
	visited := make(map[string]bool)
	var dfs func(string) bool

	dfs = func(current string) bool {
		if current == from {
			return true
		}
		if visited[current] {
			return false
		}
		visited[current] = true

		for _, dep := range r.dependencies[current] {
			if dfs(dep) {
				return true
			}
		}
		return false
	}

	return dfs(to)
}

// stopStartedServices stops services that were started before an error
func (r *DefaultServiceRegistry) stopStartedServices(ctx context.Context, order []string, failedService string) {
	// Find services to stop
	var toStop []string
	for _, name := range order {
		if name == failedService {
			break
		}
		toStop = append(toStop, name)
	}

	// Reverse order for stopping
	for i := len(toStop)/2 - 1; i >= 0; i-- {
		opp := len(toStop) - 1 - i
		toStop[i], toStop[opp] = toStop[opp], toStop[i]
	}

	// Stop services
	for _, name := range toStop {
		if err := r.stopService(ctx, name); err != nil {
			r.logger.ErrorContext(ctx, "Failed to stop service during rollback",
				"service", name,
				"error", err)
		}
	}
}

// Close shuts down the registry
func (r *DefaultServiceRegistry) Close() error {
	r.cancel()
	return nil
}
