package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/dimaglobin/notifier/internal/model"
)

// ── Test doubles ─────────────────────────────────────────────────────────────

// fakeReader replays a scripted sequence of FetchMessage results, then
// returns context.Canceled to terminate Consumer.Run gracefully.
type fakeReader struct {
	mu         sync.Mutex
	queue      []fetchResult
	idx        int
	committed  []kafka.Message // offsets in the order they were committed
	commitErr  error           // error to return from CommitMessages, if any
}

type fetchResult struct {
	msg kafka.Message
	err error
}

func (f *fakeReader) FetchMessage(_ context.Context) (kafka.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.idx >= len(f.queue) {
		// Queue exhausted → signal end-of-stream so Consumer.Run exits
		// promptly. The Consumer treats io.EOF as a normal shutdown.
		return kafka.Message{}, io.EOF
	}

	r := f.queue[f.idx]
	f.idx++
	return r.msg, r.err
}

func (f *fakeReader) CommitMessages(_ context.Context, msgs ...kafka.Message) error {
	if f.commitErr != nil {
		return f.commitErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.committed = append(f.committed, msgs...)
	return nil
}

func (f *fakeReader) committedOffsets() []int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]int64, len(f.committed))
	for i, m := range f.committed {
		out[i] = m.Offset
	}
	return out
}

// fakeHandler records every event passed to it, can be configured to fail
// on specific OrderIDs.
type fakeHandler struct {
	mu         sync.Mutex
	handled    []model.OrderEvent
	failOnIDs  map[int64]error // OrderID → error to return
}

func (h *fakeHandler) HandleOrderEvent(_ context.Context, evt model.OrderEvent) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handled = append(h.handled, evt)
	if err, ok := h.failOnIDs[evt.OrderID]; ok {
		return err
	}
	return nil
}

func (h *fakeHandler) handledIDs() []int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]int64, len(h.handled))
	for i, e := range h.handled {
		out[i] = e.OrderID
	}
	return out
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
}

// mustEvent JSON-marshals an event, failing the test on error.
func mustEvent(t *testing.T, evt model.OrderEvent) []byte {
	t.Helper()
	b, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	return b
}

// runConsumerWithTimeout runs Consumer.Run with a deadline so a hung test
// doesn't block the suite. Returns whatever Run returns.
func runConsumerWithTimeout(t *testing.T, c *Consumer) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	return c.Run(ctx)
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestConsumer_HappyPath_CommitsAfterEachHandle(t *testing.T) {
	reader := &fakeReader{
		queue: []fetchResult{
			{msg: kafka.Message{Offset: 1, Value: mustEvent(t, model.OrderEvent{Type: "order.created", OrderID: 1})}},
			{msg: kafka.Message{Offset: 2, Value: mustEvent(t, model.OrderEvent{Type: "order.created", OrderID: 2})}},
		},
	}
	handler := &fakeHandler{}

	c := NewConsumer(reader, handler, discardLogger())
	_ = runConsumerWithTimeout(t, c)

	if got := handler.handledIDs(); len(got) != 2 {
		t.Fatalf("expected 2 events handled, got %d: %v", len(got), got)
	}
	if got := reader.committedOffsets(); len(got) != 2 {
		t.Fatalf("expected 2 commits, got %d: %v", len(got), got)
	}
	if reader.committedOffsets()[0] != 1 || reader.committedOffsets()[1] != 2 {
		t.Errorf("committed in wrong order: %v", reader.committedOffsets())
	}
}

func TestConsumer_HandlerError_DoesNotCommit(t *testing.T) {
	// First message's handler fails — its offset must NOT be committed.
	// The second succeeds — its offset is committed normally. This is the
	// at-least-once guarantee: failed messages are re-delivered on next run.
	reader := &fakeReader{
		queue: []fetchResult{
			{msg: kafka.Message{Offset: 1, Value: mustEvent(t, model.OrderEvent{Type: "order.created", OrderID: 1})}},
			{msg: kafka.Message{Offset: 2, Value: mustEvent(t, model.OrderEvent{Type: "order.created", OrderID: 2})}},
		},
	}
	handler := &fakeHandler{
		failOnIDs: map[int64]error{1: errors.New("downstream dead")},
	}

	c := NewConsumer(reader, handler, discardLogger())
	_ = runConsumerWithTimeout(t, c)

	// Both messages were attempted.
	if len(handler.handledIDs()) != 2 {
		t.Fatalf("expected handler called twice (both attempted), got %d", len(handler.handledIDs()))
	}
	// Only the successful one (offset 2) is committed.
	committed := reader.committedOffsets()
	if len(committed) != 1 || committed[0] != 2 {
		t.Errorf("expected only offset 2 committed, got %v", committed)
	}
}

func TestConsumer_PoisonMessage_IsSkippedAndCommitted(t *testing.T) {
	// Invalid JSON in Value should not block the partition. The Consumer
	// must log + skip + COMMIT (otherwise Kafka redelivers the same bad
	// message forever).
	reader := &fakeReader{
		queue: []fetchResult{
			{msg: kafka.Message{Offset: 1, Value: []byte("{not valid json")}},
			{msg: kafka.Message{Offset: 2, Value: mustEvent(t, model.OrderEvent{Type: "order.created", OrderID: 99})}},
		},
	}
	handler := &fakeHandler{}

	c := NewConsumer(reader, handler, discardLogger())
	_ = runConsumerWithTimeout(t, c)

	// Handler invoked ONLY for the valid message.
	if got := handler.handledIDs(); len(got) != 1 || got[0] != 99 {
		t.Errorf("handler should see only the valid message, got %v", got)
	}
	// BOTH offsets committed — poison is "successfully skipped".
	if got := reader.committedOffsets(); len(got) != 2 {
		t.Errorf("expected both offsets committed, got %v", got)
	}
}

func TestConsumer_ContextCanceled_ReturnsCleanly(t *testing.T) {
	// blockingReader stalls FetchMessage until ctx cancels — simulates the
	// real reader waiting for new messages.
	reader := blockingReader{}
	c := NewConsumer(reader, &fakeHandler{}, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	if err := c.Run(ctx); err != nil {
		t.Errorf("Run should return nil on context cancel, got %v", err)
	}
}

// blockingReader's FetchMessage blocks until the passed context is canceled.
// Used to test graceful shutdown on context cancellation.
type blockingReader struct{}

func (blockingReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	<-ctx.Done()
	return kafka.Message{}, ctx.Err()
}

func (blockingReader) CommitMessages(context.Context, ...kafka.Message) error {
	return nil
}

func TestConsumer_EOFFromReader_ReturnsCleanly(t *testing.T) {
	// Some Kafka clients return io.EOF when the reader is closed during
	// shutdown. Consumer treats this as a normal stop.
	reader := &fakeReader{
		queue: []fetchResult{
			{err: io.EOF},
		},
	}
	c := NewConsumer(reader, &fakeHandler{}, discardLogger())

	if err := runConsumerWithTimeout(t, c); err != nil {
		t.Errorf("Run should return nil on io.EOF, got %v", err)
	}
}
