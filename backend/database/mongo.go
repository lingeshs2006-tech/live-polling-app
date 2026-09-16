package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	DB       *mongo.Database
	UsersCol *mongo.Collection
	PollsCol *mongo.Collection
	VotesCol *mongo.Collection
)

func ConnectMongo(uri, dbName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("mongo connect: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("mongo ping: %w", err)
	}

	DB = client.Database(dbName)
	UsersCol = DB.Collection("users")
	PollsCol = DB.Collection("polls")
	VotesCol = DB.Collection("votes")

	log.Println("connected to mongo")
	return nil
}