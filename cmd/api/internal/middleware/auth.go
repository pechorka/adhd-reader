package middleware

import (
	"context"

	"github.com/pechorka/adhd-reader/pkg/jwt"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var userIDKey struct{}

func Auth(jwtService *jwt.Service) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return errors.New("no authorisation header")
		}
		accessToken := md.Get("authorisation")
		if len(accessToken) == 0 {
			return errors.New("no authorisation header")
		}

		userID, err := jwtService.Check(accessToken[0])
		if err != nil {
			return errors.Wrap(err, "invalid access token")
		}

		ctx = context.WithValue(ctx, userIDKey, userID)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func UserID(ctx context.Context) int64 {
	return ctx.Value(userIDKey).(int64)
}
