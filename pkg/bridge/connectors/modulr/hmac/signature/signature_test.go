package signature

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

func TestGenerateReturnsSignatureWithKeyId(t *testing.T) {
	signature := Build("api_key", "api_secret", "", date())
	expectedPrefix := "Signature keyId=\"api_key\","
	hasKeyId := strings.HasPrefix(signature, expectedPrefix)
	assert.True(t, hasKeyId, "HMAC signature must contain the keyId")
}

func TestGenerateReturnsSignatureWithAlgorithm(t *testing.T) {
	signature := Build("api_key", "api_secret", "", date())
	expectedAlgorithm := "algorithm=\"hmac-sha1\","
	actualValue := signature[26:48]
	assert.Equal(t, expectedAlgorithm, actualValue, "HMAC signature must contain the algorithm used")
}

func TestGenerateReturnsSignatureWithHeaders(t *testing.T) {
	signature := Build("api_key", "api_secret", "", date())
	expectedHeaders := "headers=\"date x-mod-nonce\","
	actualValue := signature[48:75]
	assert.Equal(t, expectedHeaders, actualValue, "HMAC signature must contain the headers")
}

func TestGenerateReturnsSignatureWithSignatureValue(t *testing.T) {
	signature := Build("api_key", "api_secret", "", date())
	expectedSignature := "signature=\""
	actualValue := signature[75:86]
	assert.Equal(t, expectedSignature, actualValue, "HMAC signature must contain the signature")
}

func TestGenerateReturnsHashedSignature(t *testing.T) {
	signature := Build("api_key", "api_secret", "", date())
	actualValue := signature[86:117]
	assert.True(t, actualValue != "", "Encoded HMAC signature should be present")
}

func TestGenerateAcceptsANonce(t *testing.T) {
	signature := Build("api_key", "api_secret", "nonce", date())
	actualValue := signature[86:116]
	expected := "9V8gi5Mp9MsL%2FO7mV6qZlBM9%2FR"
	assert.Equal(t, expected, actualValue, "HMAC signature must contain the signature")
}

func date() string {
	now, _ := time.Parse(time.RFC1123, "Mon, 02 Jan 2020 15:04:05 GMT")
	return now.String()
}
