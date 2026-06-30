package probe

import (
	"sync"

	"github.com/lwshen/go-server-monitor/internal/models"
)

// SampleBuffer is a bounded ring buffer of StatReport samples (REQ-PROBE-08).
// When full, Append discards the oldest sample. This is real (pure & simple),
// not a stub.
type SampleBuffer struct {
	mu      sync.Mutex
	samples []*models.StatReport
	maxSize int
}

// NewSampleBuffer creates a buffer holding up to maxSize samples (min 1).
func NewSampleBuffer(maxSize int) *SampleBuffer {
	if maxSize < 1 {
		maxSize = 1
	}
	return &SampleBuffer{
		samples: make([]*models.StatReport, 0, maxSize),
		maxSize: maxSize,
	}
}

// Append adds a sample, evicting the oldest if the buffer is full.
func (b *SampleBuffer) Append(s *models.StatReport) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.samples) >= b.maxSize {
		// Drop the oldest sample.
		b.samples = b.samples[1:]
	}
	b.samples = append(b.samples, s)
}

// Drain returns all buffered samples and clears the buffer.
func (b *SampleBuffer) Drain() []*models.StatReport {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.samples) == 0 {
		return nil
	}
	out := b.samples
	b.samples = make([]*models.StatReport, 0, b.maxSize)
	return out
}

// Len returns the current number of buffered samples.
func (b *SampleBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.samples)
}
