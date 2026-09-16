package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

func ConnectRedis(addr, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}

	RDB = client
	log.Println("connected to redis")
	return nil
}

// Key helpers. Redis is the live source of truth for vote counts.

func PollCountsKey(pollID string) string { return "poll:" + pollID + ":counts" }

func PollVotersKey(pollID string) string { return "poll:" + pollID + ":voters" }

func PollChannel(pollID string) string { return "poll:" + pollID + ":events" }