package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"livepoll-backend/database"
	"livepoll-backend/models"
	"livepoll-backend/realtime"
	"livepoll-backend/services"
)

type voteRequest struct {
	Option int `json:"option" binding:"required"`
}

// Vote is the realtime heart of the app:
//  1. Validate the option against MongoDB (never trust the client).
//  2. HINCRBY the option's count in Redis (atomic, single source of truth).
//  3. Optional per-voter dedupe using the voter token in Redis.
//  4. Write the vote to MongoDB for durability.
//  5. Build an event and publish it to the poll's Redis channel. The hub's
//     wildcard subscription picks it up and pushes it to every browser.
func Vote(hub *realtime.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		pollID := c.Param("id")
		if !primitive.IsValidObjectID(pollID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid poll id"})
			return
		}

		var req voteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "option index is required"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		pollOID, _ := primitive.ObjectIDFromHex(pollID)

		// Check the poll exists and the option is in range. Also verify the
		// poll hasn't expired if it was given an expiry.
		poll, _, _, err := services.SyncPollCountsFromRedis(ctx, pollID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "poll not found"})
			return
		}
		if req.Option < 0 || req.Option >= len(poll.Options) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid option"})
			return
		}
		if poll.ExpiresAt != nil && time.Now().After(*poll.ExpiresAt) {
			c.JSON(http.StatusForbidden, gin.H{"error": "poll has closed"})
			return
		}

		// Per-voter dedupe: one person = one vote. We use a token from the
		// client (a persistent cookie on the voter's browser). Not bulletproof
		// against an attacker, but stops accidental double-votes by real users.
		voterToken := voterID(c)
		if voterToken != "" {
			added, err := database.RDB.SAdd(ctx, database.PollVotersKey(pollID), voterToken).Result()
			if err == nil && added == 0 {
				c.JSON(http.StatusConflict, gin.H{"error": "you have already voted on this poll"})
				return
			}
		}

		// Atomic increment in Redis.
		if err := database.RDB.HIncrBy(ctx, database.PollCountsKey(pollID), strconv.Itoa(req.Option), 1).Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not record vote"})
			return
		}

		// Durability: mirror the vote into MongoDB.
		vote := models.Vote{
			ID:        primitive.NewObjectID(),
			PollID:    pollOID,
			OptionIdx: req.Option,
			VoterID:   voterToken,
			CreatedAt: time.Now().UTC(),
		}
		_, _ = database.VotesCol.InsertOne(ctx, vote)

		// Build the fresh snapshot and broadcast it. Read the counts right
		// back from Redis so everyone (including this voter) gets live data.
		poll, counts, total, err := services.SyncPollCountsFromRedis(ctx, pollID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load counts"})
			return
		}

		hub.PublishEvent(ctx, pollID, realtime.Event{
			Type:       "vote",
			PollID:     pollID,
			Counts:     counts,
			TotalVotes: total,
			Percent:    services.Percentages(counts, total),
		})

		c.JSON(http.StatusOK, gin.H{
			"counts":    counts,
			"total":     total,
			"percent":   services.Percentages(counts, total),
			"question":  poll.Question,
		})
	}
}

// voterID derives a stable anonymous fingerprint from the voter's request.
// Without cookies/JS this is a good-enough token for the demo; the same
// browser tab gets the same token so refresh/re-vote is blocked.
func voterID(c *gin.Context) string {
	var raw string
	if c.GetHeader("X-Voter-Id") != "" {
		raw = c.GetHeader("X-Voter-Id")
	} else {
		// Fall back to the client IP address.
		raw = c.ClientIP()
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}