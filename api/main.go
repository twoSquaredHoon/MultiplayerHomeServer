package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	firstPort = 7770
	lastPort  = 7790
	lobbyTTL  = 2 * time.Hour
)

var (
	db         *pgxpool.Pool
	rdb        *redis.Client
	jwtSecret  []byte
	publicHost string

	// Game server settings. If gameImage is empty, lobbies work but no game server is started.
	gameImage        string // e.g. multiplayer-gameserver:latest
	gameNetwork      string // docker network the game servers join
	apiURLForServers string // how game servers reach this API
	serverKey        string // game servers send this with heartbeats
)

func main() {
	ctx := context.Background()

	var err error
	db, err = pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	rdb = redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDR")})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("redis: ", err)
	}
	jwtSecret = []byte(os.Getenv("JWT_SECRET"))
	// An empty secret would let anyone make valid login tokens.
	if len(jwtSecret) < 16 {
		log.Fatal("JWT_SECRET must be set (16+ chars)")
	}
	publicHost = os.Getenv("PUBLIC_HOST")
	gameImage = os.Getenv("GAME_SERVER_IMAGE")
	gameNetwork = os.Getenv("GAME_NETWORK")
	apiURLForServers = os.Getenv("API_URL_FOR_SERVERS")
	serverKey = os.Getenv("SERVER_KEY")

	_, err = db.Exec(ctx, `CREATE TABLE IF NOT EXISTS users (
		id            SERIAL PRIMARY KEY,
		username      TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at    TIMESTAMPTZ DEFAULT now()
	)`)
	if err != nil {
		log.Fatal(err)
	}

	// Start from a clean slate: game servers from a previous run aren't tracked anymore.
	removeOldGameServers()
	go cleanupLoop()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("GET /metrics", metrics)
	mux.HandleFunc("POST /register", register)
	mux.HandleFunc("POST /login", login)
	mux.HandleFunc("POST /lobbies", auth(createLobby))
	mux.HandleFunc("POST /lobbies/{code}/join", auth(joinLobby))
	mux.HandleFunc("POST /servers/heartbeat", heartbeat)

	log.Println("API listening on :8080")
	// Timeouts so a slow or stuck client can't hold a connection open forever.
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
