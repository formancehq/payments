package hmac

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGenerateReturnsAnHMACString(t *testing.T) {
	headers, _ := GenerateHeaders("api_key", "api_secret", "", false)
	expectedSignature := "Signature keyId=\"api_key\",algorithm=\"hmac-sha1\",headers=\"date x-mod-nonce\",signature=\""
	assert.Equal(t, expectedSignature, headers["Authorization"][0:86], "generate should return the hmac headers")
}

func TestGenerateReturnsADateHeader(t *testing.T) {
	injectMockDate()
	headers, _ := GenerateHeaders("api_key", "api_secret", "", false)
	expectedDate := "Thu, 02 Jan 2020 15:04:05 GMT"
	assert.Equal(t, expectedDate, headers["Date"])
}

func TestGenerateReturnsANonceHeaderWithExpectedValue(t *testing.T) {
	nonce := "thisIsTheNonce"
	headers, _ := GenerateHeaders("api_key", "api_secret", nonce, false)
	assert.Equal(t, nonce, headers["x-mod-nonce"])
}

func TestGenerateReturnsARetryHeaderWithTrueIfRetryIsExpected(t *testing.T) {
	headers, _ := GenerateHeaders("api_key", "api_secret", "", true)
	assert.Equal(t, "true", headers["x-mod-retry"])
}

func TestGenerateReturnsARetryHeaderWithFalseIfRetryIsNotExpected(t *testing.T) {
	headers, _ := GenerateHeaders("api_key", "api_secret", "", false)
	assert.Equal(t, "false", headers["x-mod-retry"])
}

func TestGenerateReturnsAGeneratedNonceHeaderIfNonceIsEmpty(t *testing.T) {
	headers, _ := GenerateHeaders("api_key", "api_secret", "", false)
	assert.True(t, headers["x-mod-nonce"] != "", "x-mod-nonce header should have been populated")
}

func TestGenerateThrowsErrorIfApiKeyIsNull(t *testing.T) {
	_, err := GenerateHeaders("", "api_secret", "", false)
	assert.Equal(t, "api_key cannot be empty", err.Message)
}

func TestGenerateThrowsErrorIfApiSecretIsNull(t *testing.T) {
	_, err := GenerateHeaders("api_key", "", "", false)
	assert.Equal(t, "api_secret cannot be empty", err.Message)
}

func injectMockDate() {
	dateNow = func() time.Time {
		now, _ := time.Parse(time.RFC1123, "Mon, 02 Jan 2020 15:04:05 GMT")
		return now
	}
}
