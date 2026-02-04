# Detached Worker

## How to run (Only on a single node)
- Start a local ray cluster:

```n
ray start --head --port=6379

```n
- Run the server

```n
python3 server.py

```n
- On another terminal, Run the client

```n
python3 client.py

```
