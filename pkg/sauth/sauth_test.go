package sauth

import (
	"context"
	"crypto/rsa"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func genKeyPair() (*rsa.PrivateKey, error) {
	pk, err := rsa.GenerateKey(
		rand.New(rand.NewSource(time.Now().UnixNano())),
		1024)
	return pk, err
}

func TestAuth__properTTL(t *testing.T) {
	so := assert.New(t)

	userID := rand.Int63()
	ctx := context.Background()

	pk, err := genKeyPair()
	so.Nil(err)

	cfg := Config{
		TokenAccessTTL:  time.Second * 2,
		TokenRefreshTTL: time.Second * 5,
		SignKey:         pk,
		ValidateKey:     &pk.PublicKey,
	}

	author := New(cfg)

	rsp, err := author.GenerateTokens(ctx, userID)
	so.Nil(err)
	so.NotNil(rsp)

	so.True(rsp.AccessExpiresAt.After(rsp.IssuedAt))

	so.NotEqual("", rsp.Pair.Refresh, "tokens shouldn't be empty")
	so.NotEqual("", rsp.Pair.Access)

	gotUser, err := author.Check(ctx, rsp.Pair.Access)
	so.Nil(err, "access token should extract user successfully")
	so.Equal(userID, gotUser)

	_, err = author.Check(ctx, rsp.Pair.Refresh)
	so.NotNil(err, "refresh token should error out .Check() (only access tokens should work)")

	prevPair := *rsp
	rsp, err = author.Refresh(ctx, rsp.Pair, userID)
	so.Nil(err, "refresh should work successfully")
	so.Equal(prevPair.Pair.Refresh, rsp.Pair.Refresh, "shouldn't recreate new refresh token")
	so.NotEqual(prevPair.Pair.Access, rsp.Pair.Access, "should recreate any access token")

	<-time.After(time.Second * 3)

	_, err = author.Check(context.Background(), rsp.Pair.Access)
	so.NotNil(err, "refresh token should expire by now")

	prevPair = *rsp
	rsp, err = author.Refresh(context.Background(), rsp.Pair, userID)
	so.Nil(err)
	so.NotEqual(prevPair.Pair.Refresh, rsp.Pair.Refresh, "should recreate refresh token close to expiration")
	so.NotEqual(prevPair.Pair.Access, rsp.Pair.Access, "should recreate any access token")
}
