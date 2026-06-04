package auth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testSecret = []byte("test-secret-do-not-use-in-prod")

func TestSignAndVerifyToken_RoundTrip(t *testing.T) {
	token, err := auth.SignToken(42, true, testSecret)
	require.NoError(t, err)

	claims, err := auth.VerifyToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.UserID)
	assert.True(t, claims.IsAdmin)
}

func TestVerifyToken_WrongSecret_Rejected(t *testing.T) {
	token, err := auth.SignToken(42, false, testSecret)
	require.NoError(t, err)

	_, err = auth.VerifyToken(token, []byte("a-different-secret"))
	assert.Error(t, err)
}

func TestVerifyToken_TamperedPayload_Rejected(t *testing.T) {
	token, err := auth.SignToken(42, false, testSecret)
	require.NoError(t, err)

	// Swap a character in the payload (header.payload.signature).
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	payload := []byte(parts[1])
	if payload[0] == 'A' {
		payload[0] = 'B'
	} else {
		payload[0] = 'A'
	}
	tampered := parts[0] + "." + string(payload) + "." + parts[2]

	_, err = auth.VerifyToken(tampered, testSecret)
	assert.Error(t, err)
}

func TestVerifyToken_Expired_Rejected(t *testing.T) {
	// SignToken always issues 30-day tokens, so craft an expired one directly
	// with the same claims shape and signing method.
	claims := auth.Claims{
		UserID: 42,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(testSecret)
	require.NoError(t, err)

	_, err = auth.VerifyToken(token, testSecret)
	assert.Error(t, err)
}

func TestVerifyToken_NoneAlgorithm_Rejected(t *testing.T) {
	// Classic alg-confusion attack: an unsigned token must never verify.
	claims := auth.Claims{
		UserID:  42,
		IsAdmin: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = auth.VerifyToken(token, testSecret)
	assert.Error(t, err)
}

func TestVerifyToken_Garbage_Rejected(t *testing.T) {
	_, err := auth.VerifyToken("not.a.jwt", testSecret)
	assert.Error(t, err)
}
