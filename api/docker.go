package main

// Talks to Docker through its socket (/var/run/docker.sock) using plain HTTP,
// the same API the docker command uses.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
)

const gameServerLabel = "multiplayer.gameserver"

var dockerClient = &http.Client{
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", "/var/run/docker.sock")
		},
	},
}

func dockerRequest(method, path string, body any) ([]byte, int, error) {
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, "http://docker"+path, reader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := dockerClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return out, resp.StatusCode, nil
}

// dockerStart runs a game server container for one lobby and returns its ID.
// Same as: docker run -d --rm -p PORT:PORT/udp --network NET -e PORT=... IMAGE
func dockerStart(port int, code string) (string, error) {
	portKey := fmt.Sprintf("%d/udp", port)
	config := map[string]any{
		"Image": gameImage,
		"Env": []string{
			fmt.Sprintf("PORT=%d", port),
			"LOBBY_CODE=" + code,
			"API_URL=" + apiURLForServers,
			"SERVER_KEY=" + serverKey,
		},
		"Labels":       map[string]string{gameServerLabel: "true"},
		"ExposedPorts": map[string]any{portKey: map[string]any{}},
		"HostConfig": map[string]any{
			"PortBindings": map[string]any{portKey: []map[string]string{{"HostPort": fmt.Sprint(port)}}},
			"NetworkMode":  gameNetwork,
			"AutoRemove":   true,
		},
	}

	name := fmt.Sprintf("gameserver-%d", port)
	out, status, err := dockerRequest("POST", "/containers/create?name="+name, config)
	if err != nil {
		return "", err
	}
	if status != http.StatusCreated {
		return "", fmt.Errorf("create container: %d %s", status, out)
	}
	var created struct{ Id string }
	json.Unmarshal(out, &created)

	out, status, err = dockerRequest("POST", "/containers/"+created.Id+"/start", nil)
	if err != nil {
		return "", err
	}
	if status != http.StatusNoContent {
		dockerRemove(created.Id)
		return "", fmt.Errorf("start container: %d %s", status, out)
	}
	return created.Id, nil
}

// dockerRemove force-stops and deletes a container.
func dockerRemove(id string) {
	dockerRequest("DELETE", "/containers/"+id+"?force=true", nil)
}

// dockerListGameServers returns the IDs of all game server containers.
func dockerListGameServers() ([]string, error) {
	filters := url.QueryEscape(`{"label":["` + gameServerLabel + `=true"]}`)
	out, status, err := dockerRequest("GET", "/containers/json?all=true&filters="+filters, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("list containers: %d %s", status, out)
	}
	var list []struct{ Id string }
	json.Unmarshal(out, &list)

	ids := []string{}
	for _, c := range list {
		ids = append(ids, c.Id)
	}
	return ids, nil
}
