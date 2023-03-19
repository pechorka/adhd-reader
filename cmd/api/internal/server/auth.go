package server

import (
	"context"
	"time"

	"github.com/pechorka/adhd-reader/generated/api"
	"github.com/pechorka/adhd-reader/pkg/jwt"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/datetime"
	"google.golang.org/protobuf/types/known/durationpb"
)

type JwtService interface {
	GenerateTokens(userID int64) (*jwt.TokenResponse, error)
	Refresh(refreshToken string) (*jwt.TokenResponse, error)
}

type AuthService interface {
	VerifyPassword(password string) (int64, error)
}

type Auth struct {
	jwtService  JwtService
	authService AuthService
	api.UnimplementedAuthServiceServer
}

var _ api.AuthServiceServer = (*Auth)(nil)

func NewAuth(jwtService JwtService, authService AuthService) *Auth {
	return &Auth{
		jwtService:  jwtService,
		authService: authService,
	}
}

func (s *Auth) Auth(ctx context.Context, req *api.AuthRequest) (*api.AuthResponse, error) {
	userID, err := s.authService.VerifyPassword(req.Password)
	if err != nil {
		return nil, errors.Wrap(err, "verifying password")
	}
	tokenResponse, err := s.jwtService.GenerateTokens(userID)
	if err != nil {
		return nil, errors.Wrap(err, "generating tokens")
	}
	return &api.AuthResponse{
		TokenInfo: mapTokenResponse(tokenResponse),
	}, nil
}

func (s *Auth) Refresh(ctx context.Context, req *api.RefreshRequest) (*api.RefreshResponse, error) {
	tokenResponse, err := s.jwtService.Refresh(req.RefreshToken)
	if err != nil {
		return nil, errors.Wrap(err, "refreshing tokens")
	}
	return &api.RefreshResponse{
		TokenInfo: mapTokenResponse(tokenResponse),
	}, nil
}

func mapTokenResponse(tokenResponse *jwt.TokenResponse) *api.TokenInfo {
	return &api.TokenInfo{
		TokenPair: &api.TokenPair{
			AccessToken:  tokenResponse.Pair.Access,
			RefreshToken: tokenResponse.Pair.Refresh,
		},
		AccessExpiresAt: golantTimeToGrpcTime(tokenResponse.AccessExpiresAt),
		IssuedAt:        golantTimeToGrpcTime(tokenResponse.IssuedAt),
	}
}

func golantTimeToGrpcTime(t time.Time) *datetime.DateTime {
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	return &datetime.DateTime{
		Year:    int32(year),
		Month:   int32(month),
		Day:     int32(day),
		Hours:   int32(hour),
		Minutes: int32(min),
		Seconds: int32(sec),
		Nanos:   int32(t.Nanosecond()),
		TimeOffset: &datetime.DateTime_UtcOffset{
			UtcOffset: durationpb.New(t.UTC().Sub(t)),
		},
	}
}
