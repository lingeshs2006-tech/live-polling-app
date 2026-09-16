package realtime

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 45 * time.Second
	maxMsgSize = 512
)

// Client wraps one websocket connection belonging to a poll viewer.
type Client struct {
	pollID    string
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	closeOnce sync.Once
}

func NewClient(hub *Hub, pollID string, conn *websocket.Conn) *Client {
	c := &Client{
		pollID: pollID,
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
	}
	hub.Subscribe(pollID, c)
	return c
}

// ReadPump discards incoming messages but keeps the connection alive by
// handling pong responses from the browser side.
func (c *Client) ReadPump() {
	defer c.Close()
	c.conn.SetReadLimit(maxMsgSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

// WritePump pushes broadcast messages and periodic pings down the socket.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.hub.Unsubscribe(c.pollID, c)
		_ = c.conn.Close()
		close(c.send)
		log.Println("client disconnected")
	})
}
