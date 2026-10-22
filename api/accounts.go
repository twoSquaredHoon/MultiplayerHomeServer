package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

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

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
