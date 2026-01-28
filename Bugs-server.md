# Bug Report: server

**Analysis Date:** 2026-01-28
**Directory Analyzed:** `/home/user/openpairing/backend/internal/server`
**Files Analyzed:** 1 file
**Bugs Found:** 12 bugs

---

## Summary

The server package implements a WebSocket hub for real-time chess game communication. The analysis revealed critical race conditions in concurrent data structure access, security vulnerabilities in origin checking, resource leaks in connection management, and several logic errors in client registration and game state broadcasting. The most severe issues involve unsynchronized access to shared maps that could cause panics or data corruption in production under concurrent load.

**Critical Issues:**
- Race condition when iterating over gameClients map (could cause panic)
- CSRF vulnerability from accepting all WebSocket origins
- Race condition with multiple concurrent connections per user
- Unsynchronized access to Client fields from multiple goroutines

---

## Bugs

## Bug #1: Race Condition in broadcastToGame Map Iteration

**Severity:** Critical

**Description:**
The `broadcastToGame` function copies the map reference under RLock, then releases the lock and iterates over the map without any protection. If another goroutine (such as the unregister handler) modifies the map during iteration, it will cause a data race and potential panic. This is a classic concurrent map read/write violation in Go.

**Location:**
- File: `/home/user/openpairing/backend/internal/server/websocket.go`
- Function: `broadcastToGame`
- Lines: 455-465

**Steps to Reproduce:**
1. Connect multiple clients to the same game (gameID)
2. Have one client disconnect (triggering unregister at line 106)
3. Simultaneously call `broadcastToGame` for that game (e.g., from a move)
4. The unregister handler modifies gameClients map at line 115 while broadcastToGame iterates at line 459
5. Result: Data race detected by Go race detector, potential panic

**Expected Behavior:** Safe concurrent access to the gameClients map
**Actual Behavior:** Unprotected iteration causes race condition

**Fix:**
Hold the read lock during the entire iteration, or copy the client list to a slice:

```diff
 func (h *Hub) broadcastToGame(gameID uuid.UUID, msgType string, data interface{}) {
 	msg := map[string]interface{}{
 		"type": msgType,
 		"data": data,
 	}

 	bytes, err := json.Marshal(msg)
 	if err != nil {
+		h.logger.Error().Err(err).Msg("Failed to marshal broadcast message")
 		return
 	}

 	h.mutex.RLock()
-	clients := h.gameClients[gameID]
-	h.mutex.RUnlock()
-
-	for client := range clients {
+	clientList := make([]*Client, 0, len(h.gameClients[gameID]))
+	for client := range h.gameClients[gameID] {
+		clientList = append(clientList, client)
+	}
+	h.mutex.RUnlock()
+
+	for _, client := range clientList {
 		select {
 		case client.Send <- bytes:
 		default:
 			// Client buffer full, skip
 		}
 	}
 }
```

---

## Bug #2: CSRF Vulnerability - CheckOrigin Accepts All Origins

**Severity:** Critical

**Description:**
The WebSocket upgrader's `CheckOrigin` function returns `true` for all requests without validating the origin header. This allows malicious websites to establish WebSocket connections to the application from any domain, enabling Cross-Site WebSocket Hijacking (CSWSH) attacks. An attacker could create a malicious website that connects authenticated users to the WebSocket endpoint and performs unauthorized actions.

**Location:**
- File: `/home/user/openpairing/backend/internal/server/websocket.go`
- Variable: `upgrader`
- Lines: 17-24

**Steps to Reproduce:**
1. User logs into openpairing.com and gets an authenticated session
2. User visits malicious-site.com (in another tab)
3. Malicious site's JavaScript establishes WebSocket connection to openpairing.com
4. Browser includes authentication cookies automatically
5. Attacker can now send WebSocket messages (make moves, resign games, etc.) as the user

**Expected Behavior:** Only allow WebSocket connections from trusted origins
**Actual Behavior:** All origins are accepted, enabling CSRF attacks

**Fix:**
Implement proper origin validation:

```diff
 var upgrader = websocket.Upgrader{
 	ReadBufferSize:  1024,
 	WriteBufferSize: 1024,
 	CheckOrigin: func(r *http.Request) bool {
-		// In production, check origin properly
-		return true
+		origin := r.Header.Get("Origin")
+		// Allow same-origin and trusted domains
+		allowedOrigins := []string{
+			"http://localhost:3000",  // Development
+			"https://openpairing.com", // Production
+			"https://www.openpairing.com",
+		}
+		for _, allowed := range allowedOrigins {
+			if origin == allowed {
+				return true
+			}
+		}
+		return false
 	},
 }
```

Or use environment variables for configuration:

```go
CheckOrigin: func(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	allowedOrigin := os.Getenv("ALLOWED_WEBSOCKET_ORIGIN")
	return origin == allowedOrigin
},
```

---

## Bug #3: Race Condition - Multiple Clients Per User Causes Orphaned Connections

**Severity:** High

**Description:**
The Hub stores only one client per user in the `userClients` map (line 98). When a user opens multiple connections (e.g., different browser tabs, mobile + desktop), the second connection overwrites the first in `userClients`, but the old client remains in the `clients` map. This causes:
1. Memory leak of abandoned Client objects
2. The old connection never gets properly cleaned up from `clients` map via `userClients` lookup
3. Potential confusion about which connection is active
4. Messages may be sent to wrong connections

**Location:**
- File: `/home/user/openpairing/backend/internal/server/websocket.go`
- Function: `Hub.Run` (register case)
- Lines: 95-104

**Steps to Reproduce:**
1. User logs in and establishes WebSocket connection A
2. Connection A is registered: clients[A]=true, userClients[userID]=A
3. Same user opens another tab and establishes connection B
4. Connection B is registered: clients[B]=true, userClients[userID]=B (overwrites A)
5. Now clients map has both A and B, but userClients only has B
6. When connection A disconnects, unregister tries to delete userClients[userID] but finds B, not A
7. Connection A remains in clients map indefinitely

**Expected Behavior:** Either reject multiple connections per user OR properly track all connections per user
**Actual Behavior:** Silent overwrite causes memory leak and orphaned connections

**Fix:**
Change `userClients` to store a slice of clients per user:

```diff
 type Hub struct {
 	// Registered clients
 	clients map[*Client]bool

-	// Clients by user ID
-	userClients map[uuid.UUID]*Client
+	// Clients by user ID (multiple connections allowed)
+	userClients map[uuid.UUID][]*Client

 	// Clients by game ID (including spectators)
 	gameClients map[uuid.UUID]map[*Client]bool
```

And update register/unregister logic:

```diff
 case client := <-h.register:
 	h.mutex.Lock()
 	h.clients[client] = true
-	h.userClients[client.UserID] = client
+	h.userClients[client.UserID] = append(h.userClients[client.UserID], client)
 	h.mutex.Unlock()

 case client := <-h.unregister:
 	h.mutex.Lock()
 	if _, ok := h.clients[client]; ok {
 		delete(h.clients, client)
-		delete(h.userClients, client.UserID)
+		// Remove this specific client from the user's client list
+		if clientList, ok := h.userClients[client.UserID]; ok {
+			for i, c := range clientList {
+				if c == client {
+					h.userClients[client.UserID] = append(clientList[:i], clientList[i+1:]...)
+					break
+				}
+			}
+			// If no more clients for this user, remove the entry
+			if len(h.userClients[client.UserID]) == 0 {
+				delete(h.userClients, client.UserID)
+			}
+		}

 		// Remove from game clients
 		if client.GameID != nil {
```

---

## Bug #4: Race Condition - Client Fields Accessed Without Synchronization

**Severity:** High

**Description:**
The `Client` struct has a mutex field `mu` (line 35) but it's never used. Meanwhile, `Client.GameID` and `Client.IsSpectator` are read and written from multiple goroutines without any synchronization:
- Written in `handleJoinGame` (lines 188-189) and `handleSpectate` (lines 403-404)
- Written in `handleLeaveGame` (lines 223-224)
- Read in `getSpectatorCount` (line 475)
- Read in `unregister` handler (line 113)

This violates Go's memory model and causes data races.

**Location:**
- File: `/home/user/openpairing/backend/internal/server/websocket.go`
- Struct: `Client`
- Lines: 27-36 (struct definition), 188-189, 223-224, 403-404, 475

**Steps to Reproduce:**
1. Client joins a game (handleJoinGame sets GameID at line 188)
2. Simultaneously, another goroutine calls getSpectatorCount (reads IsSpectator at line 475)
3. Run with `go run -race` to detect the race condition
4. Expected: Data race detected
5. Actual: Unpredictable behavior, potential incorrect spectator counts

**Expected Behavior:** All access to Client fields should be synchronized
**Actual Behavior:** Concurrent reads/writes without synchronization

**Fix:**
Use the existing Client.mu mutex for all field access:

```diff
 type Client struct {
 	ID         uuid.UUID
 	UserID     uuid.UUID
 	Conn       *websocket.Conn
 	Send       chan []byte
 	Hub        *Hub
-	GameID     *uuid.UUID // Current game being watched/played
-	IsSpectator bool
-	mu         sync.Mutex
+
+	mu          sync.RWMutex
+	gameID      *uuid.UUID // Current game being watched/played
+	isSpectator bool
 }
```

Add getter/setter methods:

```go
func (c *Client) GetGameID() *uuid.UUID {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.gameID
}

func (c *Client) SetGameID(gameID *uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gameID = gameID
}

func (c *Client) IsSpectator() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isSpectator
}

func (c *Client) SetIsSpectator(spectator bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isSpectator = spectator
}
```

Update all access sites to use these methods.

---

## Bug #5: Resource Leak - WebSocket Connections Not Closed on Shutdown

**Severity:** High

**Description:**
When the Hub's context is cancelled (line 92), the `Run` function returns immediately without cleaning up active clients. All open WebSocket connections remain open, and the read/write goroutines continue running indefinitely. This causes:
1. Goroutine leak (2 goroutines per client: readPump + writePump)
2. Memory leak (Client objects and their buffers)
3. TCP connection leak
4. Graceful shutdown is impossible

**Location:**
- File: `/home/user/openpairing/backend/internal/server/websocket.go`
- Function: `Hub.Run`
- Lines: 89-93

**Steps to Reproduce:**
1. Start server with multiple active WebSocket connections
2. Send context cancellation signal (e.g., SIGTERM)
3. Hub.Run returns at line 93
4. Use `pprof` to check goroutines: observe readPump and writePump goroutines still running
5. Expected: All connections closed, goroutines terminated
6. Actual: Connections and goroutines remain indefinitely

**Expected Behavior:** Close all connections and cleanup resources on shutdown
**Actual Behavior:** Immediate return leaves resources dangling

**Fix:**
Close all client connections before returning:

```diff
 func (h *Hub) Run(ctx context.Context) {
 	for {
 		select {
 		case <-ctx.Done():
+			// Cleanup: close all client connections
+			h.logger.Info().Msg("Shutting down WebSocket hub")
+			h.mutex.Lock()
+			for client := range h.clients {
+				close(client.Send)
+				client.Conn.Close()
+			}
+			h.mutex.Unlock()
+			h.logger.Info().Int("closed_connections", len(h.clients)).Msg("Hub shutdown complete")
 			return

 		case client := <-h.register:
```

Also update readPump and writePump to respect context:

```go
func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	// ... existing code ...

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// ... existing read logic ...
		}
	}
}
```

---

## Bug #6: Logic Error - Spectator Count Broadcast for Non-Spectators

**Severity:** Medium

**Description:**
When a client leaves a game via `handleLeaveGame`, the code always broadcasts a spectator count update (lines 227-230), regardless of whether the leaving client was a spectator or a player. This is misleading because:
1. Players leaving should not affect spectator count
2. Clients receive incorrect spectator count updates
3. The spectator count only changes when actual spectators join/leave

**Location:**
- File: `/home/user/openpairing/backend/internal/server/websocket.go`
- Function: `handleLeaveGame`
- Lines: 209-231

**Steps to Reproduce:**
1. Start a game with White and Black players
2. Add 2 spectators (spectator count = 2)
3. White player sends "leave_game" message
4. All clients receive "spectator_count" message with count=2
5. Expected: No spectator count update (white is a player, not spectator)
6. Actual: Spectator count broadcast even though count didn't change

**Expected Behavior:** Only broadcast spectator count when a spectator leaves
**Actual Behavior:** Always broadcasts spectator count

**Fix:**
Check if the leaving client was a spectator before broadcasting:

```diff
 func (h *Hub) handleLeaveGame(msg *Message) {
 	if msg.Client.GameID == nil {
 		return
 	}

 	gameID := *msg.Client.GameID
+	wasSpectator := msg.Client.IsSpectator

 	h.mutex.Lock()
 	if gameClients, ok := h.gameClients[gameID]; ok {
 		delete(gameClients, msg.Client)
 		if len(gameClients) == 0 {
 			delete(h.gameClients, gameID)
 		}
 	}
 	msg.Client.GameID = nil
 	msg.Client.IsSpectator = false
 	h.mutex.Unlock()

-	// Notify spectator count
-	h.broadcastToGame(gameID, "spectator_count", map[string]int{
-		"count": h.getSpectatorCount(gameID),
-	})
+	// Only notify spectator count if a spectator left
+	if wasSpectator {
+		h.broadcastToGame(gameID, "spectator_count", map[string]int{
+			"count": h.getSpectatorCount(gameID),
+		})
+	}
 }
```

---

## Bug #7: Error Handling - Silent JSON Marshal Failures

**Severity:** Medium

**Description:**
In `sendToClient` (lines 432-435) and `broadcastToGame` (lines 450-452), if `json.Marshal` fails, the functions silently return without logging the error or notifying anyone. This makes debugging extremely difficult because:
1. Messages fail to send without any indication
2. Clients don't receive expected updates
3. No logs to trace the issue
4. Could hide data structure serialization problems

**Location:**
- File: `/home/user/openpairing/backend/internal/server/websocket.go`
- Functions: `sendToClient`, `broadcastToGame`
- Lines: 426-442, 444-466

**Steps to Reproduce:**
1. Pass invalid data to sendToClient (e.g., circular reference, unsupported type)
2. json.Marshal returns an error at line 432
3. Function returns silently at line 434
4. Client never receives the message
5. No error logged, no way to know what happened

**Expected Behavior:** Log errors when JSON marshaling fails
**Actual Behavior:** Silent failure with no indication

**Fix:**
Add error logging:

```diff
 func (h *Hub) sendToClient(client *Client, msgType string, data interface{}) {
 	msg := map[string]interface{}{
 		"type": msgType,
 		"data": data,
 	}

 	bytes, err := json.Marshal(msg)
 	if err != nil {
+		h.logger.Error().
+			Err(err).
+			Str("type", msgType).
+			Str("client_id", client.ID.String()).
+			Msg("Failed to marshal message for client")
 		return
 	}

 	select {
 	case client.Send <- bytes:
 	default:
-		// Client buffer full, skip
+		h.logger.Warn().
+			Str("client_id", client.ID.String()).
+			Str("type", msgType).
+			Msg("Client send buffer full, message dropped")
 	}
 }
```

Apply similar changes to `broadcastToGame`.

---

## Bug #8: Security Vulnerability - No Rate Limiting on WebSocket Messages

**Severity:** High

**Description:**
The `readPump` function (lines 541-571) reads and processes messages from WebSocket clients without any rate limiting or throttling. A malicious or compromised client can flood the server with messages, causing:
1. CPU exhaustion from processing excessive messages
2. Memory exhaustion from queuing messages in the broadcast channel (buffer: 256)
3. DoS attack by overwhelming the Hub's event loop
4. Degraded performance for legitimate users

**Location:**
- File: `/home/user/openpairing/backend/internal/server/websocket.go`
- Function: `Client.readPump`
- Lines: 541-571

**Steps to Reproduce:**
1. Connect a WebSocket client
2. Send 10,000 messages per second
3. Server processes all messages without any throttling
4. Broadcast channel fills up
5. Hub event loop becomes overwhelmed
6. Expected: Client is rate-limited or disconnected
7. Actual: Server continues processing, causing DoS

**Expected Behavior:** Rate limit messages per client (e.g., 100 msg/second)
**Actual Behavior:** Unlimited message processing

**Fix:**
Implement token bucket rate limiting:

```diff
+import "golang.org/x/time/rate"

 type Client struct {
 	ID         uuid.UUID
 	UserID     uuid.UUID
 	Conn       *websocket.Conn
 	Send       chan []byte
 	Hub        *Hub
 	GameID     *uuid.UUID
 	IsSpectator bool
 	mu         sync.Mutex
+	limiter    *rate.Limiter
 }
```

Initialize in HandleWebSocket:

```diff
 	client := &Client{
 		ID:     uuid.New(),
 		UserID: userID,
 		Conn:   conn,
 		Send:   make(chan []byte, 256),
 		Hub:    h,
+		limiter: rate.NewLimiter(rate.Limit(100), 200), // 100 msg/sec, burst 200
 	}
```

Apply in readPump:

```diff
 	for {
 		_, messageBytes, err := c.Conn.ReadMessage()
 		if err != nil {
 			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
 				c.Hub.logger.Error().Err(err).Msg("WebSocket read error")
 			}
 			break
 		}

+		// Rate limiting
+		if !c.limiter.Allow() {
+			c.Hub.logger.Warn().
+				Str("client_id", c.ID.String()).
+				Str("user_id", c.UserID.String()).
+				Msg("Client rate limit exceeded")
+			// Send error or disconnect
+			c.Conn.WriteMessage(websocket.CloseMessage,
+				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "rate limit exceeded"))
+			break
+		}
+
 		var msg Message
 		if err := json.Unmarshal(messageBytes, &msg); err != nil {
 			continue
 		}
```

---

## Bug #9: Logic Error - Duplicate Code in handleJoinGame and handleSpectate

**Severity:** Medium

**Description:**
The `handleSpectate` function (lines 383-420) duplicates almost all the logic from `handleJoinGame` (lines 168-207). The only difference is that `handleSpectate` always sets `IsSpectator=true` (line 404), while `handleJoinGame` determines it correctly based on `game.IsUserInGame()` (line 189). This causes:
1. Code duplication and maintenance burden
2. Potential for inconsistencies if one is updated but not the other
3. Confusion about which message type to use
4. Risk: if a player accidentally sends "spectate" instead of "join_game", they'll be incorrectly marked as a spectator even though they're playing

**Location:**
- File: `/home/user/openpairing/backend/internal/server/websocket.go`
- Functions: `handleJoinGame`, `handleSpectate`
- Lines: 168-207, 383-420

**Steps to Reproduce:**
1. User is White player in a game
2. Client accidentally sends "spectate" message instead of "join_game"
3. handleSpectate sets IsSpectator=true (line 404)
4. User is now marked as spectator even though they're the White player
5. This could cause UI issues or permission problems

**Expected Behavior:** Single function correctly handles both players and spectators
**Actual Behavior:** Duplicate code with potential for incorrect classification

**Fix:**
Remove `handleSpectate` and use only `handleJoinGame` (which already handles both cases):

```diff
 func (h *Hub) handleMessage(msg *Message) {
 	h.logger.Debug().
 		Str("type", msg.Type).
 		Msg("Handling message")

 	switch msg.Type {
 	case "join_game":
 		h.handleJoinGame(msg)
 	case "leave_game":
 		h.handleLeaveGame(msg)
 	case "make_move":
 		h.handleMakeMove(msg)
 	case "offer_draw":
 		h.handleOfferDraw(msg)
 	case "accept_draw":
 		h.handleAcceptDraw(msg)
 	case "decline_draw":
 		h.handleDeclineDraw(msg)
 	case "resign":
 		h.handleResign(msg)
 	case "abort":
 		h.handleAbort(msg)
-	case "spectate":
-		h.handleSpectate(msg)
+	case "spectate":
+		// Spectate is handled by join_game
+		h.handleJoinGame(msg)
 	}
 }
-
-func (h *Hub) handleSpectate(msg *Message) {
-	// ... remove entire function (lines 383-420)
-}
```

The `handleJoinGame` function already correctly determines if a user is a spectator at line 189, so no additional changes needed.

---

## Bug #10: Race Condition - getSpectatorCount Accesses Client Without Lock

**Severity:** Medium

**Description:**
The `getSpectatorCount` function acquires `Hub.mutex.RLock` (line 469) to safely iterate over the `gameClients` map, but then reads `client.IsSpectator` (line 475) without holding the client's mutex. Due to Bug #4, `IsSpectator` can be modified by other goroutines concurrently, causing a data race.

**Location:**
- File: `/home/user/openpairing/backend/internal/server/websocket.go`
- Function: `getSpectatorCount`
- Lines: 468-481

**Steps to Reproduce:**
1. Multiple clients connected to a game
2. One client sends "join_game" (writes IsSpectator at line 189)
3. Simultaneously, another goroutine calls getSpectatorCount (reads IsSpectator at line 475)
4. Run with `-race` flag
5. Expected: Data race detected
6. Actual: Inconsistent spectator counts

**Expected Behavior:** Read IsSpectator with proper synchronization
**Actual Behavior:** Unsynchronized read causes race condition

**Fix:**
This will be resolved by fixing Bug #4 (adding synchronization to Client fields). After implementing getter methods, update this function:

```diff
 func (h *Hub) getSpectatorCount(gameID uuid.UUID) int {
 	h.mutex.RLock()
 	defer h.mutex.RUnlock()

 	count := 0
 	if clients, ok := h.gameClients[gameID]; ok {
 		for client := range clients {
-			if client.IsSpectator {
+			if client.IsSpectator() {  // Use synchronized getter method
 				count++
 			}
 		}
 	}
 	return count
 }
```

---

## Bug #11: Resource Management - Channel Close Race in Unregister

**Severity:** Medium

**Description:**
The unregister handler closes the `client.Send` channel (line 122) while the `writePump` goroutine may still be trying to write to it. Although Go allows receiving from a closed channel, there's a timing issue:
1. Unregister closes Send channel (line 122)
2. WritePump is blocked on `case message, ok := <-c.Send` (line 582)
3. WritePump receives `ok=false` and writes CloseMessage (line 585)
4. However, if writePump is already in the middle of writing (lines 589-604), closing the channel prematurely could cause confusion

Additionally, the Send channel might be closed twice:
- Once in unregister (line 122)
- Once in writePump cleanup (line 577) via `c.Conn.Close()`

While closing a channel twice causes a panic, the defer in writePump (line 575-577) only closes the connection, not the channel. But there's still potential for issues if unregister closes Send while writePump is actively using it.

**Location:**
- File: `/home/user/openpairing/backend/internal/server/websocket.go`
- Functions: `Hub.Run` (unregister case), `Client.writePump`
- Lines: 106-128, 573-613

**Steps to Reproduce:**
1. Client connected and actively sending/receiving
2. WritePump is in the middle of draining queue (lines 595-600)
3. Unregister fires and closes Send channel (line 122)
4. WritePump tries to read from Send in line 599: `w.Write(<-c.Send)`
5. Receives nil or stale data from closed channel
6. Potential panic or data corruption

**Expected Behavior:** Graceful channel closure with proper coordination
**Actual Behavior:** Race between channel close and active writes

**Fix:**
Don't close the Send channel in unregister; let writePump own the channel lifecycle:

```diff
 		case client := <-h.unregister:
 			h.mutex.Lock()
 			if _, ok := h.clients[client]; ok {
 				delete(h.clients, client)
 				delete(h.userClients, client.UserID)

 				// Remove from game clients
 				if client.GameID != nil {
 					if gameClients, ok := h.gameClients[*client.GameID]; ok {
 						delete(gameClients, client)
 						if len(gameClients) == 0 {
 							delete(h.gameClients, *client.GameID)
 						}
 					}
 				}

-				close(client.Send)
+				// Don't close Send here; writePump will detect unregister
+				// via connection close and clean up
 			}
 			h.mutex.Unlock()
```

Update writePump to close the channel:

```diff
 func (c *Client) writePump() {
 	ticker := time.NewTicker(30 * time.Second)
 	defer func() {
 		ticker.Stop()
+		close(c.Send)  // Close the channel when writePump exits
 		c.Conn.Close()
 	}()
```

---

## Bug #12: Missing Error Handling - Nil Game After Accept/Resign

**Severity:** Low

**Description:**
In `handleAcceptDraw` (lines 304-320) and `handleResign` (lines 350-365), there's workaround code to handle the case where the game is nil after calling GameManager methods. The code logs a warning and broadcasts partial information. However, this indicates a deeper design issue:

The GameManager's `cleanupGame` method (lines 298-311 in manager.go) removes user entries from `userGames` but keeps the game in the `games` map intentionally (see comment on line 309). However, there's a separate cleanup goroutine that can remove games from the `games` map (lines 72-88 in manager.go). This creates a race condition window where:
1. AcceptDraw/Resign completes successfully
2. Cleanup goroutine runs and removes the game
3. WebSocket handler tries to GetGame
4. Returns nil

While the workaround handles this gracefully, it's a symptom of poor coordination between game lifecycle management and real-time notification.

**Location:**
- File: `/home/user/openpairing/backend/internal/server/websocket.go`
- Functions: `handleAcceptDraw`, `handleResign`
- Lines: 292-321, 338-366

**Steps to Reproduce:**
1. Start a game that's been running for 11 minutes
2. Player resigns
3. Between line 345 (Resign) and line 350 (GetGame), cleanup goroutine runs
4. Cleanup removes game because CompletedAt > 10 minutes ago
5. GetGame returns nil (line 350)
6. Warning logged, clients receive incomplete game_over message

**Expected Behavior:** Atomic game state retrieval and broadcast
**Actual Behavior:** Race condition between cleanup and state retrieval

**Fix:**
Modify GameManager to return the final state from Resign/AcceptDraw:

```go
// In chess/manager.go
func (gm *GameManager) Resign(gameID uuid.UUID, userID uuid.UUID) (*GameState, error) {
	game := gm.GetGame(gameID)
	if game == nil {
		return nil, ErrGameNotFound
	}

	if err := game.Resign(userID); err != nil {
		return nil, err
	}

	gm.logger.Info().
		Str("game_id", gameID.String()).
		Str("user", userID.String()).
		Str("result", game.Result).
		Msg("Player resigned")

	gm.cleanupGame(gameID)
	return game, nil  // Return the game state before potential cleanup
}
```

Then update WebSocket handlers:

```diff
 func (h *Hub) handleResign(msg *Message) {
 	if msg.Client.GameID == nil {
 		h.sendError(msg.Client, "not in a game")
 		return
 	}

 	gameID := *msg.Client.GameID
-	if err := h.gameManager.Resign(gameID, msg.Client.UserID); err != nil {
+	game, err := h.gameManager.Resign(gameID, msg.Client.UserID)
+	if err != nil {
 		h.sendError(msg.Client, err.Error())
 		return
 	}

-	game := h.gameManager.GetGame(gameID)
-	if game == nil {
-		// Game was cleaned up between Resign and GetGame
-		h.logger.Warn().Str("game_id", gameID.String()).Msg("Game not found after resignation")
-		h.broadcastToGame(gameID, "game_over", map[string]interface{}{
-			"termination": "resignation",
-		})
-		return
-	}
 	state := game.GetState()

 	h.broadcastToGame(gameID, "game_over", map[string]interface{}{
 		"result":      game.Result,
 		"termination": game.Termination,
 		"state":       state,
 	})
 }
```

Apply similar changes to `handleAcceptDraw` and other handlers.

---

## Statistics

- **Critical:** 2 bugs (Race condition in map iteration, CSRF vulnerability)
- **High:** 3 bugs (Multiple clients per user, unsynchronized client fields, resource leak on shutdown)
- **Medium:** 6 bugs (Spectator count logic, JSON marshal errors, duplicate code, spectator count race, channel close race, nil game handling)
- **Low:** 1 bug (Game state retrieval after operations)

---

## Recommendations

### Immediate Actions (Critical)
1. **Fix the CheckOrigin vulnerability** - This is a production security issue that must be addressed immediately
2. **Fix the broadcastToGame race condition** - This can cause panics under load
3. **Implement rate limiting** - Prevent DoS attacks via WebSocket flooding

### High Priority
4. **Add proper Client field synchronization** - Use the existing mutex or make fields atomic
5. **Handle multiple connections per user** - Either reject or properly track them
6. **Implement graceful shutdown** - Close all connections when context is cancelled

### Medium Priority
7. **Add comprehensive error logging** - Make debugging easier with proper error logs
8. **Consolidate join/spectate logic** - Remove duplicate code
9. **Fix spectator count broadcast logic** - Only broadcast when spectators change

### Code Quality Improvements
10. **Add integration tests** - Test concurrent scenarios with `-race` flag
11. **Add metrics** - Track connection count, message rate, error rate
12. **Document concurrency patterns** - Add comments explaining lock ordering and ownership
13. **Consider using channels for all client state changes** - This would eliminate many race conditions by serializing state changes through the Hub event loop

### Architectural Considerations
- Consider using a more robust WebSocket framework like Melody or Centrifugo
- Implement proper session management to handle multiple devices per user
- Add message queuing (Redis Streams) for guaranteed delivery
- Implement connection recovery and message replay for disconnections
- Add distributed tracing for debugging message flow
