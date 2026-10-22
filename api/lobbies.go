package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

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

	// Start a game server for this lobby.
	if gameImage != "" {
		if err := startGameServer(port, code); err != nil {
			log.Println("could not start game server:", err)
			deleteLobby(code, port)
			http.Error(w, "could not start game server", http.StatusInternalServerError)
			return
		}
	}

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

func randomCode() string {
	letters := "ABCDEFGHJKLMNPQRSTUVWXYZ" // no I or O so codes are easy to read
	b := make([]byte, 4)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
