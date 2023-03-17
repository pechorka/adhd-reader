package queue

import (
	"bytes"
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

	inQueueTtl time.Duration // how long to keep message in queue
	ttlCheck   time.Duration // how often to check for expired messages
}

type Config struct {
	InQueueTtl time.Duration
	TtlCheck   time.Duration
}

func NewMessageQueue(cfg Config) *MessageQueue {
	if cfg.InQueueTtl == 0 {
		cfg.InQueueTtl = 2 * time.Second
	}
	if cfg.TtlCheck == 0 {
		cfg.TtlCheck = 1 * time.Second
	}
	return &MessageQueue{
		mu:          &sync.RWMutex{},
		tempStorage: make(map[int64]*Item),
		stopCh:      make(chan struct{}, 1),
		inQueueTtl:  cfg.InQueueTtl,
		ttlCheck:    cfg.TtlCheck,
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
	item.ExpireAt = time.Now().Add(q.inQueueTtl)
	q.tempStorage[userID] = item
}

func (q *MessageQueue) Stop() {
	q.stopCh <- struct{}{}
}

func (q *MessageQueue) Run(onTextReady func(userID int64, text string)) {
	go func() {
		ticker := time.NewTicker(q.ttlCheck)
		for {
			select {
			case <-q.stopCh:
				return
			case now := <-ticker.C:
				itemsToSend := q.collectItemsToSend(now)
				sendItems(itemsToSend, onTextReady)
			}
		}
	}()
}

func (q *MessageQueue) collectItemsToSend(now time.Time) map[int64]*Item {
	itemsToSend := make(map[int64]*Item, 100)
	q.mu.RLock()
	defer q.mu.RUnlock()
	for userID, item := range q.tempStorage {
		if before(now, item.ExpireAt) {
			delete(q.tempStorage, userID)
			itemsToSend[userID] = item
		}
	}
	return itemsToSend
}

func before(now, expireAt time.Time) bool {
	return expireAt.Before(now) || expireAt.Equal(now)
}

const maxTextLength = 4096

func sendItems(itemsToSend map[int64]*Item, sendFunc func(userID int64, text string)) {
	var maxTextChunkCount int
	for _, item := range itemsToSend {
		if len(item.Texts) > maxTextChunkCount {
			maxTextChunkCount = len(item.Texts)
		}
	}
	builder := bytes.NewBuffer(nil)
	builder.Grow(maxTextChunkCount * maxTextLength)
	for userID, item := range itemsToSend {
		for _, text := range item.Texts {
			builder.WriteString(text)
		}
		sendFunc(userID, builder.String())
		builder.Reset()
	}
}
