package reviews

import (
	"context"
	"sync"
)

// Broker delivers live events within one Reviews process. It is intentionally
// not a durable queue; use shared pub/sub before running multiple replicas.
type Broker struct {
	mu          sync.Mutex
	subscribers map[chan *Review]string
}

func NewBroker() *Broker { return &Broker{subscribers: make(map[chan *Review]string)} }
func (b *Broker) Subscribe(ctx context.Context, bookID string) <-chan *Review {
	ch := make(chan *Review, 16)
	b.mu.Lock()
	b.subscribers[ch] = bookID
	b.mu.Unlock()
	go func() {
		<-ctx.Done()
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subscribers[ch]; ok {
			delete(b.subscribers, ch)
			close(ch)
		}
	}()
	return ch
}
func (b *Broker) Publish(review *Review) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch, bookID := range b.subscribers {
		if bookID != review.Book.ID {
			continue
		}
		select {
		case ch <- review:
		default:
			// End slow subscriptions rather than block writes or silently drop events.
			delete(b.subscribers, ch)
			close(ch)
		}
	}
}
