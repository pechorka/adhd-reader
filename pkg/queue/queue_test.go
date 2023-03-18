package queue

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMessageQueue_collectItemsToSend(t *testing.T) {
	queue := NewMessageQueue(Config{})
	const (
		expireUser1   = 1
		expireUser2   = 2
		notExpireUser = 3
	)
	now := time.Now()
	var (
		expireUser1Item = &Item{
			Texts:    []string{"text1", "text2"},
			ExpireAt: now.Add(time.Second),
		}
		expireUser2Item = &Item{
			Texts:    []string{"text3", "text4"},
			ExpireAt: now.Add(2 * time.Second),
		}
	)
	queue.tempStorage = map[int64]*Item{
		expireUser1: expireUser1Item,
		expireUser2: expireUser2Item,
		notExpireUser: {
			Texts:    []string{"text5", "text6"},
			ExpireAt: now.Add(3 * time.Second),
		},
	}

	itemsToSend := queue.collectItemsToSend(now.Add(2 * time.Second))
	require.Len(t, itemsToSend, 2)
	require.Equal(t, expireUser1Item, itemsToSend[expireUser1])
	require.Equal(t, expireUser2Item, itemsToSend[expireUser2])
	require.NotContains(t, itemsToSend, notExpireUser)
}

func Test_sendItems(t *testing.T) {
	const (
		user1 = 1
		user2 = 2
	)
	var (
		user1Item = &Item{
			Texts: []string{"text1", "text2"},
		}
		user2Item = &Item{
			Texts: []string{"text3", "text4"},
		}
	)
	itemsToSend := map[int64]*Item{
		user1: user1Item,
		user2: user2Item,
	}
	expected := map[int64]string{
		user1: "text1text2",
		user2: "text3text4",
	}
	sendFunc := func(userID int64, text string) {
		require.Equal(t, expected[userID], text)
	}
	sendItems(itemsToSend, sendFunc)
}
