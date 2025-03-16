package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"webhook-receiver/internal/model"

	"go.uber.org/zap"
)

// Batcher handles log batching based on size and time interval.
type Batcher struct {
	mu       sync.Mutex
	payloads []model.Payload
	size     int
	interval time.Duration
	endpoint string
	ticker   *time.Ticker
	stop     chan struct{}
	logger   *zap.SugaredLogger
	lastSent time.Time
}

type ServiceBatcher interface{
	Add(payload model.Payload)
}

// NewBatcher initializes the batch processor.
func NewBatcher(size, interval int, endpoint string, logger *zap.SugaredLogger) *Batcher {
	return &Batcher{
		size:     size,
		interval: time.Duration(interval) * time.Second,
		endpoint: endpoint,
		ticker:   time.NewTicker(time.Duration(interval) * time.Second),
		stop:     make(chan struct{}),
		logger:   logger,
		lastSent: time.Now(),
	}
}

// Add stores a payload and checks if a batch should be sent.
func (b *Batcher) Add(payload model.Payload) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.payloads = append(b.payloads, payload)

	b.logger.Infof("New log added. Current batch size: %d", len(b.payloads))

	// Send batch immediately if size limit is reached.
	if len(b.payloads) >= b.size {
		b.logger.Infof("Batch size limit (%d). reached. Sending batch...", len(b.payloads))
		b.sendBatch()
	}
}

// Run starts the timer-based batching.
func (b *Batcher) Run() {
	for {
		select {
		case <-b.ticker.C:
			b.mu.Lock()
			elapsed := time.Since(b.lastSent)
			if len(b.payloads) > 0 && elapsed >= b.interval {
				b.logger.Infof("Time-based trigger: %v seconds. Sending batch...", time.Since(b.lastSent).Seconds())
				b.sendBatch()
			} else {
				b.logger.Debugf("Batch interval elapsed but no logs to send.")
			}
			b.mu.Unlock()
		case <-b.stop:
			b.logger.Info("Batch processor stopping.")
			return
		}
	}
}

//  sendBatch processes and clears the batch after successful forwarding.
func (b *Batcher) sendBatch() {
	if len(b.payloads) == 0 {
		b.logger.Info("Skipping batch send (empty batch).")
		return
	}

	data, err := json.Marshal(b.payloads)
	if err != nil {
		b.logger.Errorf("Failed to serialize batch: %v", err)
		return
	}

	start := time.Now()
	err = b.postWithRetry(data, 3, 2*time.Second)
	duration := time.Since(start)

	if err != nil {
		b.logger.Errorf("Batch sending failed: %v", err)
		return
	}

	b.logger.Infof("Successfully sent batch of %d records in %s", len(b.payloads), duration)
	b.payloads = nil
	b.lastSent = time.Now()
}

// postWithRetry attempts to send a batch, retrying if necessary.
func (b *Batcher) postWithRetry(data []byte, maxRetries int, delay time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Post(b.endpoint, "application/json", bytes.NewBuffer(data))
		if err == nil && resp.StatusCode < 300 {
			return nil
		}
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}

		b.logger.Warnf("Batch send attempt %d failed. Retrying in %s...", i+1, delay)
		time.Sleep(delay)
	}

	return errors.New("max retries exceeded")
}

// Stop gracefully shuts down batch processing.
func (b *Batcher) Stop() {
	b.ticker.Stop()
	close(b.stop)
}
