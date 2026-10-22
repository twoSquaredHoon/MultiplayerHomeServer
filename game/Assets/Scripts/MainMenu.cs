using System.Collections;
using FishNet;
using FishNet.Managing;
using UnityEngine;

// Login + host/join lobby menu for players. Drawn with OnGUI so no UI setup is needed.
// Put on the same GameObject as the NetworkManager.
public class MainMenu : MonoBehaviour
{
    [Tooltip("Your server's API, e.g. http://100.101.102.103:8080 (the server's Tailscale IP)")]
    public string apiUrl = "http://127.0.0.1:8080";

    string username = "";
    string password = "";
    string joinCode = "";
    string message = "";
    string lobbyCode = "";
    bool loggedIn = false;
    bool busy = false;

    void Start()
    {
        // The dedicated server has no menu.
        if (Application.isBatchMode)
        {
            enabled = false;
            return;
        }
        ApiClient.BaseUrl = apiUrl;
    }

    void OnGUI()
    {
        // Make the menu bigger on phones.
        float scale = Screen.dpi > 200 ? 2.5f : 1.5f;
        GUI.matrix = Matrix4x4.Scale(new Vector3(scale, scale, 1));
        GUILayout.BeginArea(new Rect(10, 10, 320, 400));

        if (lobbyCode != "")
        {
            GUILayout.Label("Lobby code: " + lobbyCode + "  (send this to friends)");
            if (GUILayout.Button("Leave")) Leave();
        }
        else if (!loggedIn)
        {
            GUILayout.Label("Username");
            username = GUILayout.TextField(username);
            GUILayout.Label("Password (8+ characters)");
            password = GUILayout.PasswordField(password, '*');
            GUI.enabled = !busy;
            if (GUILayout.Button("Log in")) Login();
            if (GUILayout.Button("Create account")) Register();
            GUI.enabled = true;
        }
        else
        {
            GUI.enabled = !busy;
            if (GUILayout.Button("Host new lobby")) HostLobby();
            GUILayout.Label("Or join with a code");
            joinCode = GUILayout.TextField(joinCode, 4).ToUpper();
            if (GUILayout.Button("Join")) JoinLobby();
            GUI.enabled = true;
        }

        GUILayout.Label(message);
        GUILayout.EndArea();
    }

    string CredentialsJson()
    {
        return JsonUtility.ToJson(new Credentials { username = username, password = password });
    }

    void Register()
    {
        busy = true;
        StartCoroutine(ApiClient.Post("/register", CredentialsJson(), (ok, text) =>
        {
            busy = false;
            if (!ok) { message = text; return; }
            Login();
        }));
    }

    void Login()
    {
        busy = true;
        StartCoroutine(ApiClient.Post("/login", CredentialsJson(), (ok, text) =>
        {
            busy = false;
            if (!ok) { message = text; return; }
            ApiClient.Token = JsonUtility.FromJson<TokenResponse>(text).token;
            loggedIn = true;
            message = "";
        }));
    }

    void HostLobby()
    {
        busy = true;
        message = "Starting game server...";
        StartCoroutine(ApiClient.Post("/lobbies", "{}", (ok, text) =>
        {
            busy = false;
            if (!ok) { message = text; return; }
            Connect(JsonUtility.FromJson<LobbyResponse>(text));
        }));
    }

    void JoinLobby()
    {
        busy = true;
        StartCoroutine(ApiClient.Post("/lobbies/" + joinCode + "/join", "{}", (ok, text) =>
        {
            busy = false;
            if (!ok) { message = text; return; }
            Connect(JsonUtility.FromJson<LobbyResponse>(text));
        }));
    }

    void Connect(LobbyResponse lobby)
    {
        lobbyCode = lobby.code;
        StartCoroutine(ConnectWithRetry(lobby.host, (ushort)lobby.port));
    }

    // A new game server takes a few seconds to boot, so keep trying for a bit.
    IEnumerator ConnectWithRetry(string host, ushort port)
    {
        NetworkManager nm = InstanceFinder.NetworkManager;
        for (int attempt = 1; attempt <= 10; attempt++)
        {
            message = "Connecting... (" + attempt + ")";
            nm.ClientManager.StartConnection(host, port);

            float waited = 0f;
            while (waited < 3f && !nm.ClientManager.Started)
            {
                waited += Time.deltaTime;
                yield return null;
            }
            if (nm.ClientManager.Started)
            {
                message = "";
                yield break;
            }

            nm.ClientManager.StopConnection();
            yield return new WaitForSeconds(1f);
        }
        message = "Could not connect to the game server.";
        lobbyCode = "";
    }

    void Leave()
    {
        InstanceFinder.NetworkManager.ClientManager.StopConnection();
        lobbyCode = "";
    }
}
