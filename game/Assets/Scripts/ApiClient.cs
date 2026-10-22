using System;
using System.Collections;
using System.Text;
using UnityEngine;
using UnityEngine.Networking;

[Serializable] public class Credentials { public string username; public string password; }
[Serializable] public class TokenResponse { public string token; }
[Serializable] public class LobbyResponse { public string code; public string host; public int port; }

// Talks to the Go API (login, lobbies).
public static class ApiClient
{
    public static string BaseUrl = "http://127.0.0.1:8080";
    public static string Token = "";

    // POSTs JSON to the API. onDone gets (success, response text or error).
    public static IEnumerator Post(string path, string json, Action<bool, string> onDone)
    {
        using (var req = new UnityWebRequest(BaseUrl + path, "POST"))
        {
            req.uploadHandler = new UploadHandlerRaw(Encoding.UTF8.GetBytes(json));
            req.downloadHandler = new DownloadHandlerBuffer();
            req.SetRequestHeader("Content-Type", "application/json");
            if (Token != "") req.SetRequestHeader("Authorization", "Bearer " + Token);

            yield return req.SendWebRequest();

            bool ok = req.result == UnityWebRequest.Result.Success;
            string text = req.downloadHandler.text;
            if (!ok && text == "") text = req.error;
            onDone(ok, text.Trim());
        }
    }
}
