package transport_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dimaglobin/order-service/internal/transport"
)

func TestRequestID_GeneratesIfMissing(t *testing.T) {
	var seenInHandler string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seenInHandler = transport.RequestIDFrom(r.Context())
	})

	w := httptest.NewRecorder()
	transport.RequestID(next).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if seenInHandler == "" {
		t.Error("expected request ID in context, got empty")
	}
	if got := w.Header().Get("X-Request-ID"); got != seenInHandler {
		t.Errorf("response header %q != context value %q", got, seenInHandler)
	}
}

func TestRequestID_PropagatesIncomingHeader(t *testing.T) {
	const incoming = "incoming-rid-42"
	var seenInHandler string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seenInHandler = transport.RequestIDFrom(r.Context())
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", incoming)
	w := httptest.NewRecorder()
	transport.RequestID(next).ServeHTTP(w, req)

	if seenInHandler != incoming {
		t.Errorf("context: want %q, got %q", incoming, seenInHandler)
	}
	if got := w.Header().Get("X-Request-ID"); got != incoming {
		t.Errorf("response header: want %q, got %q", incoming, got)
	}
}

func TestRecover_PanicReturns500(t *testing.T) {
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	})

	w := httptest.NewRecorder()
	transport.Recover(discardLogger())(next).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: want %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestRecover_NoPanicPassesThrough(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	w := httptest.NewRecorder()
	transport.Recover(discardLogger())(next).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusTeapot {
		t.Errorf("status: want %d, got %d", http.StatusTeapot, w.Code)
	}
}

func TestChain_AppliesMiddlewareInOrder(t *testing.T) {
	var order []string
	mw := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "before:"+name)
				next.ServeHTTP(w, r)
				order = append(order, "after:"+name)
			})
		}
	}
	final := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		order = append(order, "handler")
	})

	chained := transport.Chain(final, mw("a"), mw("b"))
	chained.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	want := []string{"before:a", "before:b", "handler", "after:b", "after:a"}
	if len(order) != len(want) {
		t.Fatalf("order: want %v, got %v", want, order)
	}
	for i, v := range want {
		if order[i] != v {
			t.Errorf("order[%d]: want %q, got %q", i, v, order[i])
		}
	}
}
