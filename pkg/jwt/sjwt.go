package jwt

import (
	"crypto/rsa"
	"time"

	"github.com/pkg/errors"
)

type Service struct {
	c Config
}

type Config struct {
	TokenAccessTTL  time.Duration
	TokenRefreshTTL time.Duration

	SignKey     *rsa.PrivateKey
	ValidateKey *rsa.PublicKey
}

func New(cfg Config) *Service {
	return &Service{
		c: cfg,
	}
}

// GenerateTokens creates tokens for a user.
func (s *Service) GenerateTokens(userID int64) (*TokenResponse, error) {
	tokAccess, expiresAt, err := genTokenAccess(userID, s.c.TokenAccessTTL, s.c.SignKey)
	if err != nil {
		return nil, errors.Wrap(err, "signing access token")
	}

	tokRefresh, err := genTokenRefresh(userID, s.c.TokenRefreshTTL, s.c.SignKey)
	if err != nil {
		return nil, errors.Wrap(err, "signing refresh token")
	}

	return &TokenResponse{
		UserID: userID,
		Pair: TokenPair{
			Access:  tokAccess,
			Refresh: tokRefresh,
		},
		AccessExpiresAt: expiresAt,
		IssuedAt:        time.Now(),
	}, nil
}

func (s *Service) extractValidateRefreshTok(str string) (*jwtClaims, error) {
	// TODO validate that user has this AccessToken as last used accessTok
	//
	// If this refreshToken is valid, but user already refreshed this accessToken
	// then it means that this refreshToken was stolen OR app crashed and didn't save
	// a new token.
	//
	// Maybe we should allow those old accToks if user has the same IP as on the last refresh?

	claims, err := jwtValidate(str, s.c.ValidateKey)
	if err != nil {
		return nil, errors.Wrap(err, "bad refresh token")
	}
	if !claims.IsRefreshToken {
		return nil, errors.Wrap(err, "access token used as a refresh")
	}

	return claims, nil
}

func (s *Service) Refresh(refreshToken string) (*TokenResponse, error) {
	claims, err := s.extractValidateRefreshTok(refreshToken)
	if err != nil {
		return nil, errors.Wrap(err, "invalid refresh token")
	}

	tokAccess, expAt, err := genTokenAccess(claims.UserID, s.c.TokenAccessTTL, s.c.SignKey)
	if err != nil {
		return nil, errors.Wrap(err, "signing access token")
	}

	ret := TokenResponse{
		UserID: claims.UserID,
		Pair: TokenPair{
			Access:  tokAccess,
			Refresh: refreshToken,
		},
		AccessExpiresAt: expAt,
		IssuedAt:        time.Now(),
	}

	// if refresh token has more than half of its life span left then return it as-is
	secUntilExpiration := claims.ExpiresAt - time.Now().Unix()
	if secUntilExpiration > (int64(s.c.TokenRefreshTTL.Seconds()) / 2) { // nolint: gomnd
		return &ret, nil
	}

	ret.Pair.Refresh, err = genTokenRefresh(claims.UserID, s.c.TokenRefreshTTL, s.c.SignKey)
	return &ret, errors.Wrap(err, "signing refresh token")
}

func (s *Service) Check(accessToken string) (int64, error) {
	cl, err := jwtValidate(accessToken, s.c.ValidateKey)
	if err != nil {
		return 0, err
	}
	if cl.IsRefreshToken {
		return 0, errors.New("Check got refresh token instead of access token")
	}

	return cl.UserID, nil
}
