package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"livepoll-backend/realtime"
	"livepoll-backend/services"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Live viewers connect from the frontend origin. Policies are managed
	// at the network layer in production; this enables the dev setup.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WebSocket connects a viewer to the live results stream for a single poll.
// The first message the browser receives is the current snapshot; after that
// it's push-only from Redis via the hub.
func WebSocket(hub *realtime.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		pollID := c.Query("poll")
		if !primitive.IsValidObjectID(pollID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "poll id is required"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("websocket upgrade: %v", err)
			return
		}

		client := realtime.NewClient(hub, pollID, conn)
		go client.ReadPump()
		go client.WritePump()

		// Immediately send the current live snapshot so the client can render
		// instantly instead of waiting for the next vote.
		ctx := c.Request.Context()
		poll, counts, total, err := services.SyncPollCountsFromRedis(ctx, pollID)
		if err == nil {
			_ = conn.WriteJSON(map[string]interface{}{
				"type":       "snapshot",
				"pollId":     pollID,
				"counts":     counts,
				"totalVotes": total,
				"percent":    services.Percentages(counts, total),
				"question":   poll.Question,
			})
		}
	}
}