package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type TradeID string

func (p TradeID) String() string {
	return string(p)
}

func (p TradeID) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, p.String())), nil
}

func TradeIDFromString(s string) (TradeID, error) {
	if s == "" {
		return "", fmt.Errorf("trade id cannot be empty")
	}

	return TradeID(s), nil
}

func MustTradeIDFromString(s string) TradeID {
	id, err := TradeIDFromString(s)
	if err != nil {
		panic(err)
	}

	return id
}

func TradeID_FromReference(reference string, connectorID ConnectorID) TradeID {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s-%s", connectorID.String(), reference)))

	return TradeID(hex.EncodeToString(h.Sum(nil)))
}

