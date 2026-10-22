using FishNet.Object;
using UnityEngine;

// Put on the player prefab together with NetworkObject and NetworkTransform.
// Keyboard: WASD / arrows. Phone: touch and hold toward where you want to go.
public class PlayerController : NetworkBehaviour
{
    public float speed = 5f;

    public override void OnStartClient()
    {
        base.OnStartClient();
        // Your own player is green, everyone else is red.
        Renderer r = GetComponent<Renderer>();
        if (r != null) r.material.color = IsOwner ? Color.green : Color.red;
    }

    void Update()
    {
        if (!IsOwner) return;

        Vector3 move = new Vector3(Input.GetAxisRaw("Horizontal"), 0, Input.GetAxisRaw("Vertical"));

        if (Input.touchCount > 0)
        {
            Vector2 center = new Vector2(Screen.width / 2f, Screen.height / 2f);
            Vector2 d = Input.GetTouch(0).position - center;
            move = new Vector3(d.x, 0, d.y);
        }

        transform.position += move.normalized * speed * Time.deltaTime;
    }
}
