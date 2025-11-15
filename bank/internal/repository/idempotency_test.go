package repository

import (
	"context"
	"testing"
	"time"

	"github.com/benx421/payment-gateway/bank/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdempotencyRepository_Store_And_Get(t *testing.T) {
	database := setupTestDB(t)
	defer cleanupTestDB(t, database)
	truncateTables(t, database)

	repo := NewIdempotencyRepository(database)

	tests := []struct {
		name        string
		key         string
		requestPath string
		body        string
		status      int
	}{
		{
			name:        "store and retrieve simple key",
			key:         "test-key-1",
			requestPath: "/api/v1/authorizations",
			status:      200,
			body:        `{"status":"success"}`,
		},
		{
			name:        "store and retrieve different path",
			key:         "test-key-2",
			requestPath: "/api/v1/captures",
			status:      201,
			body:        `{"capture_id":"cap_123"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idemKey := &models.IdempotencyKey{
				Key:            tt.key,
				RequestPath:    tt.requestPath,
				ResponseStatus: tt.status,
				ResponseBody:   tt.body,
			}

			err := repo.Store(context.Background(), idemKey)
			require.NoError(t, err, "failed to store idempotency key")

			retrieved, err := repo.Get(context.Background(), tt.key, tt.requestPath)
			require.NoError(t, err, "failed to get idempotency key")
			require.NotNil(t, retrieved, "expected idempotency key")

			assert.Equal(t, tt.key, retrieved.Key, "key mismatch")
			assert.Equal(t, tt.requestPath, retrieved.RequestPath, "request path mismatch")
			assert.Equal(t, tt.status, retrieved.ResponseStatus, "status mismatch")
			assert.Equal(t, tt.body, retrieved.ResponseBody, "body mismatch")
		})
	}
}

func TestIdempotencyRepository_Get_NotFound(t *testing.T) {
	database := setupTestDB(t)
	defer cleanupTestDB(t, database)
	truncateTables(t, database)

	repo := NewIdempotencyRepository(database)

	result, err := repo.Get(context.Background(), "non-existent-key", "/api/v1/test")
	require.NoError(t, err, "unexpected error")
	assert.Nil(t, result, "expected nil for non-existent key")
}

func TestIdempotencyRepository_Store_OnConflict(t *testing.T) {
	database := setupTestDB(t)
	defer cleanupTestDB(t, database)
	truncateTables(t, database)

	repo := NewIdempotencyRepository(database)

	key := "duplicate-key"
	path := "/api/v1/authorizations"

	first := &models.IdempotencyKey{
		Key:            key,
		RequestPath:    path,
		ResponseStatus: 200,
		ResponseBody:   `{"first":"response"}`,
	}
	err := repo.Store(context.Background(), first)
	require.NoError(t, err, "failed to store first key")

	// Try to store again with different response (should be ignored due to ON CONFLICT)
	second := &models.IdempotencyKey{
		Key:            key,
		RequestPath:    path,
		ResponseStatus: 400,
		ResponseBody:   `{"second":"response"}`,
	}
	err = repo.Store(context.Background(), second)
	require.NoError(t, err, "failed to store second key")

	retrieved, err := repo.Get(context.Background(), key, path)
	require.NoError(t, err, "failed to get key")

	assert.Equal(t, first.ResponseStatus, retrieved.ResponseStatus, "first response should win (status)")
	assert.Equal(t, first.ResponseBody, retrieved.ResponseBody, "first response should win (body)")
}

func TestIdempotencyRepository_SameKey_DifferentPath(t *testing.T) {
	database := setupTestDB(t)
	defer cleanupTestDB(t, database)
	truncateTables(t, database)

	repo := NewIdempotencyRepository(database)

	key := "same-key"

	first := &models.IdempotencyKey{
		Key:            key,
		RequestPath:    "/api/v1/authorizations",
		ResponseStatus: 200,
		ResponseBody:   `{"auth":"response"}`,
	}
	err := repo.Store(context.Background(), first)
	require.NoError(t, err, "failed to store first")

	second := &models.IdempotencyKey{
		Key:            key,
		RequestPath:    "/api/v1/captures",
		ResponseStatus: 201,
		ResponseBody:   `{"capture":"response"}`,
	}
	err = repo.Store(context.Background(), second)
	require.NoError(t, err, "failed to store second")

	retrieved1, err := repo.Get(context.Background(), key, "/api/v1/authorizations")
	require.NoError(t, err, "failed to get first")
	assert.Equal(t, first.ResponseBody, retrieved1.ResponseBody, "first path body mismatch")

	retrieved2, err := repo.Get(context.Background(), key, "/api/v1/captures")
	require.NoError(t, err, "failed to get second")
	assert.Equal(t, second.ResponseBody, retrieved2.ResponseBody, "second path body mismatch")
}

func TestIdempotencyRepository_DeleteOlderThan(t *testing.T) {
	database := setupTestDB(t)
	defer cleanupTestDB(t, database)
	truncateTables(t, database)

	repo := NewIdempotencyRepository(database)

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	oldKey := &models.IdempotencyKey{
		Key:            "old-key",
		RequestPath:    "/api/v1/test",
		ResponseStatus: 200,
		ResponseBody:   "old",
		CreatedAt:      yesterday.Add(-1 * time.Hour), // 25 hours ago
	}
	err := repo.Store(context.Background(), oldKey)
	require.NoError(t, err, "failed to store old key")

	// Store recent key
	recentKey := &models.IdempotencyKey{
		Key:            "recent-key",
		RequestPath:    "/api/v1/test",
		ResponseStatus: 200,
		ResponseBody:   "recent",
		CreatedAt:      now.Add(-1 * time.Hour), // 1 hour ago
	}
	err = repo.Store(context.Background(), recentKey)
	require.NoError(t, err, "failed to store recent key")

	// Delete keys older than 24 hours
	deletedCount, err := repo.DeleteOlderThan(context.Background(), yesterday)
	require.NoError(t, err, "failed to delete old keys")
	assert.Equal(t, int64(1), deletedCount, "deleted count mismatch")

	oldResult, err := repo.Get(context.Background(), "old-key", "/api/v1/test")
	require.NoError(t, err, "unexpected error checking old key")
	assert.Nil(t, oldResult, "old key should have been deleted")

	recentResult, err := repo.Get(context.Background(), "recent-key", "/api/v1/test")
	require.NoError(t, err, "unexpected error checking recent key")
	assert.NotNil(t, recentResult, "recent key should still exist")
}

func TestIdempotencyRepository_DeleteOlderThan_NoneDeleted(t *testing.T) {
	database := setupTestDB(t)
	defer cleanupTestDB(t, database)

	repo := NewIdempotencyRepository(database)

	_, err := database.ExecContext(context.Background(), "DELETE FROM idempotency_keys")
	require.NoError(t, err, "failed to clean idempotency keys")

	recentKey := &models.IdempotencyKey{
		Key:            "recent-key",
		RequestPath:    "/api/v1/test",
		ResponseStatus: 200,
		ResponseBody:   "recent",
		CreatedAt:      time.Now(), // Explicitly set to now
	}
	err = repo.Store(context.Background(), recentKey)
	require.NoError(t, err, "failed to store key")

	// Try to delete keys older than 1 year ago (should delete nothing)
	veryOld := time.Now().Add(-365 * 24 * time.Hour)
	deletedCount, err := repo.DeleteOlderThan(context.Background(), veryOld)
	require.NoError(t, err, "unexpected error")
	assert.Equal(t, int64(0), deletedCount, "deleted count should be 0")
}
