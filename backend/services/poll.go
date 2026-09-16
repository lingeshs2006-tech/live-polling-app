package services

import (
	"context"
	"log"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"livepoll-backend/database"
	"livepoll-backend/models"
)

// SyncPollCountsFromRedis reads the live vote counts out of Redis and
// persists them back to MongoDB. Called after a vote so the durable store
// stays in step with the realtime counter. It returns the snapshot that is
// also used to build the broadcast event.
func SyncPollCountsFromRedis(ctx context.Context, pollID string) (p models.Poll, counts []int64, total int64, err error) {
	pollOID, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		return p, nil, 0, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := database.PollsCol.FindOne(ctx, bson.M{"_id": pollOID}).Decode(&p); err != nil {
		return p, nil, 0, err
	}

	exists, err := database.RDB.Exists(ctx, database.PollCountsKey(pollID)).Result()
	if err != nil {
		return p, nil, 0, err
	}

	// Redis is the realtime source of truth. If it doesn't have the hash yet
	// (fresh restart, or created by another instance before Redis warmed up),
	// hydrate it from MongoDB so Redis is back in charge.
	if exists == 0 {
		if err := hydrateRedisFromMongo(ctx, &p); err != nil {
			return p, nil, 0, err
		}
	}

	vals, err := database.RDB.HGetAll(ctx, database.PollCountsKey(pollID)).Result()
	if err != nil {
		return p, nil, 0, err
	}

	counts = make([]int64, len(p.Options))
	total = 0
	for i := range p.Options {
		if raw, ok := vals[strconv.Itoa(i)]; ok {
			n, _ := strconv.ParseInt(raw, 10, 64)
			counts[i] = n
			total += n
		}
	}

	// Durable write.
	filter := bson.M{"_id": pollOID}
	update := bson.M{
		"$set": bson.M{
			"options":     buildOptionsWithCounts(p.Options, counts),
			"total_votes": total,
		},
	}
	if _, err := database.PollsCol.UpdateOne(ctx, filter, update); err != nil {
		log.Printf("mongo update counts: %v", err)
	}
	p.TotalVotes = total
	return p, counts, total, nil
}

func hydrateRedisFromMongo(ctx context.Context, p *models.Poll) error {
	key := database.PollCountsKey(p.ID.Hex())
	pairs := make([]interface{}, 0, len(p.Options)*2)
	for i, opt := range p.Options {
		pairs = append(pairs, strconv.Itoa(i), opt.Votes)
	}
	return database.RDB.HSet(ctx, key, pairs...).Err()
}

func buildOptionsWithCounts(options []models.PollOption, counts []int64) []models.PollOption {
	out := make([]models.PollOption, len(options))
	for i, opt := range options {
		out[i] = models.PollOption{Text: opt.Text, Votes: counts[i]}
	}
	return out
}

// Percentages returns the share each option has of the total, rounded to one
// decimal place, with a total of 0 handled safely.
func Percentages(counts []int64, total int64) []float64 {
	percent := make([]float64, len(counts))
	if total == 0 {
		return percent
	}
	for i, c := range counts {
		percent[i] = float64(c) / float64(total) * 100
	}
	return percent
}