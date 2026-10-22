package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
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
)

func main() {
	ctx := context.Background()

	var err error
	db, err = pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	rdb = redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDR")})
	jwtSecret = []byte(os.Getenv("JWT_SECRET"))
	publicHost = os.Getenv("PUBLIC_HOST")

	_, err = db.Exec(ctx, `CREATE TABLE IF NOT EXISTS users (
		id            SERIAL PRIMARY KEY,
		username      TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at    TIMESTAMPTZ DEFAULT now()
	)`)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("POST /register", register)
	mux.HandleFunc("POST /login", login)
	mux.HandleFunc("POST /lobbies", auth(createLobby))
	mux.HandleFunc("POST /lobbies/{code}/join", auth(joinLobby))

	log.Println("API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

// ---------- accounts ----------

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func register(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if json.NewDecoder(r.Body).Decode(&c) != nil || c.Username == "" || len(c.Password) < 8 {
		http.Error(w, "need username and password (8+ chars)", http.StatusBadRequest)
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)

	var id int
	err := db.QueryRow(r.Context(),
		"INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id",
		c.Username, string(hash)).Scan(&id)
	if err != nil {
		http.Error(w, "username taken", http.StatusConflict)
		return
	}
	writeJSON(w, map[string]any{"id": id})
}

func login(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if json.NewDecoder(r.Body).Decode(&c) != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var id int
	var hash string
	err := db.QueryRow(r.Context(),
		"SELECT id, password_hash FROM users WHERE username = $1", c.Username).Scan(&id, &hash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(c.Password)) != nil {
		http.Error(w, "wrong username or password", http.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  fmt.Sprint(id),
		"name": c.Username,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
	})
	signed, _ := token.SignedString(jwtSecret)
	writeJSON(w, map[string]any{"token": signed})
}

// auth checks the "Authorization: Bearer <token>" header before running the handler.
func auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		_, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) { return jwtSecret, nil },
			jwt.WithValidMethods([]string{"HS256"}))
		if err != nil {
			http.Error(w, "not logged in", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// ---------- lobbies ----------

func createLobby(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Pick a lobby code that isn't in use.
	code := ""
	for i := 0; i < 10; i++ {
		c := randomCode()
		if rdb.Exists(ctx, "lobby:"+c).Val() == 0 {
			code = c
			break
		}
	}
	if code == "" {
		http.Error(w, "could not make a lobby code", http.StatusServiceUnavailable)
		return
	}

	// Grab the first free UDP port. The key expires with the lobby, so the port frees itself.
	port := 0
	for p := firstPort; p <= lastPort; p++ {
		if rdb.SetNX(ctx, fmt.Sprintf("port:%d", p), code, lobbyTTL).Val() {
			port = p
			break
		}
	}
	if port == 0 {
		http.Error(w, "server full", http.StatusServiceUnavailable)
		return
	}

	rdb.HSet(ctx, "lobby:"+code, "port", port)
	rdb.Expire(ctx, "lobby:"+code, lobbyTTL)

	// TODO (Phase 4): start a game server container on this port.

	writeJSON(w, map[string]any{"code": code, "host": publicHost, "port": port})
}

func joinLobby(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(r.PathValue("code"))
	port, err := rdb.HGet(r.Context(), "lobby:"+code, "port").Int()
	if err != nil {
		http.Error(w, "lobby not found", http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]any{"code": code, "host": publicHost, "port": port})
}

// ---------- helpers ----------

func randomCode() string {
	letters := "ABCDEFGHJKLMNPQRSTUVWXYZ" // no I or O so codes are easy to read
	b := make([]byte, 4)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
