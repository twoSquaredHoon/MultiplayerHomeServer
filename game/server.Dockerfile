# Game server image. Built from the Unity Linux Server build in Builds/Server/.
# Use deploy/push-gameserver.sh instead of building this by hand.
FROM ubuntu:24.04
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /game
COPY Builds/Server/ .
RUN chmod +x Server.x86_64
# PORT, API_URL, SERVER_KEY come from the API when it starts the container
ENTRYPOINT ["./Server.x86_64", "-batchmode", "-nographics"]
