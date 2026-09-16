package realtime

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
)

// Event is the payload pushed to every browser watching a poll.
type Event struct {
	Type       string `json:"type"` // "vote" | "poll-deleted"
	PollID     string `json:"pollId"`
	Counts     []int64  `json:"counts,omitempty"`
	TotalVotes int64    `json:"totalVotes,omitempty"`
	Percent    []float64 `json:"percent,omitempty"`
}

// Hub keeps a map of pollID -> connected clients and broadcasts events
// to every client subscribed to that poll.
type Hub struct {
	mu    sync.RWMutex
	polls map[string]map[*Client]bool
	redis *redis.Client
	ctx   context.Context
}

func NewHub(rdb *redis.Client, ctx context.Context) *Hub {
	return &Hub{
		polls: make(map[string]map[*Client]bool),
		redis: rdb,
		ctx:   ctx,
	}
}

func (h *Hub) Subscribe(pollID string, c *Client) {
	h.mu.Lock()
	if h.polls[pollID] == nil {
		h.polls[pollID] = make(map[*Client]bool)
	}
	h.polls[pollID][c] = true
	h.mu.Unlock()
}

func (h *Hub) Unsubscribe(pollID string, c *Client) {
	h.mu.Lock()
	if clients, ok := h.polls[pollID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.polls, pollID)
		}
	}
	h.mu.Unlock()
}

func (h *Hub) broadcast(pollID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.polls[pollID] {
		select {
		case c.send <- msg:
		default: // slow consumer, drop rather than block the hub
		}
	}
}

// StartRedisListener subscribes to every poll event channel using the
// "poll:*" wildcard and forwards raw payloads to the right hub subscribers.
// This is what makes Redis the live engine: votes are written to Redis and
// every client is updated entirely from these messages.
func (h *Hub) StartRedisListener() {
	sub := h.redis.Subscribe(h.ctx, "poll:*")
	ch := sub.Channel()

	go func() {
		for msg := range ch {
			pollID := pollIDFromChannel(msg.Channel)
			if pollID == "" {
				continue
			}
			h.broadcast(pollID, []byte(msg.Payload))
		}
		log.Println("redis listener stopped")
	}()
}

func pollIDFromChannel(channel string) string {
	const prefix = "poll:"
	const suffix = ":events"
	if len(channel) <= len(prefix)+len(suffix) {
		return ""
	}
	if !strings.HasPrefix(channel, prefix) || !strings.HasSuffix(channel, suffix) {
		return ""
	}
	return channel[len(prefix) : len(channel)-len(suffix)]
}

// PublishEvent writes a JSON event to a poll's Redis channel. The hub's own
// Redis listener receives it on the wildcard subscription and fans it out
// to the connected browsers. Every backend instance can publish and every
// instance listens, so this scales to multiple servers.
func (h *Hub) PublishEvent(ctx context.Context, pollID string, ev Event) {
	data, err := json.Marshal(ev)
	if err != nil {
		log.Printf("marshal event: %v", err)
		return
	}
	if err := h.redis.Publish(ctx, "poll:"+pollID+":events", data).Err(); err != nil {
		log.Printf("redis publish: %v", err)
	}
}