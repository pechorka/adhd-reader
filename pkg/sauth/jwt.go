package sauth

import (
	"crypto/rsa"
	"math/rand"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pechorka/adhd-reader/pkg/saferand"
	"github.com/pkg/errors"
)

const (
	jwtIssuer = "beyond100-auth"
)

var randSrc = rand.New(saferand.NewSource(time.Now().UnixNano()))

func genTokenAccess(userID int64, ttlAccess time.Duration, key *rsa.PrivateKey) (string, time.Time, error) {
	claim := jwtClaims{
		UserID: userID,
	}
	expiresAt := time.Now().Add(ttlAccess)
	tok, err := jwtSign(claim, expiresAt, key)
	return tok, expiresAt, err
}

func genTokenRefresh(userID int64, ttlRefresh time.Duration, key *rsa.PrivateKey) (string, error) {
	claim := jwtClaims{
		UserID:         userID,
		IsRefreshToken: true,
	}
	expiresAt := time.Now().Add(ttlRefresh)
	tok, err := jwtSign(claim, expiresAt, key)
	return tok, err
}

func jwtSign(claims jwtClaims, expAt time.Time, key *rsa.PrivateKey) (string, error) {
	claims.StandardClaims.ExpiresAt = expAt.Unix()
	claims.StandardClaims.IssuedAt = time.Now().Unix()
	claims.StandardClaims.Issuer = jwtIssuer
	claims.Rand = randSrc.Int63()

	// RS512 -> public/private key pair
	tok := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	return tok.SignedString(key)
}

func jwtExtract(tokStr string, key *rsa.PublicKey) (*jwtClaims, error) {
	p := &jwt.Parser{SkipClaimsValidation: true}
	token, err := p.ParseWithClaims(tokStr, &jwtClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodRS512.Alg() {
				return nil, errors.Errorf("bad alg '%v'", token.Method.Alg())
			}
			return key, nil
		})
	if err != nil {
		return nil, errors.Wrap(err, "validation failed")
	}

	claims := token.Claims.(*jwtClaims)
	if !claims.VerifyIssuer(jwtIssuer, true) {
		return nil, errors.Errorf("unknown issuer '%v'", claims.Issuer)
	}
	if claims.UserID == 0 {
		return nil, errors.New("user ID is empty")
	}
	return claims, nil
}

func jwtValidate(tokStr string, key *rsa.PublicKey) (*jwtClaims, error) {
	claims, err := jwtExtract(tokStr, key)
	if err != nil {
		return nil, errors.Wrap(err, "cannot extract JWT token")
	}
	now := time.Now()
	if !claims.VerifyIssuedAt(now.Unix(), true) {
		return nil, errors.New("token issued in the future?")
	}
	if !claims.VerifyExpiresAt(now.Unix(), true) {
		return nil, errors.Errorf("token is expired, user ID: %d", claims.UserID)
	}
	if err = claims.Valid(); err != nil {
		return nil, errors.Wrap(err, "token claims are invalid")
	}
	return claims, nil
}
