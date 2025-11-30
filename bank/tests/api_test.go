//nolint:errcheck // unchecked errors are acceptable in test files
package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthorizeAndCapture(t *testing.T) {
	ts := SetupTest(t)
	defer ts.Close()

	authResp := ts.Authorize(t, "4111111111111111", "123", 10000, "auth-cap-key-1")
	require.Equal(t, http.StatusOK, authResp.StatusCode)

	var authBody map[string]any
	require.NoError(t, json.NewDecoder(authResp.Body).Decode(&authBody))
	authResp.Body.Close()

	assert.Equal(t, "approved", authBody["status"])
	assert.Equal(t, float64(10000), authBody["amount"])
	authID := authBody["authorization_id"].(string)
	assert.Contains(t, authID, "auth_")

	captureResp := ts.Capture(t, authID, 10000, "cap-key-1")
	require.Equal(t, http.StatusOK, captureResp.StatusCode)

	var captureBody map[string]any
	require.NoError(t, json.NewDecoder(captureResp.Body).Decode(&captureBody))
	captureResp.Body.Close()

	assert.Equal(t, "captured", captureBody["status"])
	assert.Equal(t, authID, captureBody["authorization_id"])
	assert.Contains(t, captureBody["capture_id"].(string), "cap_")
}

func TestAuthorizeAndVoid(t *testing.T) {
	ts := SetupTest(t)
	defer ts.Close()

	authResp := ts.Authorize(t, "4111111111111111", "123", 20000, "auth-void-key-1")
	require.Equal(t, http.StatusOK, authResp.StatusCode)

	var authBody map[string]any
	require.NoError(t, json.NewDecoder(authResp.Body).Decode(&authBody))
	authResp.Body.Close()
	authID := authBody["authorization_id"].(string)

	voidResp := ts.Void(t, authID, "void-key-1")
	require.Equal(t, http.StatusOK, voidResp.StatusCode)

	var voidBody map[string]any
	require.NoError(t, json.NewDecoder(voidResp.Body).Decode(&voidBody))
	voidResp.Body.Close()

	assert.Equal(t, "voided", voidBody["status"])
	assert.Equal(t, authID, voidBody["authorization_id"])
}

func TestFullFlow_AuthorizeCaptureRefund(t *testing.T) {
	ts := SetupTest(t)
	defer ts.Close()

	authResp := ts.Authorize(t, "4111111111111111", "123", 15000, "full-flow-auth-1")
	require.Equal(t, http.StatusOK, authResp.StatusCode)

	var authBody map[string]any
	require.NoError(t, json.NewDecoder(authResp.Body).Decode(&authBody))
	authResp.Body.Close()
	authID := authBody["authorization_id"].(string)

	captureResp := ts.Capture(t, authID, 15000, "full-flow-cap-1")
	require.Equal(t, http.StatusOK, captureResp.StatusCode)

	var captureBody map[string]any
	require.NoError(t, json.NewDecoder(captureResp.Body).Decode(&captureBody))
	captureResp.Body.Close()
	captureID := captureBody["capture_id"].(string)

	refundResp := ts.Refund(t, captureID, 15000, "full-flow-refund-1")
	require.Equal(t, http.StatusOK, refundResp.StatusCode)

	var refundBody map[string]any
	require.NoError(t, json.NewDecoder(refundResp.Body).Decode(&refundBody))
	refundResp.Body.Close()

	assert.Equal(t, "refunded", refundBody["status"])
	assert.Equal(t, captureID, refundBody["capture_id"])
	assert.Contains(t, refundBody["refund_id"].(string), "ref_")
}

func TestAuthorization_InvalidCard(t *testing.T) {
	ts := SetupTest(t)
	defer ts.Close()

	resp := ts.Authorize(t, "4111111111111112", "123", 10000, "invalid-card-key") // Invalid Luhn
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	resp.Body.Close()

	assert.Equal(t, "invalid_card", body["error"])
}

func TestAuthorization_InvalidCVV(t *testing.T) {
	ts := SetupTest(t)
	defer ts.Close()

	resp := ts.Authorize(t, "4111111111111111", "999", 10000, "invalid-cvv-key")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	resp.Body.Close()

	assert.Equal(t, "invalid_cvv", body["error"])
}

func TestAuthorization_ExpiredCard(t *testing.T) {
	ts := SetupTest(t)
	defer ts.Close()

	resp := ts.Authorize(t, "5105105105105100", "321", 10000, "expired-card-key") // Expiry 03/2020
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	resp.Body.Close()

	assert.Equal(t, "card_expired", body["error"])
}

func TestAuthorization_InsufficientFunds(t *testing.T) {
	ts := SetupTest(t)
	defer ts.Close()

	resp := ts.Authorize(t, "5555555555554444", "789", 100, "insufficient-funds-key") // Balance: $0
	require.Equal(t, http.StatusPaymentRequired, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	resp.Body.Close()

	assert.Equal(t, "insufficient_funds", body["error"])
}

func TestCapture_AuthorizationAlreadyUsed(t *testing.T) {
	ts := SetupTest(t)
	defer ts.Close()

	authResp := ts.Authorize(t, "4111111111111111", "123", 10000, "double-cap-auth")
	require.Equal(t, http.StatusOK, authResp.StatusCode)

	var authBody map[string]any
	require.NoError(t, json.NewDecoder(authResp.Body).Decode(&authBody))
	authResp.Body.Close()
	authID := authBody["authorization_id"].(string)

	cap1 := ts.Capture(t, authID, 10000, "double-cap-1")
	require.Equal(t, http.StatusOK, cap1.StatusCode)
	cap1.Body.Close()

	cap2 := ts.Capture(t, authID, 10000, "double-cap-2")
	require.Equal(t, http.StatusBadRequest, cap2.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(cap2.Body).Decode(&body))
	cap2.Body.Close()

	assert.Equal(t, "authorization_already_used", body["error"])
}

func TestVoid_AfterCapture(t *testing.T) {
	ts := SetupTest(t)
	defer ts.Close()

	authResp := ts.Authorize(t, "4111111111111111", "123", 10000, "void-after-cap-auth")
	require.Equal(t, http.StatusOK, authResp.StatusCode)

	var authBody map[string]any
	require.NoError(t, json.NewDecoder(authResp.Body).Decode(&authBody))
	authResp.Body.Close()
	authID := authBody["authorization_id"].(string)

	capResp := ts.Capture(t, authID, 10000, "void-after-cap-cap")
	require.Equal(t, http.StatusOK, capResp.StatusCode)
	capResp.Body.Close()

	voidResp := ts.Void(t, authID, "void-after-cap-void")
	require.Equal(t, http.StatusBadRequest, voidResp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(voidResp.Body).Decode(&body))
	voidResp.Body.Close()

	// After capture, auth status is COMPLETED so we get "authorization_already_used"
	assert.Equal(t, "authorization_already_used", body["error"])
}

func TestIdempotency_ReplaysSameResponse(t *testing.T) {
	ts := SetupTest(t)
	defer ts.Close()

	idempotencyKey := "replay-test-key"

	resp1 := ts.Authorize(t, "4111111111111111", "123", 10000, idempotencyKey)
	require.Equal(t, http.StatusOK, resp1.StatusCode)
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	resp2 := ts.Authorize(t, "4111111111111111", "123", 10000, idempotencyKey)
	require.Equal(t, http.StatusOK, resp2.StatusCode)
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	assert.Equal(t, string(body1), string(body2))
	assert.Equal(t, "true", resp2.Header.Get("X-Idempotent-Replayed"))
}

func TestIdempotency_DifferentKeysCreateDifferentAuthorizations(t *testing.T) {
	ts := SetupTest(t)
	defer ts.Close()

	resp1 := ts.Authorize(t, "4111111111111111", "123", 10000, "key-1")
	require.Equal(t, http.StatusOK, resp1.StatusCode)

	var body1 map[string]any
	require.NoError(t, json.NewDecoder(resp1.Body).Decode(&body1))
	resp1.Body.Close()

	resp2 := ts.Authorize(t, "4111111111111111", "123", 10000, "key-2")
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	var body2 map[string]any
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&body2))
	resp2.Body.Close()

	assert.NotEqual(t, body1["authorization_id"], body2["authorization_id"])
}

func TestConcurrentCaptures_OnlyOneSucceeds(t *testing.T) {
	ts := SetupTest(t)
	defer ts.Close()

	authResp := ts.Authorize(t, "4111111111111111", "123", 10000, "concurrent-cap-auth")
	require.Equal(t, http.StatusOK, authResp.StatusCode)

	var authBody map[string]any
	require.NoError(t, json.NewDecoder(authResp.Body).Decode(&authBody))
	authResp.Body.Close()
	authID := authBody["authorization_id"].(string)

	const numGoroutines = 10
	var wg sync.WaitGroup
	results := make(chan int, numGoroutines)

	for i := range numGoroutines {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			resp := ts.Capture(t, authID, 10000, "concurrent-cap-"+string(rune('a'+idx)))
			results <- resp.StatusCode
			resp.Body.Close()
		}(i)
	}

	wg.Wait()
	close(results)

	successCount := 0
	failCount := 0
	for code := range results {
		switch code {
		case http.StatusOK:
			successCount++
		case http.StatusBadRequest:
			failCount++
		}
	}

	assert.Equal(t, 1, successCount, "exactly one capture should succeed")
	assert.Equal(t, numGoroutines-1, failCount, "all others should fail")
}

func TestGetAuthorization(t *testing.T) {
	ts := SetupTest(t)
	defer ts.Close()

	authResp := ts.Authorize(t, "4111111111111111", "123", 10000, "get-auth-key")
	require.Equal(t, http.StatusOK, authResp.StatusCode)

	var authBody map[string]any
	require.NoError(t, json.NewDecoder(authResp.Body).Decode(&authBody))
	authResp.Body.Close()
	authID := authBody["authorization_id"].(string)

	resp, err := http.Get(ts.URL("/api/v1/authorizations/" + authID))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var getBody map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&getBody))
	resp.Body.Close()

	assert.Equal(t, authID, getBody["authorization_id"])
	assert.Equal(t, float64(10000), getBody["amount"])
}

func TestGetAuthorization_NotFound(t *testing.T) {
	ts := SetupTest(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL("/api/v1/authorizations/auth_00000000-0000-0000-0000-000000000000"))
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}
