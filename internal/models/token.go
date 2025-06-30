package models

import "time"

type Token struct {
	Token     string
	ExpiresAt time.Time
}
