package middleware

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benx421/payment-gateway/bank/internal/models"
	"github.com/benx421/payment-gateway/bank/internal/service/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func testHandler(status int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body)) //nolint:errcheck // test helper
	})
}

func TestIdempotency_GETRequestsBypassed(t *testing.T) {
	repo := mocks.NewMockIdempotencyRepository(t)
	middleware := Idempotency(repo, testLogger())

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/authorizations", nil)
	req.Header.Set("Idempotency-Key", "test-key")
	rec := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rec, req)

	assert.True(t, handlerCalled, "handler should be called for GET requests")
	repo.AssertNotCalled(t, "Get")
	repo.AssertNotCalled(t, "Store")
}

func TestIdempotency_NonIdempotentPathBypassed(t *testing.T) {
	repo := mocks.NewMockIdempotencyRepository(t)
	middleware := Idempotency(repo, testLogger())

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/other", nil)
	req.Header.Set("Idempotency-Key", "test-key")
	rec := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rec, req)

	assert.True(t, handlerCalled, "handler should be called for non-idempotent paths")
	repo.AssertNotCalled(t, "Get")
	repo.AssertNotCalled(t, "Store")
}

func TestIdempotency_MissingKeyPassesThrough(t *testing.T) {
	repo := mocks.NewMockIdempotencyRepository(t)
	middleware := Idempotency(repo, testLogger())

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/authorizations", nil)
	// No Idempotency-Key header
	rec := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rec, req)

	assert.True(t, handlerCalled, "handler should be called without idempotency key")
	repo.AssertNotCalled(t, "Get")
	repo.AssertNotCalled(t, "Store")
}

func TestIdempotency_FirstRequestCached(t *testing.T) {
	repo := mocks.NewMockIdempotencyRepository(t)
	repo.On("Get", mock.Anything, "unique-key-123", "/api/v1/authorizations").Return(nil, nil)
	repo.On("Store", mock.Anything, mock.AnythingOfType("*models.IdempotencyKey")).Return(nil)

	middleware := Idempotency(repo, testLogger())
	handler := testHandler(http.StatusOK, `{"status":"success"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/authorizations", nil)
	req.Header.Set("Idempotency-Key", "unique-key-123")
	rec := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, `{"status":"success"}`, rec.Body.String())
	assert.Empty(t, rec.Header().Get("X-Idempotent-Replayed"), "first request should not have replay header")

	repo.AssertCalled(t, "Store", mock.Anything, mock.AnythingOfType("*models.IdempotencyKey"))
}

func TestIdempotency_SecondRequestReturnsCached(t *testing.T) {
	repo := mocks.NewMockIdempotencyRepository(t)

	// First call returns nil (no cache), second returns cached value
	cached := &models.IdempotencyKey{
		Key:            "duplicate-key",
		RequestPath:    "/api/v1/authorizations",
		ResponseStatus: 200,
		ResponseBody:   `{"call":1}`,
	}
	repo.On("Get", mock.Anything, "duplicate-key", "/api/v1/authorizations").Return(cached, nil)

	middleware := Idempotency(repo, testLogger())

	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"call":` + string(rune('0'+callCount)) + `}`)) //nolint:errcheck // test helper
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/authorizations", nil)
	req.Header.Set("Idempotency-Key", "duplicate-key")
	rec := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rec, req)

	assert.Equal(t, 0, callCount, "handler should not be called when cached")
	assert.Equal(t, "true", rec.Header().Get("X-Idempotent-Replayed"))
	assert.Equal(t, `{"call":1}`, rec.Body.String())
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestIdempotency_SameKeyDifferentPathsAreSeparate(t *testing.T) {
	repo := mocks.NewMockIdempotencyRepository(t)
	repo.On("Get", mock.Anything, "shared-key", mock.Anything).Return(nil, nil)
	repo.On("Store", mock.Anything, mock.AnythingOfType("*models.IdempotencyKey")).Return(nil)

	middleware := Idempotency(repo, testLogger())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"path":"` + r.URL.Path + `"}`)) //nolint:errcheck // test helper
	})

	// Request to authorizations
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/authorizations", nil)
	req1.Header.Set("Idempotency-Key", "shared-key")
	rec1 := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rec1, req1)

	// Request to captures with same key
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/captures", nil)
	req2.Header.Set("Idempotency-Key", "shared-key")
	rec2 := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rec2, req2)

	assert.Contains(t, rec1.Body.String(), "authorizations")
	assert.Contains(t, rec2.Body.String(), "captures")

	// Verify Get was called with different paths
	repo.AssertCalled(t, "Get", mock.Anything, "shared-key", "/api/v1/authorizations")
	repo.AssertCalled(t, "Get", mock.Anything, "shared-key", "/api/v1/captures")
}

func TestIdempotency_5xxResponsesNotCached(t *testing.T) {
	repo := mocks.NewMockIdempotencyRepository(t)
	repo.On("Get", mock.Anything, "error-key", "/api/v1/authorizations").Return(nil, nil)
	// Store should NOT be called for 5xx responses

	middleware := Idempotency(repo, testLogger())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"server error"}`)) //nolint:errcheck // test helper
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/authorizations", nil)
	req.Header.Set("Idempotency-Key", "error-key")
	rec := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	repo.AssertNotCalled(t, "Store")
}

func TestIdempotency_4xxResponsesNotCached(t *testing.T) {
	repo := mocks.NewMockIdempotencyRepository(t)
	repo.On("Get", mock.Anything, "bad-request-key", "/api/v1/authorizations").Return(nil, nil)

	middleware := Idempotency(repo, testLogger())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`)) //nolint:errcheck // test helper
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/authorizations", nil)
	req.Header.Set("Idempotency-Key", "bad-request-key")
	rec := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	repo.AssertNotCalled(t, "Store")
}

func TestIdempotency_RepoGetErrorFailsOpen(t *testing.T) {
	repo := mocks.NewMockIdempotencyRepository(t)
	repo.On("Get", mock.Anything, "test-key", "/api/v1/authorizations").Return(nil, errors.New("database connection failed"))

	middleware := Idempotency(repo, testLogger())

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/authorizations", nil)
	req.Header.Set("Idempotency-Key", "test-key")
	rec := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rec, req)

	assert.True(t, handlerCalled, "handler should be called on repo.Get error (fail open)")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestIdempotency_RepoStoreErrorDoesNotAffectResponse(t *testing.T) {
	repo := mocks.NewMockIdempotencyRepository(t)
	repo.On("Get", mock.Anything, "test-key", "/api/v1/authorizations").Return(nil, nil)
	repo.On("Store", mock.Anything, mock.AnythingOfType("*models.IdempotencyKey")).Return(errors.New("failed to store"))

	middleware := Idempotency(repo, testLogger())
	handler := testHandler(http.StatusOK, `{"status":"success"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/authorizations", nil)
	req.Header.Set("Idempotency-Key", "test-key")
	rec := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rec, req)

	// Response should still be successful even if caching failed
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, `{"status":"success"}`, rec.Body.String())
}

func TestIdempotency_AllIdempotentPaths(t *testing.T) {
	paths := []string{
		"/api/v1/authorizations",
		"/api/v1/captures",
		"/api/v1/voids",
		"/api/v1/refunds",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			repo := mocks.NewMockIdempotencyRepository(t)
			repo.On("Get", mock.Anything, "test-key", path).Return(nil, nil)
			repo.On("Store", mock.Anything, mock.AnythingOfType("*models.IdempotencyKey")).Return(nil)

			middleware := Idempotency(repo, testLogger())
			handler := testHandler(http.StatusOK, `{"path":"`+path+`"}`)

			req := httptest.NewRequest(http.MethodPost, path, nil)
			req.Header.Set("Idempotency-Key", "test-key")
			rec := httptest.NewRecorder()

			middleware(handler).ServeHTTP(rec, req)

			repo.AssertCalled(t, "Store", mock.Anything, mock.AnythingOfType("*models.IdempotencyKey"))
		})
	}
}

func TestIdempotency_CachedResponseHasCorrectContentType(t *testing.T) {
	repo := mocks.NewMockIdempotencyRepository(t)

	cached := &models.IdempotencyKey{
		Key:            "content-type-key",
		RequestPath:    "/api/v1/authorizations",
		ResponseStatus: 200,
		ResponseBody:   `{"status":"success"}`,
	}
	repo.On("Get", mock.Anything, "content-type-key", "/api/v1/authorizations").Return(cached, nil)

	middleware := Idempotency(repo, testLogger())
	handler := testHandler(http.StatusOK, `{"status":"success"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/authorizations", nil)
	req.Header.Set("Idempotency-Key", "content-type-key")
	rec := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rec, req)

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}
