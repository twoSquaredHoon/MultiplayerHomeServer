# game/

Unity project: one project builds both the game client and the headless game server.

The scripts are in `Assets/Scripts/`. The scene and prefab have to be made in the Unity editor (steps below).

| Script | What it does |
|---|---|
| `ApiClient.cs` | Calls the Go API (login, lobbies) |
| `MainMenu.cs` | Login / create account / host lobby / join by code menu |
| `ServerBootstrap.cs` | Server build only: starts on `PORT`, sends heartbeats to the API |
| `PlayerController.cs` | Moves your player (keyboard or touch) |

## 1. Open the project
1. Unity Hub → **Add → Add project from disk** → pick this `game/` folder. Use **Unity 6 LTS**.
2. In Unity Hub → Installs → your Unity version → Add modules: **Linux Dedicated Server Build Support**, **Mac/Windows/Linux Build Support**, **Android**, **iOS**.
3. Package Manager → **+ → Add package from git URL**:
   `https://github.com/FirstGearGames/FishNet.git?path=Assets/FishNet`
4. Edit → Project Settings → Player → Other Settings:
   - **Active Input Handling** → **Both**
   - **Allow downloads over HTTP** → **Always allowed** (the API is plain http over Tailscale)

## 2. Make the player prefab
1. GameObject → 3D Object → **Capsule**, name it `Player`.
2. Add components: **NetworkObject**, **NetworkTransform**, **PlayerController**.
3. Drag it into `Assets/Prefabs/` to make a prefab, then delete it from the scene.

## 3. Set up the scene
1. GameObject → 3D Object → **Plane** (the ground). Move the Main Camera up and angle it down at the plane.
2. Create an empty GameObject `NetworkManager` and add: **NetworkManager** (FishNet adds Tugboat and the other managers), **PlayerSpawner**, **MainMenu**, **ServerBootstrap**.
3. On **PlayerSpawner**, set Player Prefab → your `Player` prefab.
4. On **MainMenu**, set Api Url → `http://<server Tailscale IP>:8080`.
5. Save the scene and add it in File → Build Profiles → Scene List.

## 4. Test on your Mac first
Run the backend locally (`deploy/`, with `GAME_SERVER_IMAGE` empty), make a lobby in the editor, then run a server build by hand with `PORT=<port from the lobby>` set. Or test against the home server once step 5 is done.

## 5. Build the game server
File → Build Profiles → **Linux Server** → Build into `game/Builds/Server/` with the name `Server`
(so the binary is `Builds/Server/Server.x86_64`).

Then load it onto the home server:
```bash
./deploy/push-gameserver.sh you@your-server
```
and set `GAME_SERVER_IMAGE=multiplayer-gameserver:latest` in `deploy/.env` on the server, then `docker compose up -d`.

## 6. Build the clients
Same Build Profiles window: **macOS**, **Windows**, **Linux**, **Android**, **iOS** (iOS needs Xcode and an Apple developer account).

## Notes
- Movement is client-authoritative (NetworkTransform default) to keep things simple. Making the server decide movement is a good next step.
- Players see the lobby code in the top-left after connecting; friends type it into Join.
