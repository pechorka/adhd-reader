package queue

import (
	"strings"
	"sync"
	"time"
)

type Item struct {
	Texts    []string
	ExpireAt time.Time
}

type MessageQueue struct {
	mu          *sync.RWMutex
	tempStorage map[int64]*Item
	stopCh      chan struct{}
}

const ttl = 2 * time.Second

func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		mu:          &sync.RWMutex{},
		tempStorage: make(map[int64]*Item),
		stopCh:      make(chan struct{}, 1),
	}
}

func (q *MessageQueue) Add(userID int64, text string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	item, ok := q.tempStorage[userID]
	if !ok {
		item = &Item{}
	}
	item.Texts = append(item.Texts, text)
	item.ExpireAt = time.Now().Add(ttl)
	q.tempStorage[userID] = item
}

func (q *MessageQueue) Stop() {
	q.stopCh <- struct{}{}
}

func (q *MessageQueue) Run(onTextReady func(userID int64, text string)) {
	go func() {
		ticker := time.NewTicker(ttl)
		for {
			select {
			case <-q.stopCh:
				return
			case now := <-ticker.C:
				itemsToSend := make(map[int64]*Item, 100) // TODO: make it configurable
				q.mu.RLock()
				for userID, item := range q.tempStorage {
					if item.ExpireAt.Before(now) {
						delete(q.tempStorage, userID)
						itemsToSend[userID] = item
					}
				}
				q.mu.RUnlock()
				for userID, item := range itemsToSend {
					totalLength := 0
					for _, text := range item.Texts {
						totalLength += len(text)
					}
					builder := &strings.Builder{}
					builder.Grow(totalLength)
					for _, text := range item.Texts {
						builder.WriteString(text)
					}
					onTextReady(userID, builder.String())
				}
			}
		}
	}()
}
