package jwt

import "time"

type User struct {
}

type TokenResponse struct {
	UserID int64

	Pair TokenPair
	// IssuedAt is the time when this Pair was generated.
	// Token that has IssuedAt > now() is invalid.
	IssuedAt time.Time
	// AccessExpiresAt is the time when Pair's Access token will become invalid.
	AccessExpiresAt time.Time
}

type TokenPair struct {
	Access  string
	Refresh string
}
