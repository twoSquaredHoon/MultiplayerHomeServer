# game/

Unity project goes here (client + headless server build).

## Setup (Phase 2)
1. Unity Hub → New project → **Universal 3D** or **2D (URP)**, location: this `game/` folder.
2. Install modules in Unity Hub: **Linux Dedicated Server Build Support**, Android, iOS.
3. Add **FishNet** (Asset Store or `https://github.com/FirstGearGames/FishNet.git?path=Assets/FishNet` via Package Manager → Add from git URL).
4. Scene: `NetworkManager` + `Tugboat` transport, port 7770.
5. Player prefab with `NetworkObject` + `NetworkTransform`, move only if `IsOwner`.

## Server build
File → Build Profiles → **Linux Server** → build into `game/Builds/Server/`.
Run headless: `./Server.x86_64 -batchmode -nographics -port 7770`

Later: add `game/server-docker/Dockerfile` that copies `Builds/Server/` into an Ubuntu image.

## Client flow
1. `POST /login` → token
2. `POST /lobbies` (host) or `POST /lobbies/{code}/join` → `{host, port}`
3. Connect FishNet client to that host/port
