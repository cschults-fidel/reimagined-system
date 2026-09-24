# reimagined-system

A minimal deployments API in Go, using only the standard library. Data lives in memory and is lost when the server stops.

## Run

```sh
go run .            # listens on :8080 (override with PORT=9000)
go test ./...
```

## Endpoints

| Method | Path                | Description                    |
| ------ | ------------------- | ------------------------------ |
| POST   | `/deployments`      | Create a deployment            |
| GET    | `/deployments`      | List all deployments           |
| GET    | `/deployments/{id}` | Get one deployment by ID       |

### Create

`name` and `image` are required. `replicas` defaults to `1`.

```sh
curl -i -X POST localhost:8080/deployments \
  -H 'Content-Type: application/json' \
  -d '{"name":"web","image":"nginx:1.27","replicas":2}'
```

```json
{"id":"1","name":"web","image":"nginx:1.27","replicas":2,"status":"pending","created_at":"2026-09-24T19:50:00Z"}
```

### Read

```sh
curl localhost:8080/deployments
curl localhost:8080/deployments/1
```

Errors are returned as `{"error": "..."}` with a 400 or 404 status.
