package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"webhook-receiver/internal/model"

	"go.uber.org/zap"
)

// Batcher handles log batching based on size and time interval using channels.
type Batcher struct {
	payloadCh chan model.Payload
	size      int
	interval  time.Duration
	endpoint  string
	logger    *zap.SugaredLogger
	done      chan struct{}
	quit      chan struct{}
}

// ServiceBatcher defines methods to add payloads and stop the batch processor.
type ServiceBatcher interface {
	Add(payload model.Payload)
	Stop()
}

// NewBatcher initializes the batch processor.
func NewBatcher(size, interval int, endpoint string, logger *zap.SugaredLogger) *Batcher {
	return &Batcher{
		payloadCh: make(chan model.Payload, size*2), // Buffered channel for non-blocking
		size:      size,
		interval:  time.Duration(interval) * time.Second,
		endpoint:  endpoint,
		logger:    logger,
		done:      make(chan struct{}),
		quit:      make(chan struct{}),
	}
}

// Add pushes a payload to the channel.
func (b *Batcher) Add(payload model.Payload) {
	select {
	case b.payloadCh <- payload:
		b.logger.Debug("Added payload to channel")
	default:
		b.logger.Warn("Payload channel is full, dropping payload")
	}
}

// Run starts the batch processing.
func (b *Batcher) Run() {
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	var batch []model.Payload

	for {
		select {
		case payload := <-b.payloadCh:
			batch = append(batch, payload)
			b.logger.Debug("Received payload from channel")
			if len(batch) >= b.size {
				b.sendBatch(batch)
				batch = nil
			}
		case <-ticker.C:
			if len(batch) > 0 {
				b.sendBatch(batch)
				batch = nil
			}
		case <-b.done:
			b.logger.Info("Received shutdown signal, flushing remaining payloads")
			if len(batch) > 0 {
				b.sendBatch(batch)
			}
			close(b.quit)
			return
		}
	}
}

// Stop batch processor gracefully.
func (b *Batcher) Stop() {
	close(b.done)
	<-b.quit
}

// sendBatch sends a batch of payloads to the endpoint.
func (b *Batcher) sendBatch(batch []model.Payload) {
	if len(batch) == 0 {
		b.logger.Warn("Attempted to send an empty batch")
		return
	}

	data, err := json.Marshal(batch)
	if err != nil {
		b.logger.Errorf("Failed to serialize batch: %v", err)
		return
	}

	start := time.Now()
	err = b.postWithRetry(data, 3, 2*time.Second)
	duration := time.Since(start)

	if err != nil {
		b.logger.Errorf("Failed to send batch after retries: %v", err)
	} else {
		b.logger.Infof("Successfully sent batch of %d records in %v", len(batch), duration)
	}
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

		b.logger.Warnf("Batch send attempt %d failed, retrying in %v", i+1, delay)
		time.Sleep(delay)
	}
	return errors.New("max retries exceeded")
}
