package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Counter is a thread-safe counter metric.
type Counter struct {
	value atomic.Int64
}

func (c *Counter) Inc() {
	c.value.Add(1)
}

func (c *Counter) Add(delta int64) {
	c.value.Add(delta)
}

func (c *Counter) Get() int64 {
	return c.value.Load()
}

// Gauge is a thread-safe gauge metric.
type Gauge struct {
	value atomic.Int64
}

func (g *Gauge) Set(value int64) {
	g.value.Store(value)
}

func (g *Gauge) Inc() {
	g.value.Add(1)
}

func (g *Gauge) Dec() {
	g.value.Add(-1)
}

func (g *Gauge) Add(delta int64) {
	g.value.Add(delta)
}

func (g *Gauge) Get() int64 {
	return g.value.Load()
}

// Histogram is a simple histogram metric using buckets.
type Histogram struct {
	buckets []float64
	counts  []atomic.Int64
	sum     atomic.Int64
	count   atomic.Int64
	mu      sync.RWMutex
}

func NewHistogram(buckets []float64) *Histogram {
	if len(buckets) == 0 {
		buckets = []float64{10, 50, 100, 200, 500, 1000, 2000, 5000, 10000}
	}
	counts := make([]atomic.Int64, len(buckets)+1)
	return &Histogram{
		buckets: buckets,
		counts:  counts,
	}
}

func (h *Histogram) Observe(value float64) {
	h.sum.Add(int64(value * 1000)) // Store in microseconds
	h.count.Add(1)

	for i, bucket := range h.buckets {
		if value <= bucket {
			h.counts[i].Add(1)
			return
		}
	}
	// Value exceeds all buckets - put in last bucket (+1)
	h.counts[len(h.buckets)].Add(1)
}

func (h *Histogram) Count() int64 {
	return h.count.Load()
}

func (h *Histogram) Sum() float64 {
	return float64(h.sum.Load()) / 1000.0
}

// MetricsRegistry holds all application metrics.
type MetricsRegistry struct {
	// Chat metrics
	ChatRequestsTotal      *Counter
	ChatErrorsTotal        *Counter
	ChatLatency            *Histogram

	// Workflow metrics
	WorkflowTasksTotal     *Counter
	WorkflowCompletedTotal *Counter
	WorkflowFailedTotal    *Counter

	// Rule engine metrics
	RuleEvaluationsTotal   *Counter
	RuleFailuresTotal      *Counter
	RuleReviewsTotal       *Counter
	RuleInsufficientTotal  *Counter

	// Document metrics
	DocumentsGeneratedTotal     *Counter
	DocumentGenerationErrors    *Counter
	DocumentGenerationLatency   *Histogram

	// Document Engine metrics
	DocEngineRequestsTotal    *Counter
	DocEngineSuccessTotal     *Counter
	DocEngineErrorsTotal      *Counter
	DocEngineTimeoutsTotal    *Counter
	DocEngineFallbacksTotal   *Counter
	DocEngineLatency          *Histogram

	// RAG metrics
	RagSearchesTotal    *Counter
	RagZeroResultTotal  *Counter
	RagLatency          *Histogram

	// Rule engine metrics
	RuleEngineEvaluationsTotal *Counter
	RuleEnginePassTotal        *Counter
	RuleEngineFailTotal        *Counter
	RuleEngineReviewTotal      *Counter
	RuleEngineInsufficientTotal *Counter
	RuleEngineLatency          *Histogram

	mu sync.RWMutex
}

// NewMetricsRegistry creates a new metrics registry with default buckets.
func NewMetricsRegistry() *MetricsRegistry {
	return &MetricsRegistry{
		ChatRequestsTotal:        &Counter{},
		ChatErrorsTotal:          &Counter{},
		ChatLatency:              NewHistogram(nil),

		WorkflowTasksTotal:     &Counter{},
		WorkflowCompletedTotal: &Counter{},
		WorkflowFailedTotal:    &Counter{},

		RuleEvaluationsTotal:   &Counter{},
		RuleFailuresTotal:      &Counter{},
		RuleReviewsTotal:       &Counter{},
		RuleInsufficientTotal:  &Counter{},

		DocumentsGeneratedTotal:  &Counter{},
		DocumentGenerationErrors: &Counter{},
		DocumentGenerationLatency: NewHistogram(nil),

		DocEngineRequestsTotal:  &Counter{},
		DocEngineSuccessTotal:   &Counter{},
		DocEngineErrorsTotal:    &Counter{},
		DocEngineTimeoutsTotal:  &Counter{},
		DocEngineFallbacksTotal: &Counter{},
		DocEngineLatency:        NewHistogram(nil),

		RagSearchesTotal:   &Counter{},
		RagZeroResultTotal: &Counter{},
		RagLatency:         NewHistogram(nil),

		RuleEngineEvaluationsTotal:  &Counter{},
		RuleEnginePassTotal:         &Counter{},
		RuleEngineFailTotal:         &Counter{},
		RuleEngineReviewTotal:       &Counter{},
		RuleEngineInsufficientTotal: &Counter{},
		RuleEngineLatency:           NewHistogram(nil),
	}
}

// RecordChatRequest increments chat request counter and records latency.
func (m *MetricsRegistry) RecordChatRequest(latency time.Duration) {
	m.ChatRequestsTotal.Inc()
	m.ChatLatency.Observe(float64(latency.Microseconds()))
}

// RecordChatError increments chat error counter.
func (m *MetricsRegistry) RecordChatError() {
	m.ChatErrorsTotal.Inc()
}

// RecordWorkflowTaskCreated increments workflow task counter.
func (m *MetricsRegistry) RecordWorkflowTaskCreated() {
	m.WorkflowTasksTotal.Inc()
}

// RecordWorkflowCompleted increments workflow completed counter.
func (m *MetricsRegistry) RecordWorkflowCompleted() {
	m.WorkflowCompletedTotal.Inc()
}

// RecordWorkflowFailed increments workflow failed counter.
func (m *MetricsRegistry) RecordWorkflowFailed() {
	m.WorkflowFailedTotal.Inc()
}

// RecordRuleEvaluation records a rule evaluation.
func (m *MetricsRegistry) RecordRuleEvaluation(status string) {
	m.RuleEvaluationsTotal.Inc()
	switch status {
	case "FAIL":
		m.RuleFailuresTotal.Inc()
	case "NEEDS_REVIEW":
		m.RuleReviewsTotal.Inc()
	case "INSUFFICIENT_EVIDENCE":
		m.RuleInsufficientTotal.Inc()
	}
}

// RecordDocumentGenerated increments document generated counter and records latency.
func (m *MetricsRegistry) RecordDocumentGenerated(latency time.Duration) {
	m.DocumentsGeneratedTotal.Inc()
	m.DocumentGenerationLatency.Observe(float64(latency.Microseconds()))
}

// RecordDocumentError increments document generation error counter.
func (m *MetricsRegistry) RecordDocumentError() {
	m.DocumentGenerationErrors.Inc()
}

// RecordDocEngineRequest records a document engine request.
func (m *MetricsRegistry) RecordDocEngineRequest(success bool, latency time.Duration) {
	m.DocEngineRequestsTotal.Inc()
	if success {
		m.DocEngineSuccessTotal.Inc()
	} else {
		m.DocEngineErrorsTotal.Inc()
	}
	m.DocEngineLatency.Observe(float64(latency.Microseconds()))
}

// RecordDocEngineTimeout increments document engine timeout counter.
func (m *MetricsRegistry) RecordDocEngineTimeout() {
	m.DocEngineTimeoutsTotal.Inc()
}

// RecordDocEngineFallback increments document engine fallback counter.
func (m *MetricsRegistry) RecordDocEngineFallback() {
	m.DocEngineFallbacksTotal.Inc()
}

// RecordRagSearch records a RAG search.
func (m *MetricsRegistry) RecordRagSearch(zeroResult bool, latency time.Duration) {
	m.RagSearchesTotal.Inc()
	if zeroResult {
		m.RagZeroResultTotal.Inc()
	}
	m.RagLatency.Observe(float64(latency.Microseconds()))
}

// RecordRuleEngineEvaluation records a rule engine evaluation.
func (m *MetricsRegistry) RecordRuleEngineEvaluation(status string, latency time.Duration) {
	m.RuleEngineEvaluationsTotal.Inc()
	m.RuleEngineLatency.Observe(float64(latency.Microseconds()))
	switch status {
	case "PASS":
		m.RuleEnginePassTotal.Inc()
	case "FAIL":
		m.RuleEngineFailTotal.Inc()
	case "NEEDS_REVIEW":
		m.RuleEngineReviewTotal.Inc()
	case "INSUFFICIENT_EVIDENCE":
		m.RuleEngineInsufficientTotal.Inc()
	}
}

// Snapshot returns a snapshot of all metrics.
func (m *MetricsRegistry) Snapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"chat_requests_total":         m.ChatRequestsTotal.Get(),
		"chat_errors_total":           m.ChatErrorsTotal.Get(),
		"chat_latency_avg_us":         m.ChatLatency.Sum() / float64(max(1, m.ChatLatency.Count())),
		"workflow_tasks_total":        m.WorkflowTasksTotal.Get(),
		"workflow_completed_total":    m.WorkflowCompletedTotal.Get(),
		"workflow_failed_total":       m.WorkflowFailedTotal.Get(),
		"rule_evaluations_total":      m.RuleEvaluationsTotal.Get(),
		"rule_failures_total":         m.RuleFailuresTotal.Get(),
		"rule_reviews_total":          m.RuleReviewsTotal.Get(),
		"rule_insufficient_total":     m.RuleInsufficientTotal.Get(),
		"documents_generated_total":   m.DocumentsGeneratedTotal.Get(),
		"document_generation_errors":  m.DocumentGenerationErrors.Get(),
		"doc_engine_requests_total":   m.DocEngineRequestsTotal.Get(),
		"doc_engine_success_total":    m.DocEngineSuccessTotal.Get(),
		"doc_engine_errors_total":     m.DocEngineErrorsTotal.Get(),
		"doc_engine_timeouts_total":   m.DocEngineTimeoutsTotal.Get(),
		"doc_engine_fallbacks_total":  m.DocEngineFallbacksTotal.Get(),
		"rag_searches_total":          m.RagSearchesTotal.Get(),
		"rag_zero_result_total":       m.RagZeroResultTotal.Get(),
		"rule_engine_evaluations":     m.RuleEngineEvaluationsTotal.Get(),
		"rule_engine_pass_total":      m.RuleEnginePassTotal.Get(),
		"rule_engine_fail_total":      m.RuleEngineFailTotal.Get(),
		"rule_engine_review_total":    m.RuleEngineReviewTotal.Get(),
		"rule_engine_insufficient":    m.RuleEngineInsufficientTotal.Get(),
	}
}

// Global metrics instance
var GlobalMetrics = NewMetricsRegistry()