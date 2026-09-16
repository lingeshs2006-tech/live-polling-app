package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Username     string               `bson:"username" json:"username"`
	PasswordHash string               `bson:"password_hash" json:"-"`
	CreatedAt    time.Time            `bson:"created_at" json:"createdAt"`
	PollIDs      []primitive.ObjectID `bson:"poll_ids" json:"pollIds"`
}

type PollOption struct {
	Text  string `bson:"text" json:"text"`
	Votes int64  `bson:"votes" json:"votes"`
}

type Poll struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CreatedBy   primitive.ObjectID `bson:"created_by" json:"createdBy"`
	Question    string             `bson:"question" json:"question"`
	Options     []PollOption       `bson:"options" json:"options"`
	TotalVotes  int64              `bson:"total_votes" json:"totalVotes"`
	CreatedAt   time.Time          `bson:"created_at" json:"createdAt"`
	ExpiresAt   *time.Time         `bson:"expires_at" json:"expiresAt,omitempty"`
	VoterTokens []string           `bson:"voter_tokens" json:"-"`
}

type Vote struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PollID    primitive.ObjectID `bson:"poll_id" json:"pollId"`
	OptionIdx int                `bson:"option_idx" json:"optionIdx"`
	VoterID   string             `bson:"voter_id" json:"voterId"`
	CreatedAt time.Time          `bson:"created_at" json:"createdAt"`
}