package main

import (
	"context"
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
	if !readJSON(w, r, &body) {
		return
	}

	serversMu.Lock()
	s, ok := servers[body.Port]
	code := ""
	if ok {
		s.players = body.Players
		s.lastHeartbeat = time.Now()
		if body.Players > 0 {
			s.lastHadPlayers = time.Now()
		}
		code = s.code
	}
	serversMu.Unlock()

	if !ok {
		http.Error(w, "unknown server", http.StatusNotFound)
		return
	}

	// Keep the lobby alive while people are playing, so a long match isn't cut off at lobbyTTL.
	if body.Players > 0 {
		ctx := r.Context()
		rdb.Expire(ctx, "lobby:"+code, lobbyTTL)
		rdb.Expire(ctx, "port:"+strconv.Itoa(body.Port), lobbyTTL)
	}
	w.Write([]byte("ok"))
}

// Every few seconds, shut down game servers that are expired, dead, or empty.
func cleanupLoop() {
	type toStop struct {
		port int
		s    *gameServer
	}
	for {
		time.Sleep(checkEvery)
		ctx := context.Background()

		// Copy the list so the lock isn't held during Redis/Docker calls
		// (heartbeats and /metrics would wait on it otherwise).
		serversMu.Lock()
		list := []toStop{}
		for port, s := range servers {
			list = append(list, toStop{port, s})
		}
		serversMu.Unlock()

		for _, item := range list {
			port, s := item.port, item.s
			serversMu.Lock()
			lastHeartbeat, lastHadPlayers := s.lastHeartbeat, s.lastHadPlayers
			serversMu.Unlock()

			reason := ""
			if rdb.Exists(ctx, "lobby:"+s.code).Val() == 0 {
				reason = "lobby expired"
			} else if time.Since(lastHeartbeat) > heartbeatLimit {
				reason = "no heartbeat"
			} else if time.Since(lastHadPlayers) > emptyLimit {
				reason = "empty"
			}
			if reason == "" {
				continue
			}

			log.Printf("stopping game server for lobby %s on port %d (%s)", s.code, port, reason)
			serversMu.Lock()
			delete(servers, port)
			serversMu.Unlock()
			dockerRemove(s.containerID)
			deleteLobby(s.code, port)
		}
	}
}

// scanKeys returns all Redis keys matching pattern. Uses SCAN instead of KEYS,
// which blocks Redis while it walks every key.
func scanKeys(ctx context.Context, pattern string) []string {
	keys := []string{}
	iter := rdb.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	return keys
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
		keys := scanKeys(ctx, pattern)
		if len(keys) > 0 {
			rdb.Del(ctx, keys...)
		}
	}
}
