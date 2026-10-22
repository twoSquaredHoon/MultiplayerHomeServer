package main

// /metrics in Prometheus text format, read by Prometheus and shown in Grafana.

import (
	"fmt"
	"net/http"
)

func metrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	serversMu.Lock()
	gameServers := len(servers)
	players := 0
	for _, s := range servers {
		players += s.players
	}
	serversMu.Unlock()

	lobbies := len(rdb.Keys(ctx, "lobby:*").Val())

	users := 0
	db.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&users)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP multiplayer_players_online Players connected to game servers.\n")
	fmt.Fprintf(w, "# TYPE multiplayer_players_online gauge\nmultiplayer_players_online %d\n", players)
	fmt.Fprintf(w, "# HELP multiplayer_game_servers Running game server containers.\n")
	fmt.Fprintf(w, "# TYPE multiplayer_game_servers gauge\nmultiplayer_game_servers %d\n", gameServers)
	fmt.Fprintf(w, "# HELP multiplayer_lobbies Open lobbies.\n")
	fmt.Fprintf(w, "# TYPE multiplayer_lobbies gauge\nmultiplayer_lobbies %d\n", lobbies)
	fmt.Fprintf(w, "# HELP multiplayer_users_total Registered accounts.\n")
	fmt.Fprintf(w, "# TYPE multiplayer_users_total gauge\nmultiplayer_users_total %d\n", users)
}
