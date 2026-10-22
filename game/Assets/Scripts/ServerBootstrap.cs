using System;
using System.Collections;
using System.Text;
using FishNet;
using FishNet.Managing;
using UnityEngine;
using UnityEngine.Networking;

// Runs only in the headless server build (-batchmode).
// Starts the FishNet server on PORT and sends heartbeats to the API.
// Put on the same GameObject as the NetworkManager.
public class ServerBootstrap : MonoBehaviour
{
    ushort port = 7770;
    string apiUrl;
    string serverKey;

    void Start()
    {
        if (!Application.isBatchMode)
        {
            enabled = false;
            return;
        }

        Application.targetFrameRate = 30;  // no need to run faster on a server

        // The API passes these when it starts the container.
        string portText = Environment.GetEnvironmentVariable("PORT");
        if (!string.IsNullOrEmpty(portText)) port = ushort.Parse(portText);
        apiUrl = Environment.GetEnvironmentVariable("API_URL");
        serverKey = Environment.GetEnvironmentVariable("SERVER_KEY");

        NetworkManager nm = InstanceFinder.NetworkManager;
        if (nm.ServerManager.Started) nm.ServerManager.StopConnection(true);  // in case "Start On Headless" is on
        nm.TransportManager.Transport.SetPort(port);
        nm.ServerManager.StartConnection();
        Debug.Log("Game server started on port " + port);

        if (!string.IsNullOrEmpty(apiUrl)) StartCoroutine(Heartbeat());
    }

    // Tells the API every 5 seconds that this server is alive and how many players it has.
    IEnumerator Heartbeat()
    {
        while (true)
        {
            int players = InstanceFinder.ServerManager.Clients.Count;
            string json = "{\"port\":" + port + ",\"players\":" + players + "}";

            using (var req = new UnityWebRequest(apiUrl + "/servers/heartbeat", "POST"))
            {
                req.uploadHandler = new UploadHandlerRaw(Encoding.UTF8.GetBytes(json));
                req.downloadHandler = new DownloadHandlerBuffer();
                req.SetRequestHeader("Content-Type", "application/json");
                req.SetRequestHeader("X-Server-Key", serverKey);
                yield return req.SendWebRequest();

                if (req.result != UnityWebRequest.Result.Success)
                    Debug.LogWarning("Heartbeat failed: " + req.error + " " + req.downloadHandler.text);
            }

            yield return new WaitForSeconds(5f);
        }
    }
}
