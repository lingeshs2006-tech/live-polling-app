# LivePoll — Real-Time Polling Tool

Create a poll, share the link, and watch results stream in live — no refresh required.

---

## How it works (architecture overview)

```
Browser ←——WebSocket——→ Go backend ←——Pub/Sub——→ Redis
                         ↕
                       MongoDB
```

1. A logged-in user **creates a poll**. The question and options are stored in MongoDB; a zero-count hash is seeded in Redis (`poll:{id}:counts`).
2. The **share link** (`/poll/{id}`) opens a public vote page. Anyone can vote — no account needed.
3. When a vote arrives, the server **`HINCRBY`'s the option's count in Redis** (atomic, single source of truth), writes the vote to MongoDB for durability, then **`PUBLISH`es a JSON event** to a Redis channel.
4. A single Redis listener in the Go hub **subscribes to `poll:*`** using a wildcard pattern, receives the event, and fans it out to every WebSocket client watching that poll.
5. The **frontend renders the new bar widths** from the live counts — zero refresh, pure push.

MongoDB is the durable store. Redis is the live engine. The WebSocket is the wire.

---

## Project structure

```
/backend
  main.go              ← server entry point, Gin router
  config/config.go     ← env-based config (ports, URIs, secrets)
  database/mongo.go    ← MongoDB connection + collection handles
  database/redis.go    ← Redis client + key naming helpers
  models/models.go     ← User, Poll, Vote structs
  handlers/auth.go     ← signup / login / me (bcrypt + JWT)
  handlers/poll.go     ← create / get / list / delete
  handlers/vote.go     ← vote handler (HINCRBY + Pub/Sub)
  handlers/ws.go       ← WebSocket upgrade + initial snapshot
  middleware/auth.go    ← JWT middleware + token generation
  middleware/cors.go    ← CORS configuration
  realtime/hub.go      ← WebSocket hub + Redis pub/sub listener
  realtime/client.go   ← per-connection read/write pumps
  services/poll.go     ← Redis ↔ Mongo count sync

/frontend
  src/
    api/client.js      ← fetch wrapper, voter token, wsUrl helper
    context/AuthContext.jsx  ← auth state + provider
    context/useAuth.js       ← useAuth hook
    components/
      Navbar.jsx       ← top nav, auth-aware
      LiveResults.jsx  ← live bar chart (driven by WS data)
      ProtectedRoute.jsx
    pages/
      Home.jsx         ← hero (guest) / create poll form (authed)
      Auth.jsx         ← login + signup
      Dashboard.jsx    ← my polls, share links, delete
      PollPage.jsx     ← vote + live results (the shareable link)
```

---

## Prerequisites

| Tool | Why |
|------|-----|
| Go ≥ 1.21 | Backend |
| Node.js ≥ 18 | Frontend |
| MongoDB running on `localhost:27017` | Persistent storage |
| Redis running on `localhost:6379` | Live vote counts + Pub/Sub |

---

## Running locally

### 1. Start MongoDB and Redis

On Windows with Docker:
```
docker run -d --name mongo -p 27017:27017 mongo:7
docker run -d --name redis -p 6379:6379 redis:7
```

Or install them natively and ensure both are listening on their default ports.

### 2. Backend

```bash
cd backend
cp .env.example .env   # or just use the defaults already there
go mod tidy
go run .
```

The server starts on `http://localhost:8080`.

### 3. Frontend

```bash
cd frontend
npm install
npm run dev
```

The app opens on `http://localhost:5173`. Vite proxies `/api` and `/ws` to the backend automatically.

---

## How the real-time pipeline actually works

### Vote path

1. `POST /api/polls/:id/vote` arrives at `handlers/vote.go`
2. The handler validates the option index against MongoDB (never trusts the client)
3. `HINCRBY poll:{id}:counts {optionIndex} 1` — atomic increment in Redis
4. `SADD poll:{id}:voters {voterHash}` — dedup check
5. `db.votes.insert(...)` — durability
6. `Read back HGETALL poll:{id}:counts` → build a fresh snapshot
7. `PUBLISH poll:{id}:events {JSON}` → Redis Pub/Sub channel

### Broadcast path

8. `hub.StartRedisListener()` subscribes to `poll:*` once at boot
9. The listener goroutine receives the message from the channel
10. The hub maps the poll ID to connected clients and pushes the JSON to each

### Render path

11. The browser's `WebSocket.onmessage` fires, parses the counts
12. React state updates → the bar widths recompute and animate

No polling. No long-polling. No page refresh.

---

## Key design decisions

- **Redis is the live source of truth, not MongoDB.** Counts live in Redis hashes. If Redis restarts, the next read rehydrates from MongoDB automatically (`services.SyncPollCountsFromRedis`).
- **One Redis subscription per hub, not per client.** The wildcard `poll:*` pattern is shared across all connected clients — scales without opening N Redis connections.
- **Voter dedup via Redis Set + SHA-256 token.** Not bulletproof against a determined attacker, but stops accidental double-votes for real users in a demo.
- **JWT-based auth for poll creation only.** Voting is public (as it should be for an audience tool). Login is only required to create and manage polls.
- **Vite proxy handles CORS in dev.** The Go backend also has CORS middleware for production deployments where the frontend is served separately.

---

## Deployment

Both parts are independent services.

- **Frontend**: `npm run build` produces static files in `dist/`. Deploy to Vercel, Netlify, Cloudflare Pages, or serve behind nginx.
- **Backend**: `go build -o server .` produces a single binary. Deploy to any VPS, Railway, Fly.io, etc. Set env vars for production Mongo URI, Redis address, and JWT secret.

When deployed, set `VITE_API_BASE` to point the frontend at the backend URL, and `CLIENT_ORIGIN` on the backend to allow the frontend domain.