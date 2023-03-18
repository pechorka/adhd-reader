package queue

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestQueue(t *testing.T) {
	const inQueueTime = 10 * time.Millisecond
	q := NewMessageQueue(Config{
		InQueueTtl: inQueueTime,
		TtlCheck:   time.Millisecond,
	})
	expectCall := &atomic.Bool{}
	expectCall.Store(true)
	expectTextForUser := &sync.Map{}
	onTextFunc := func(userID int64, text string) {
		if !expectCall.Load() {
			t.Fatal("unexpected call to onTextFunc with args", userID, text)
		}
		// expectText only once, so delete it
		expectedText, ok := expectTextForUser.LoadAndDelete(userID)
		require.True(t, ok, "expected text for user %d not found", userID)
		require.Equal(t, expectedText.(string), text, "expected text for user %d not equal to actual: want %s, got %s", userID, expectedText, text)
	}
	q.Run(onTextFunc)
	userID := int64(1)
	q.Add(userID, "text1")
	q.Add(userID, "text2")
	q.Add(userID, "text3")
	expectTextForUser.Store(userID, "text1text2text3")
	time.Sleep(inQueueTime + time.Millisecond) // add +1ms to be sure that queue will be checked
	q.Stop()
	expectCall.Store(false)
	lateUser := int64(2)
	q.Add(lateUser, "late text")
	time.Sleep(inQueueTime + time.Millisecond) // expect no call to onTextFunc
}
