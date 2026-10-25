package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func register(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if !readJSON(w, r, &c) {
		return
	}
	c.Username = strings.TrimSpace(c.Username)
	// bcrypt only works on the first 72 bytes and errors on anything longer.
	if c.Username == "" || len(c.Username) > 32 || len(c.Password) < 8 || len(c.Password) > 72 {
		http.Error(w, "need username (1-32 chars) and password (8-72 chars)", http.StatusBadRequest)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("bcrypt:", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	var id int
	err = db.QueryRow(r.Context(),
		"INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id",
		c.Username, string(hash)).Scan(&id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		http.Error(w, "username taken", http.StatusConflict)
		return
	}
	if err != nil {
		log.Println("register:", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"id": id})
}

func login(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if !readJSON(w, r, &c) {
		return
	}
	c.Username = strings.TrimSpace(c.Username)
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

// readJSON decodes a small JSON body into v. On failure it writes a 400 and returns false.
func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if json.NewDecoder(r.Body).Decode(v) != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
