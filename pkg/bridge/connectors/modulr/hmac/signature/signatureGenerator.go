package signature

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/url"
)

const (
	Algorithm       = "algorithm=\"hmac-sha1\","
	DatePrefix      = "date: "
	Headers         = "headers=\"date x-mod-nonce\","
	SignaturePrefix = "signature=\""
	SignatureSuffix = "\""
	Newline         = "\n"
	Nonce           = "x-mod-nonce: "
	KeyIdPrefix     = "Signature keyId=\""
)

func Build(apiKey string, apiSecret string, nonce string, date string) string {
	return buildKeyId(apiKey) + Algorithm + Headers + generateEncodedSignature(apiSecret, nonce, date)
}

func buildKeyId(apiKey string) string {
	return KeyIdPrefix + apiKey + "\","
}

func generateEncodedSignature(apiSecret string, nonce string, date string) string {
	encodedSig := encodeSignature(DatePrefix + date + Newline + Nonce + nonce, apiSecret)
	return SignaturePrefix + encodedSig + SignatureSuffix
}

func encodeSignature(plainSignature string, apiSecret string) string {
	mac := hmac.New(sha1.New, []byte(apiSecret))
	mac.Write([]byte(plainSignature))
	encodedMac := mac.Sum(nil)
	base64Encoded := base64.StdEncoding.EncodeToString(encodedMac)
	return url.QueryEscape(base64Encoded)
}
