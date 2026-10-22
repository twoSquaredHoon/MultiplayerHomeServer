package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	checkEvery     = 15 * time.Second
	heartbeatLimit = 2 * time.Minute // no heartbeat this long → server is dead (also boot time)
	emptyLimit     = 5 * time.Minute // no players this long → shut down
)

type gameServer struct {
	containerID    string
	code           string
	players        int
	lastHeartbeat  time.Time
	lastHadPlayers time.Time
}

// Running game servers, by port.
var (
	servers   = map[int]*gameServer{}
	serversMu sync.Mutex
)

func startGameServer(port int, code string) error {
	id, err := dockerStart(port, code)
	if err != nil {
		return err
	}
	now := time.Now()
	serversMu.Lock()
	servers[port] = &gameServer{containerID: id, code: code, lastHeartbeat: now, lastHadPlayers: now}
	serversMu.Unlock()
	log.Printf("started game server for lobby %s on port %d", code, port)
	return nil
}

// Game servers call this every few seconds with their player count.
func heartbeat(w http.ResponseWriter, r *http.Request) {
	if serverKey == "" || r.Header.Get("X-Server-Key") != serverKey {
		http.Error(w, "bad server key", http.StatusUnauthorized)
		return
	}
	var body struct {
		Port    int `json:"port"`
		Players int `json:"players"`
	}
	if json.NewDecoder(r.Body).Decode(&body) != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	serversMu.Lock()
	defer serversMu.Unlock()
	s, ok := servers[body.Port]
	if !ok {
		http.Error(w, "unknown server", http.StatusNotFound)
		return
	}
	s.players = body.Players
	s.lastHeartbeat = time.Now()
	if body.Players > 0 {
		s.lastHadPlayers = time.Now()
	}
	w.Write([]byte("ok"))
}

// Every few seconds, shut down game servers that are expired, dead, or empty.
func cleanupLoop() {
	for {
		time.Sleep(checkEvery)
		ctx := context.Background()

		serversMu.Lock()
		for port, s := range servers {
			reason := ""
			if rdb.Exists(ctx, "lobby:"+s.code).Val() == 0 {
				reason = "lobby expired"
			} else if time.Since(s.lastHeartbeat) > heartbeatLimit {
				reason = "no heartbeat"
			} else if time.Since(s.lastHadPlayers) > emptyLimit {
				reason = "empty"
			}
			if reason == "" {
				continue
			}

			log.Printf("stopping game server for lobby %s on port %d (%s)", s.code, port, reason)
			dockerRemove(s.containerID)
			deleteLobby(s.code, port)
			delete(servers, port)
		}
		serversMu.Unlock()
	}
}

func deleteLobby(code string, port int) {
	ctx := context.Background()
	rdb.Del(ctx, "lobby:"+code, "port:"+strconv.Itoa(port))
}

func removeOldGameServers() {
	ctx := context.Background()

	if gameImage != "" {
		ids, err := dockerListGameServers()
		if err != nil {
			log.Println("could not list old game servers:", err)
		}
		for _, id := range ids {
			dockerRemove(id)
		}
	}

	// Old lobbies point at game servers that no longer exist.
	for _, pattern := range []string{"lobby:*", "port:*"} {
		keys := rdb.Keys(ctx, pattern).Val()
		if len(keys) > 0 {
			rdb.Del(ctx, keys...)
		}
	}
}
