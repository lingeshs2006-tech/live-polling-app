package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"livepoll-backend/config"
	"livepoll-backend/database"
	"livepoll-backend/handlers"
	"livepoll-backend/middleware"
	"livepoll-backend/realtime"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using defaults")
	}

	cfg := config.Load()

	ctx := context.Background()

	if err := database.ConnectMongo(cfg.MongoURI, cfg.MongoDatabase); err != nil {
		log.Fatalf("mongo: %v", err)
	}
	if err := database.ConnectRedis(cfg.RedisAddr, cfg.RedisPassword); err != nil {
		log.Fatalf("redis: %v", err)
	}

	middleware.InitAuth(cfg.JWTSecret)

	hub := realtime.NewHub(database.RDB, ctx)
	hub.StartRedisListener()

	if cfg.ClientOrigin == "http://localhost:5173" || cfg.ClientOrigin == "http://localhost" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(middleware.CORS(cfg.ClientOrigin))

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/signup", handlers.SignUp)
			auth.POST("/login", handlers.Login)
		}

		authProtected := api.Group("")
		authProtected.Use(middleware.RequireAuth())
		{
			authProtected.GET("/me", handlers.Me)
			authProtected.POST("/polls", handlers.CreatePoll(hub))
			authProtected.GET("/polls", handlers.ListMyPolls)
			authProtected.DELETE("/polls/:id", handlers.DeletePoll(hub))
		}

		api.GET("/polls/:id", handlers.GetPoll)
		api.POST("/polls/:id/vote", handlers.Vote(hub))
	}

	r.GET("/ws", handlers.WebSocket(hub))

	log.Printf("server listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}