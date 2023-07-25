package auth

import (
	"context"
	"net/http"

	"github.com/pechorka/adhd-reader/internal/handler/herror"
)

type AuthService interface {
	ParseToken(token string) (int64, error)
}

type AuthMW struct {
	svc AuthService
}

var ctxKeyUser struct{}
var NotFoundUserID = int64(-1)

func NewAuthMW(svc AuthService) *AuthMW {
	return &AuthMW{svc: svc}
}

const basicPrefix = "Basic "

func (mw *AuthMW) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || len(authHeader) < len(basicPrefix) {
			herror.RespondErrorWithCode(w,
				http.StatusUnauthorized,
				herror.CODE_AUTH_HEADER_MISSING,
			)
			return
		}
		token := authHeader[len(basicPrefix):]
		userID, err := mw.svc.ParseToken(token)
		if err != nil {
			herror.RespondErrorWithCode(w,
				http.StatusUnauthorized,
				herror.CODE_AUTH_TOKEN_INVALID,
			)
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyUser, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(ctx context.Context) int64 {
	userID, ok := ctx.Value(ctxKeyUser).(int64)
	if !ok {
		return NotFoundUserID
	}
	return userID
}
