package telemetry

import (
	"log"
	"sync"
	"time"

	"nuclear-ao3/shared/models"
)

// InMemoryTelemetryCollector provides a simple in-memory telemetry collector
type InMemoryTelemetryCollector struct {
	mu              sync.RWMutex
	deliveryMetrics map[models.DeliveryChannel]*ChannelStats
	counters        map[string]int64
	gauges          map[string]float64
	latencies       map[models.DeliveryChannel][]time.Duration
	errors          map[models.DeliveryChannel][]ErrorRecord
	attempts        []AttemptRecord
	maxHistorySize  int
}

// ChannelStats holds statistics for a delivery channel
type ChannelStats struct {
	TotalAttempts       int64         `json:"total_attempts"`
	SuccessfulSent      int64         `json:"successful_sent"`
	SuccessfulDelivered int64         `json:"successful_delivered"`
	Failed              int64         `json:"failed"`
	TotalLatency        time.Duration `json:"total_latency"`
	MinLatency          time.Duration `json:"min_latency"`
	MaxLatency          time.Duration `json:"max_latency"`
	LastActivity        time.Time     `json:"last_activity"`
}

// ErrorRecord records error information with timestamp
type ErrorRecord struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	Error     string                 `json:"error"`
	Channel   models.DeliveryChannel `json:"channel"`
}

// AttemptRecord records delivery attempt information
type AttemptRecord struct {
	Timestamp time.Time               `json:"timestamp"`
	Attempt   *models.DeliveryAttempt `json:"attempt"`
}

// NewInMemoryTelemetryCollector creates a new in-memory telemetry collector
func NewInMemoryTelemetryCollector() *InMemoryTelemetryCollector {
	return &InMemoryTelemetryCollector{
		deliveryMetrics: make(map[models.DeliveryChannel]*ChannelStats),
		counters:        make(map[string]int64),
		gauges:          make(map[string]float64),
		latencies:       make(map[models.DeliveryChannel][]time.Duration),
		errors:          make(map[models.DeliveryChannel][]ErrorRecord),
		attempts:        make([]AttemptRecord, 0),
		maxHistorySize:  10000, // Keep last 10k records
	}
}

// RecordDeliveryAttempt records a delivery attempt
func (c *InMemoryTelemetryCollector) RecordDeliveryAttempt(attempt *models.DeliveryAttempt) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Initialize channel stats if not exists
	if _, exists := c.deliveryMetrics[attempt.Channel]; !exists {
		c.deliveryMetrics[attempt.Channel] = &ChannelStats{
			MinLatency: time.Hour, // Initialize to a high value
		}
	}

	stats := c.deliveryMetrics[attempt.Channel]
	stats.TotalAttempts++
	stats.LastActivity = time.Now()

	// Update status-specific counters
	switch attempt.Status {
	case models.DeliveryStatusSent:
		stats.SuccessfulSent++
	case models.DeliveryStatusDelivered:
		stats.SuccessfulDelivered++
	case models.DeliveryStatusFailed:
		stats.Failed++
	}

	// Record attempt history
	c.attempts = append(c.attempts, AttemptRecord{
		Timestamp: time.Now(),
		Attempt:   attempt,
	})

	// Trim history if too large
	if len(c.attempts) > c.maxHistorySize {
		c.attempts = c.attempts[len(c.attempts)-c.maxHistorySize:]
	}

	// Log attempt for debugging
	log.Printf("Telemetry: Recorded delivery attempt - Channel: %s, Status: %s, ID: %s",
		attempt.Channel, attempt.Status, attempt.ID)
}

// RecordLatency records delivery latency for a channel
func (c *InMemoryTelemetryCollector) RecordLatency(channel models.DeliveryChannel, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Initialize channel stats if not exists
	if _, exists := c.deliveryMetrics[channel]; !exists {
		c.deliveryMetrics[channel] = &ChannelStats{
			MinLatency: time.Hour,
		}
	}

	stats := c.deliveryMetrics[channel]
	stats.TotalLatency += duration

	// Update min/max latency
	if duration < stats.MinLatency {
		stats.MinLatency = duration
	}
	if duration > stats.MaxLatency {
		stats.MaxLatency = duration
	}

	// Store latency history
	if _, exists := c.latencies[channel]; !exists {
		c.latencies[channel] = make([]time.Duration, 0)
	}
	c.latencies[channel] = append(c.latencies[channel], duration)

	// Trim latency history
	if len(c.latencies[channel]) > 1000 {
		c.latencies[channel] = c.latencies[channel][len(c.latencies[channel])-1000:]
	}

	log.Printf("Telemetry: Recorded latency - Channel: %s, Duration: %v", channel, duration)
}

// RecordError records an error for a channel
func (c *InMemoryTelemetryCollector) RecordError(channel models.DeliveryChannel, errorType string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	errorRecord := ErrorRecord{
		Timestamp: time.Now(),
		Type:      errorType,
		Error:     err.Error(),
		Channel:   channel,
	}

	if _, exists := c.errors[channel]; !exists {
		c.errors[channel] = make([]ErrorRecord, 0)
	}
	c.errors[channel] = append(c.errors[channel], errorRecord)

	// Trim error history
	if len(c.errors[channel]) > 1000 {
		c.errors[channel] = c.errors[channel][len(c.errors[channel])-1000:]
	}

	log.Printf("Telemetry: Recorded error - Channel: %s, Type: %s, Error: %v", channel, errorType, err)
}

// IncrementCounter increments a named counter
func (c *InMemoryTelemetryCollector) IncrementCounter(name string, tags map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// For simplicity, ignore tags in this implementation
	// In a real implementation, tags would be used to create unique counter keys
	c.counters[name]++

	log.Printf("Telemetry: Incremented counter - Name: %s, Value: %d, Tags: %v", name, c.counters[name], tags)
}

// RecordGauge records a gauge value
func (c *InMemoryTelemetryCollector) RecordGauge(name string, value float64, tags map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// For simplicity, ignore tags in this implementation
	c.gauges[name] = value

	log.Printf("Telemetry: Recorded gauge - Name: %s, Value: %f, Tags: %v", name, value, tags)
}

// GetMetrics returns collected metrics for a time period
func (c *InMemoryTelemetryCollector) GetMetrics(start, end time.Time) (*models.MessageMetrics, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Calculate aggregate metrics
	var totalSent, totalDelivered, totalFailed int64
	var totalLatency time.Duration
	var latencyCount int64

	byChannel := make(map[models.DeliveryChannel]models.ChannelMetrics)

	for channel, stats := range c.deliveryMetrics {
		channelMetrics := models.ChannelMetrics{
			Sent:      stats.SuccessfulSent,
			Delivered: stats.SuccessfulDelivered,
			Failed:    stats.Failed,
		}

		// Calculate delivery rate
		if stats.TotalAttempts > 0 {
			channelMetrics.DeliveryRate = float64(stats.SuccessfulDelivered) / float64(stats.TotalAttempts)
		}

		// Calculate average latency
		if stats.TotalAttempts > 0 {
			channelMetrics.AvgLatency = stats.TotalLatency.Milliseconds() / stats.TotalAttempts
		}

		byChannel[channel] = channelMetrics

		// Aggregate totals
		totalSent += stats.SuccessfulSent
		totalDelivered += stats.SuccessfulDelivered
		totalFailed += stats.Failed
		totalLatency += stats.TotalLatency
		latencyCount += stats.TotalAttempts
	}

	// Calculate overall delivery rate
	var deliveryRate float64
	totalAttempts := totalSent + totalDelivered + totalFailed
	if totalAttempts > 0 {
		deliveryRate = float64(totalDelivered) / float64(totalAttempts)
	}

	// Calculate average latency
	var avgLatency int64
	if latencyCount > 0 {
		avgLatency = totalLatency.Milliseconds() / latencyCount
	}

	return &models.MessageMetrics{
		TotalSent:      totalSent,
		TotalDelivered: totalDelivered,
		TotalFailed:    totalFailed,
		DeliveryRate:   deliveryRate,
		AverageLatency: avgLatency,
		ByChannel:      byChannel,
	}, nil
}

// GetChannelStats returns statistics for a specific channel
func (c *InMemoryTelemetryCollector) GetChannelStats(channel models.DeliveryChannel) *ChannelStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if stats, exists := c.deliveryMetrics[channel]; exists {
		// Return a copy to avoid race conditions
		statsCopy := *stats
		return &statsCopy
	}
	return nil
}

// GetCounters returns all counters
func (c *InMemoryTelemetryCollector) GetCounters() map[string]int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	countersCopy := make(map[string]int64)
	for k, v := range c.counters {
		countersCopy[k] = v
	}
	return countersCopy
}

// GetGauges returns all gauges
func (c *InMemoryTelemetryCollector) GetGauges() map[string]float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	gaugesCopy := make(map[string]float64)
	for k, v := range c.gauges {
		gaugesCopy[k] = v
	}
	return gaugesCopy
}

// GetRecentErrors returns recent errors for a channel
func (c *InMemoryTelemetryCollector) GetRecentErrors(channel models.DeliveryChannel, limit int) []ErrorRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if errors, exists := c.errors[channel]; exists {
		if limit <= 0 || limit > len(errors) {
			limit = len(errors)
		}

		// Return most recent errors
		start := len(errors) - limit
		if start < 0 {
			start = 0
		}

		result := make([]ErrorRecord, limit)
		copy(result, errors[start:])
		return result
	}
	return []ErrorRecord{}
}

// GetRecentAttempts returns recent delivery attempts
func (c *InMemoryTelemetryCollector) GetRecentAttempts(limit int) []AttemptRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if limit <= 0 || limit > len(c.attempts) {
		limit = len(c.attempts)
	}

	start := len(c.attempts) - limit
	if start < 0 {
		start = 0
	}

	result := make([]AttemptRecord, limit)
	copy(result, c.attempts[start:])
	return result
}

// GetLatencyStats returns latency statistics for a channel
func (c *InMemoryTelemetryCollector) GetLatencyStats(channel models.DeliveryChannel) map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	latencies, exists := c.latencies[channel]
	if !exists || len(latencies) == 0 {
		return map[string]interface{}{}
	}

	// Calculate percentiles
	sortedLatencies := make([]time.Duration, len(latencies))
	copy(sortedLatencies, latencies)

	// Simple sorting (for production, use a more efficient sort)
	for i := 0; i < len(sortedLatencies)-1; i++ {
		for j := i + 1; j < len(sortedLatencies); j++ {
			if sortedLatencies[i] > sortedLatencies[j] {
				sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
			}
		}
	}

	stats := map[string]interface{}{
		"count": len(sortedLatencies),
		"min":   sortedLatencies[0].Milliseconds(),
		"max":   sortedLatencies[len(sortedLatencies)-1].Milliseconds(),
	}

	// Calculate percentiles
	if len(sortedLatencies) > 0 {
		stats["p50"] = sortedLatencies[len(sortedLatencies)*50/100].Milliseconds()
		stats["p90"] = sortedLatencies[len(sortedLatencies)*90/100].Milliseconds()
		stats["p95"] = sortedLatencies[len(sortedLatencies)*95/100].Milliseconds()
		stats["p99"] = sortedLatencies[len(sortedLatencies)*99/100].Milliseconds()
	}

	return stats
}

// Reset clears all collected metrics (useful for testing)
func (c *InMemoryTelemetryCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.deliveryMetrics = make(map[models.DeliveryChannel]*ChannelStats)
	c.counters = make(map[string]int64)
	c.gauges = make(map[string]float64)
	c.latencies = make(map[models.DeliveryChannel][]time.Duration)
	c.errors = make(map[models.DeliveryChannel][]ErrorRecord)
	c.attempts = make([]AttemptRecord, 0)

	log.Println("Telemetry: Reset all metrics")
}
