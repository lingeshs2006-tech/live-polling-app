package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"livepoll-backend/database"
	"livepoll-backend/middleware"
	"livepoll-backend/models"
	"livepoll-backend/realtime"
	"livepoll-backend/services"
)

type createPollRequest struct {
	Question string   `json:"question" binding:"required"`
	Options  []string `json:"options" binding:"required"`
}

// CreatePoll validates input server-side, stores it in MongoDB and seeds
// the zero-count hash in Redis so live votes have a realtime home.
func CreatePoll(hub *realtime.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createPollRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "question and options are required"})
			return
		}

		question := trimSpaces(req.Question)
		options := cleanOptions(req.Options)
		if question == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "question cannot be empty"})
			return
		}
		if len(options) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "a poll needs at least 2 options"})
			return
		}
		if len(options) > 10 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "a poll can have at most 10 options"})
			return
		}

		userID := middleware.UserID(c)

		pollOptions := make([]models.PollOption, len(options))
		for i, text := range options {
			pollOptions[i] = models.PollOption{Text: text, Votes: 0}
		}

		poll := models.Poll{
			ID:         primitive.NewObjectID(),
			CreatedBy:  userID,
			Question:   question,
			Options:    pollOptions,
			TotalVotes: 0,
			CreatedAt:  time.Now().UTC(),
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		if _, err := database.PollsCol.InsertOne(ctx, poll); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create poll"})
			return
		}

		// Seed Redis counts so live updates have a key from the start.
		key := database.PollCountsKey(poll.ID.Hex())
		pairs := make([]interface{}, 0, len(pollOptions)*2)
		for i := range pollOptions {
			pairs = append(pairs, i, int64(0))
		}
		if err := database.RDB.HSet(ctx, key, pairs...).Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not initialize live poll"})
			return
		}

		// Track the poll on the creator's user document (small denormalization
		// used for "my polls" listing).
		_, _ = database.UsersCol.UpdateOne(ctx,
			bson.M{"_id": userID},
			bson.M{"$addToSet": bson.M{"poll_ids": poll.ID}},
		)

		c.JSON(http.StatusCreated, gin.H{
			"poll":      poll,
			"shareLink": shareLink(c, poll.ID.Hex()),
		})
	}
}

// GetPoll returns the poll, its live counts (from Redis) and the share link.
// Public: anyone with the link can view and vote.
func GetPoll(c *gin.Context) {
	pollID := c.Param("id")
	if !primitive.IsValidObjectID(pollID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid poll id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	poll, counts, total, err := services.SyncPollCountsFromRedis(ctx, pollID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "poll not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load poll"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"poll":      poll,
		"counts":    counts,
		"total":     total,
		"percent":   services.Percentages(counts, total),
		"shareLink": shareLink(c, pollID),
		"isOwner":   middleware.UserID(c) == poll.CreatedBy,
	})
}

// ListMyPolls returns the polls created by the authenticated user.
func ListMyPolls(c *gin.Context) {
	userID := middleware.UserID(c)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var user models.User
	if err := database.UsersCol.FindOne(ctx, bson.M{"_id": userID}).Decode(&user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	cursor, err := database.PollsCol.Find(ctx, bson.M{"_id": bson.M{"$in": user.PollIDs}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load polls"})
		return
	}
	defer cursor.Close(ctx)

	var polls []models.Poll
	if err := cursor.All(ctx, &polls); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load polls"})
		return
	}
	if polls == nil {
		polls = []models.Poll{}
	}

	c.JSON(http.StatusOK, polls)
}

// DeletePoll removes a poll the authenticated user owns, from MongoDB and
// Redis, and broadcasts a "poll-deleted" event so watchers close cleanly.
func DeletePoll(hub *realtime.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		pollID := c.Param("id")
		if !primitive.IsValidObjectID(pollID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid poll id"})
			return
		}

		userID := middleware.UserID(c)
		pollOID, _ := primitive.ObjectIDFromHex(pollID)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		res, err := database.PollsCol.DeleteOne(ctx, bson.M{"_id": pollOID, "created_by": userID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete poll"})
			return
		}
		if res.DeletedCount == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "poll not found or not yours to delete"})
			return
		}

		database.RDB.Del(ctx,
			database.PollCountsKey(pollID),
			database.PollVotersKey(pollID),
		)
		_, _ = database.UsersCol.UpdateOne(ctx,
			bson.M{"_id": userID},
			bson.M{"$pull": bson.M{"poll_ids": pollOID}},
		)

		hub.PublishEvent(ctx, pollID, realtime.Event{Type: "poll-deleted", PollID: pollID})

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func trimSpaces(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func cleanOptions(opts []string) []string {
	seen := make(map[string]bool)
	out := []string{}
	for _, o := range opts {
		o = trimSpaces(o)
		if o == "" || seen[o] {
			continue
		}
		seen[o] = true
		out = append(out, o)
	}
	return out
}

func shareLink(c *gin.Context, pollID string) string {
	if host := c.GetHeader("X-Forwarded-Host"); host != "" {
		return "//" + host + "/poll/" + pollID
	}
	return "/poll/" + pollID
}