package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"gateway/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/sony/gobreaker"
)

func CircuitBreakerMiddleware(cfg config.CircuitBreakerConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil || timeout <= 0 {
		timeout = 30 * time.Second
	}

	interval, err := time.ParseDuration(cfg.Interval)
	if err != nil || interval <= 0 {
		interval = 60 * time.Second
	}

	if cfg.MaxFailures == 0 {
		cfg.MaxFailures = 5
	}
	if cfg.MinRequests == 0 {
		cfg.MinRequests = 1
	}

	breakers := &httpCircuitBreakers{
		breakers: make(map[string]*gobreaker.CircuitBreaker),
		settings: gobreaker.Settings{
			MaxRequests: cfg.MinRequests,
			Interval:    interval,
			Timeout:     timeout,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= cfg.MaxFailures
			},
		},
	}

	return func(c *gin.Context) {
		cb := breakers.Get(c.FullPath(), c.Request.URL.Path)

		_, err := cb.Execute(func() (interface{}, error) {
			c.Next()
			if c.Writer.Status() >= http.StatusInternalServerError {
				return nil, fmt.Errorf("backend service error: %d", c.Writer.Status())
			}
			return nil, nil
		})

		if err == gobreaker.ErrOpenState {
			AuditReject(c, "circuit_breaker", "open_state", http.StatusServiceUnavailable)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"code":    http.StatusServiceUnavailable,
				"message": "Service temporarily unavailable, circuit breaker is open",
			})
			c.Abort()
		}
	}
}

type httpCircuitBreakers struct {
	mu       sync.RWMutex
	breakers map[string]*gobreaker.CircuitBreaker
	settings gobreaker.Settings
}

func (h *httpCircuitBreakers) Get(routePattern string, path string) *gobreaker.CircuitBreaker {
	key := routePattern
	if key == "" {
		key = path
	}

	h.mu.RLock()
	breaker, ok := h.breakers[key]
	h.mu.RUnlock()
	if ok {
		return breaker
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if breaker, ok = h.breakers[key]; ok {
		return breaker
	}

	settings := h.settings
	settings.Name = key
	breaker = gobreaker.NewCircuitBreaker(settings)
	h.breakers[key] = breaker
	return breaker
}

type PerServiceCircuitBreaker struct {
	mu       sync.Mutex
	breakers map[string]*gobreaker.CircuitBreaker
	cfg      config.CircuitBreakerConfig
}

func NewPerServiceCircuitBreaker(cfg config.CircuitBreakerConfig) *PerServiceCircuitBreaker {
	return &PerServiceCircuitBreaker{
		breakers: make(map[string]*gobreaker.CircuitBreaker),
		cfg:      cfg,
	}
}

func (pscb *PerServiceCircuitBreaker) GetBreaker(serviceName string) *gobreaker.CircuitBreaker {
	pscb.mu.Lock()
	defer pscb.mu.Unlock()

	if breaker, exists := pscb.breakers[serviceName]; exists {
		return breaker
	}

	timeout, _ := time.ParseDuration(pscb.cfg.Timeout)
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	interval, _ := time.ParseDuration(pscb.cfg.Interval)
	if interval <= 0 {
		interval = 60 * time.Second
	}

	maxFailures := pscb.cfg.MaxFailures
	if maxFailures == 0 {
		maxFailures = 5
	}
	minRequests := pscb.cfg.MinRequests
	if minRequests == 0 {
		minRequests = 1
	}

	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        serviceName,
		MaxRequests: minRequests,
		Interval:    interval,
		Timeout:     timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= maxFailures
		},
	})

	pscb.breakers[serviceName] = breaker
	return breaker
}
