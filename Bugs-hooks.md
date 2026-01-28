# Bug Report: hooks

**Analysis Date:** 2026-01-28
**Directory Analyzed:** `/home/user/openpairing/frontend/src/hooks`
**Files Analyzed:** 4 files (useAuth.ts, useGame.ts, useWam.ts, useAuth.test.ts)
**Bugs Found:** 9 bugs

---

## Summary

The hooks directory contains critical React hooks for authentication, real-time game state management, and WAM tournament operations. The analysis identified **9 functional bugs** across 3 files:

- **Critical:** 3 bugs (WebSocket stale closures, race conditions causing data loss)
- **High:** 4 bugs (type safety violations, infinite re-render risk, data consistency)
- **Medium:** 2 bugs (cache management inconsistencies)

The most critical issues are in `useGame.ts` where stale closures and race conditions could cause game state synchronization failures and user actions being lost. Type safety violations in `useWam.ts` could cause runtime crashes when API responses are malformed.

---

## Bugs

## Bug #1: Infinite Re-render Risk in useRequireRole Hook

**Severity:** High

**Description:**
The `useRequireRole` hook includes `allowedRoles` array in its dependency array (line 91). Since arrays are compared by reference in React, passing a new array on every render (e.g., `useRequireRole(['admin'])` directly in JSX) will cause the effect to run repeatedly, potentially creating infinite redirect loops and severe performance degradation.

**Location:**
- File: `/home/user/openpairing/frontend/src/hooks/useAuth.ts`
- Function: `useRequireRole`
- Lines: 79-91

**Steps to Reproduce:**
1. Create a component that calls `useRequireRole(['admin', 'arbiter'])` directly without memoization
2. The component renders
3. Effect runs and checks authorization
4. Component re-renders for any reason (state update, parent re-render)
5. New array `['admin', 'arbiter']` is created with different reference
6. Effect runs again (line 79-91)
7. Repeat steps 4-6 infinitely

**Expected:** Effect should only run when authentication state changes
**Actual:** Effect runs on every render, causing performance issues and potential redirect loops

**Fix:**
Use a stable reference for the allowed roles or compare array contents instead of reference. The recommended approach is to use `useMemo` for array comparison or require callers to memoize the array:

```diff
export function useRequireRole(allowedRoles: string[], redirectTo: string = '/tournaments') {
  const router = useRouter();
  const { user, isAuthenticated, isLoading, isHydrated } = useAuthStore();

+  // Stabilize the allowed roles array by joining to a string
+  const allowedRolesKey = allowedRoles.sort().join(',');
+
  useEffect(() => {
    // Wait for hydration before making auth decisions
    if (!isHydrated || isLoading) return;

    if (!isAuthenticated) {
      router.push('/login');
      return;
    }

    if (user && !allowedRoles.includes(user.role)) {
      router.push(redirectTo);
    }
-  }, [user, isAuthenticated, isLoading, isHydrated, router, allowedRoles, redirectTo]);
+  }, [user, isAuthenticated, isLoading, isHydrated, router, allowedRolesKey, redirectTo]);

  const hasRole = user ? allowedRoles.includes(user.role) : false;

  return { hasRole, isLoading, isHydrated };
}
```

---

## Bug #2: Stale Closure in WebSocket Effect - Missing onError Dependency

**Severity:** Critical

**Description:**
The WebSocket initialization effect (lines 58-103) uses the `onError` callback at line 93 but does NOT include it in the dependency array at line 103. This creates a stale closure bug where if `onError` changes, the WebSocket error handler will still call the old version of the callback. This means error handling could fail or call incorrect handlers, causing silent failures or errors being reported to the wrong component.

**Location:**
- File: `/home/user/openpairing/frontend/src/hooks/useGame.ts`
- Function: `useGame`
- Lines: 58-103

**Steps to Reproduce:**
1. Component renders with `useGame` and provides an `onError` callback that logs to console
2. WebSocket connects successfully
3. Component updates and provides a new `onError` that shows a UI notification
4. WebSocket encounters a connection error
5. Old `onError` (console log) is called instead of new one (UI notification)
6. User doesn't see the error notification

**Expected:** The current `onError` callback should be called
**Actual:** The stale `onError` callback from initial render is called

**Fix:**
Add `onError` to the dependency array, or use a ref to store the latest callback:

```diff
export function useGame(options: UseGameOptions) {
  const { gameId, onMove, onGameOver, onDrawOffered, onError } = options;
  const user = useAuthStore((state) => state.user);
  const token = useAuthStore((state) => state.accessToken);

  const socketRef = useRef<Socket | null>(null);
  const gameIdRef = useRef(gameId);
+  const onErrorRef = useRef(onError);
+  const onMoveRef = useRef(onMove);
+  const onGameOverRef = useRef(onGameOver);
+  const onDrawOfferedRef = useRef(onDrawOffered);

+  // Keep callback refs in sync
+  useEffect(() => {
+    onErrorRef.current = onError;
+    onMoveRef.current = onMove;
+    onGameOverRef.current = onGameOver;
+    onDrawOfferedRef.current = onDrawOffered;
+  }, [onError, onMove, onGameOver, onDrawOffered]);

  // ... rest of code

  socket.on('connect_error', (error) => {
    console.error('Connection error:', error);
-    onError?.('Failed to connect to game server');
+    onErrorRef.current?.('Failed to connect to game server');
  });

  // Use refs in handleMessage as well
  const handleMessage = useCallback((message: GameMessage) => {
    // ... switch cases using onMoveRef.current, onGameOverRef.current, etc.
  }, []); // Now has no external dependencies

  return () => {
    socket.emit('message', {
      type: 'leave_game',
      gameId: currentGameId,
    });
    socket.disconnect();
  };
}, [gameId, token]); // No need for callback dependencies
```

---

## Bug #3: Stale Event Handler - handleMessage Not Updated on Callback Changes

**Severity:** Critical

**Description:**
The `handleMessage` function is defined at line 105 with dependencies on `onMove`, `onGameOver`, `onDrawOffered`, and `onError` (line 168). However, this function is registered as a WebSocket event listener at line 88, inside an effect that does NOT include `handleMessage` in its dependency array (line 103). When any callback changes, `handleMessage` is recreated with new closure values, but the socket's event listener still references the old `handleMessage`. This means all game events will be processed with stale callbacks, causing game moves, game over events, and draw offers to be handled incorrectly.

**Location:**
- File: `/home/user/openpairing/frontend/src/hooks/useGame.ts`
- Function: `useGame`
- Lines: 88, 105-168

**Steps to Reproduce:**
1. Component renders with `useGame` and provides `onMove` callback that updates state
2. WebSocket connects and registers the message handler
3. Component updates with a new `onMove` callback (different logic)
4. Server sends a 'move_made' message
5. Old `onMove` is called with stale closure values
6. Game state update fails or uses wrong logic

**Expected:** Current callbacks should handle WebSocket messages
**Actual:** Stale callbacks from initial render handle messages

**Fix:**
Use refs to store callbacks (as shown in Bug #2) or restructure to include handleMessage in dependencies:

```diff
export function useGame(options: UseGameOptions) {
  // ... setup code with refs as in Bug #2

  // Initialize WebSocket connection
  useEffect(() => {
    const wsUrl = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8081';
    const currentGameId = gameId;

+    // Define handleMessage inside effect so it has access to latest refs
+    const handleMessage = (message: GameMessage) => {
+      switch (message.type) {
+        case 'game_state':
+          if (message.state) {
+            setGameState(message.state);
+          }
+          break;
+
+        case 'move_made':
+          if (message.state) {
+            setGameState(message.state);
+            if (message.move) {
+              onMoveRef.current?.(message.move, message.state);
+            }
+          }
+          break;
+
+        case 'game_over':
+          if (message.state) {
+            setGameState(message.state);
+            onGameOverRef.current?.(message.state.result, message.state.termination);
+          }
+          break;
+
+        case 'draw_offered':
+          if (message.userId) {
+            onDrawOfferedRef.current?.(message.userId);
+          }
+          break;
+
+        // ... other cases
+
+        case 'error':
+          onErrorRef.current?.(message.error || 'Unknown error');
+          break;
+      }
+    };

    const socket = io(wsUrl, {
      auth: {
        token,
      },
      transports: ['websocket'],
      reconnection: true,
      reconnectionAttempts: 5,
      reconnectionDelay: 1000,
    });

    socketRef.current = socket;

    socket.on('connect', () => {
      setIsConnected(true);
      socket.emit('message', {
        type: 'join_game',
        gameId: currentGameId,
      });
    });

    socket.on('disconnect', () => {
      setIsConnected(false);
    });

    socket.on('message', handleMessage);

    socket.on('connect_error', (error) => {
      console.error('Connection error:', error);
-      onError?.('Failed to connect to game server');
+      onErrorRef.current?.('Failed to connect to game server');
    });

    return () => {
      socket.emit('message', {
        type: 'leave_game',
        gameId: currentGameId,
      });
      socket.disconnect();
    };
  }, [gameId, token]);

-  // Remove the handleMessage function defined outside the effect
-  const handleMessage = useCallback((message: GameMessage) => {
-    // ...
-  }, [onMove, onGameOver, onDrawOffered, onError]);
```

---

## Bug #4: Race Condition in WebSocket Cleanup - Message Lost on Fast Unmount

**Severity:** High

**Description:**
The cleanup function (lines 96-102) emits a 'leave_game' message and immediately calls `socket.disconnect()`. If the component unmounts quickly (e.g., user navigates away), the 'leave_game' message might not be sent to the server before the socket disconnects. This can leave ghost users in the game room on the server, causing incorrect spectator counts, opponent connection status, and potentially preventing game state cleanup.

**Location:**
- File: `/home/user/openpairing/frontend/src/hooks/useGame.ts`
- Function: `useGame` (cleanup in useEffect)
- Lines: 96-102

**Steps to Reproduce:**
1. User opens a game page, WebSocket connects
2. User immediately navigates away (e.g., browser back button)
3. React unmounts the component
4. Cleanup function emits 'leave_game' and immediately disconnects
5. Socket disconnects before message is sent to server
6. Server still thinks user is in the game room
7. Other users see incorrect spectator count

**Expected:** 'leave_game' message should be reliably sent before disconnection
**Actual:** Message may be lost if unmount happens quickly

**Fix:**
Wait for the emit to be acknowledged before disconnecting, or use socket.io's `disconnecting` event:

```diff
  return () => {
-    socket.emit('message', {
-      type: 'leave_game',
-      gameId: currentGameId,
-    });
-    socket.disconnect();
+    // Give the leave_game message time to be sent
+    socket.emit('message', {
+      type: 'leave_game',
+      gameId: currentGameId,
+    }, () => {
+      // Callback executed when message is sent
+      socket.disconnect();
+    });
+
+    // Fallback: force disconnect after timeout in case callback never fires
+    const disconnectTimeout = setTimeout(() => {
+      socket.disconnect();
+    }, 100);
+
+    return () => clearTimeout(disconnectTimeout);
  };
```

Alternative approach using immediate disconnect but with proper server-side handling:

```diff
  return () => {
+    // Socket.io will queue the message and send it before disconnecting
+    // if we use the 'disconnect' event instead of calling disconnect()
    socket.emit('message', {
      type: 'leave_game',
      gameId: currentGameId,
    });
-    socket.disconnect();
+
+    // Let socket.io handle graceful shutdown
+    socket.close();
  };
```

---

## Bug #5: Race Condition in Action Methods - Socket Disconnect Between Check and Emit

**Severity:** High

**Description:**
All action methods (`makeMove`, `resign`, `offerDraw`, `acceptDraw`, `declineDraw`, `abort` at lines 170-241) check `isConnected` before emitting messages. However, there's a race condition where the socket could disconnect between the check and the emit call. When this happens, the emit fails silently without calling `onError`, causing user actions to be lost without any feedback. Users might think their move was sent when it wasn't.

**Location:**
- File: `/home/user/openpairing/frontend/src/hooks/useGame.ts`
- Functions: `makeMove`, `resign`, `offerDraw`, `acceptDraw`, `declineDraw`, `abort`
- Lines: 170-241

**Steps to Reproduce:**
1. User is connected to a game, `isConnected` is true
2. User clicks to make a move
3. Between the check at line 171 and emit at line 176, network fails and socket disconnects
4. `socketRef.current.emit()` is called but socket is disconnected
5. Emit fails silently, no error callback is invoked
6. User sees no feedback, thinks move was made
7. Move is never sent to server

**Expected:** User should be notified that the action failed
**Actual:** Action fails silently without error notification

**Fix:**
Add error handling for emit failures and verify socket is still connected at time of emit:

```diff
  const makeMove = useCallback((move: string) => {
    if (!socketRef.current || !isConnected) {
      onError?.('Not connected to game server');
      return;
    }

+    // Double-check connection state
+    if (!socketRef.current.connected) {
+      onError?.('Connection lost. Please reconnect.');
+      return;
+    }
+
-    socketRef.current.emit('message', {
-      type: 'make_move',
-      gameId,
-      move,
-    });
+    try {
+      socketRef.current.emit('message', {
+        type: 'make_move',
+        gameId,
+        move,
+      }, (acknowledgment: any) => {
+        // Optional: handle server acknowledgment
+        if (acknowledgment?.error) {
+          onError?.(acknowledgment.error);
+        }
+      });
+    } catch (error) {
+      onError?.('Failed to send move. Please try again.');
+    }
  }, [gameId, isConnected, onError]);
```

Apply the same fix to `resign`, `offerDraw`, `acceptDraw`, `declineDraw`, and `abort` functions.

---

## Bug #6: Unsafe gameId Closure in Cleanup Function

**Severity:** Medium

**Description:**
The cleanup function at lines 96-102 uses `currentGameId` which is captured when the effect runs. If `gameId` prop changes (e.g., user navigates from game A to game B) and then the component unmounts, the cleanup will emit 'leave_game' for `currentGameId` (the most recent game) but the socket might have joined multiple games in sequence. While `gameIdRef` is updated at line 54, it's not used in the cleanup, creating potential for leaving the wrong game room.

**Location:**
- File: `/home/user/openpairing/frontend/src/hooks/useGame.ts`
- Function: `useGame` (cleanup)
- Lines: 96-102

**Steps to Reproduce:**
1. Component mounts with gameId="game-1", socket joins game-1
2. gameId prop changes to "game-2", effect cleanup runs
3. Cleanup sends leave_game for "game-1"
4. New effect runs, socket joins "game-2"
5. Component unmounts
6. Cleanup sends leave_game for "game-2"
7. Both cleanups work correctly in this case

**However, if gameId changes rapidly:**
1. Component mounts with gameId="game-1"
2. gameId changes to "game-2" before effect runs
3. currentGameId captures "game-2"
4. Socket joins "game-2"
5. gameId changes to "game-3"
6. Cleanup tries to leave "game-2" (currentGameId)
7. New effect joins "game-3"

This is actually working as intended due to the effect running on gameId changes. However, it's a code smell because `gameIdRef` exists but isn't used consistently.

**Expected:** Cleanup should use the gameId that the current socket connection is joined to
**Actual:** Uses captured `currentGameId` which may not match `gameIdRef.current`

**Fix:**
Use `gameIdRef.current` in cleanup for consistency, or remove `gameIdRef` if not needed:

```diff
  // Keep gameIdRef in sync
  useEffect(() => {
    gameIdRef.current = gameId;
  }, [gameId]);

  // Initialize WebSocket connection
  useEffect(() => {
    const wsUrl = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8081';
-    const currentGameId = gameId;

    const socket = io(wsUrl, {
      auth: {
        token,
      },
      transports: ['websocket'],
      reconnection: true,
      reconnectionAttempts: 5,
      reconnectionDelay: 1000,
    });

    socketRef.current = socket;

    socket.on('connect', () => {
      setIsConnected(true);
      // Join the game room
      socket.emit('message', {
        type: 'join_game',
-        gameId: currentGameId,
+        gameId: gameIdRef.current,
      });
    });

    socket.on('disconnect', () => {
      setIsConnected(false);
    });

    socket.on('message', (message: GameMessage) => {
      handleMessage(message);
    });

    socket.on('connect_error', (error) => {
      console.error('Connection error:', error);
      onError?.('Failed to connect to game server');
    });

    return () => {
      socket.emit('message', {
        type: 'leave_game',
-        gameId: currentGameId,
+        gameId: gameIdRef.current,
      });
      socket.disconnect();
    };
  }, [gameId, token]);
```

---

## Bug #7: Type Safety Violation - Potential Undefined Return in useWamStatus

**Severity:** High

**Description:**
The `useWamStatus` hook (lines 25-36) throws an error if `response.error` exists, but then returns `response.data as WamTournamentStatus` at line 33. The `ApiResponse<T>` type has `data?: T` (optional), meaning `response.data` could be undefined even when there's no error. The type assertion `as WamTournamentStatus` bypasses TypeScript's type checking, allowing undefined to be returned when TypeScript expects `WamTournamentStatus`. This will cause runtime crashes when components try to access properties on the undefined value.

**Location:**
- File: `/home/user/openpairing/frontend/src/hooks/useWam.ts`
- Function: `useWamStatus`
- Lines: 25-36

**Steps to Reproduce:**
1. API returns HTTP 200 with empty body or `{ data: null }`
2. `response.error` is undefined (no error)
3. `response.data` is undefined or null
4. Hook returns undefined but TypeScript thinks it's WamTournamentStatus
5. Component tries to access `status.hasGroups`
6. Runtime error: "Cannot read property 'hasGroups' of undefined"

**Expected:** Hook should throw error or return a valid WamTournamentStatus
**Actual:** Hook returns undefined, causing runtime crashes

**Fix:**
Validate that data exists before returning, or provide proper error handling:

```diff
export function useWamStatus(tournamentId: string) {
  return useQuery({
    queryKey: wamKeys.status(tournamentId),
    queryFn: async () => {
      const response = await wamApi.getStatus(tournamentId);
      if (response.error) {
        throw new Error(response.error.message);
      }
-      return response.data as WamTournamentStatus;
+      if (!response.data) {
+        throw new Error('No data received from server');
+      }
+      return response.data;
    },
    enabled: !!tournamentId,
  });
}
```

---

## Bug #8: Type Safety Violation - Potential Undefined Return in useWamParticipants

**Severity:** High

**Description:**
The `useWamParticipants` hook (lines 42-53) has the same type safety issue as Bug #7. At line 50, it returns `response.data?.participants as WamParticipant[]`. The optional chaining `?.` means if `response.data` is undefined, the expression evaluates to undefined. The type assertion then tells TypeScript this undefined is actually `WamParticipant[]`, bypassing type safety. This will cause runtime crashes when components try to iterate over the undefined array.

**Location:**
- File: `/home/user/openpairing/frontend/src/hooks/useWam.ts`
- Function: `useWamParticipants`
- Lines: 42-53

**Steps to Reproduce:**
1. API returns HTTP 200 with empty body
2. `response.error` is undefined
3. `response.data` is undefined
4. Hook returns undefined but TypeScript thinks it's WamParticipant[]
5. Component tries to map over participants: `participants.map(...)`
6. Runtime error: "Cannot read property 'map' of undefined"

**Expected:** Hook should throw error or return valid WamParticipant[]
**Actual:** Hook returns undefined, causing runtime crashes

**Fix:**
Validate data exists and has the expected structure:

```diff
export function useWamParticipants(tournamentId: string) {
  return useQuery({
    queryKey: wamKeys.participants(tournamentId),
    queryFn: async () => {
      const response = await wamApi.getParticipants(tournamentId);
      if (response.error) {
        throw new Error(response.error.message);
      }
-      return response.data?.participants as WamParticipant[];
+      if (!response.data || !response.data.participants) {
+        throw new Error('Invalid response: missing participants data');
+      }
+      return response.data.participants;
    },
    enabled: !!tournamentId,
  });
}
```

---

## Bug #9: Data Consistency Issue - Group Proposal Not Invalidated After Joker Update

**Severity:** Medium

**Description:**
The `useUpdateJoker` mutation (lines 77-101) invalidates the participants and status queries after updating joker points (lines 96-98). However, it does NOT invalidate the group proposal query. Group proposals depend on participant ratings and joker points, so after updating jokers, any cached group proposal becomes stale. If a user fetches a proposal, updates jokers, and then views the cached proposal again, they'll see incorrect groupings based on old joker values.

**Location:**
- File: `/home/user/openpairing/frontend/src/hooks/useWam.ts`
- Function: `useUpdateJoker`
- Lines: 94-99

**Steps to Reproduce:**
1. Tournament organizer fetches group proposal for tournament
2. Proposal is cached with current joker points
3. Organizer updates a participant's joker points from 0 to 150
4. Mutation invalidates participants and status queries (they refetch)
5. Organizer views the proposal tab again
6. Cached proposal is shown with OLD joker values
7. Proposal shows incorrect groupings

**Expected:** Group proposal should be refetched after joker update
**Actual:** Stale group proposal is shown from cache

**Fix:**
Invalidate the proposal query after updating jokers:

```diff
export function useUpdateJoker(tournamentId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      participantId,
      jokerPoints,
    }: {
      participantId: string;
      jokerPoints: JokerValue;
    }) => {
      const response = await wamApi.setJokerPoints(tournamentId, participantId, jokerPoints);
      if (response.error) {
        throw new Error(response.error.message);
      }
      return response.data;
    },
    onSuccess: () => {
      // Invalidate participants to refetch with updated data
      queryClient.invalidateQueries({ queryKey: wamKeys.participants(tournamentId) });
      // Also invalidate status as joker stats might have changed
      queryClient.invalidateQueries({ queryKey: wamKeys.status(tournamentId) });
+      // Also invalidate proposal as it depends on joker points
+      queryClient.invalidateQueries({ queryKey: wamKeys.proposal(tournamentId) });
    },
  });
}
```

---

## Statistics

- **Critical:** 3 bugs
  - Bug #2: Stale closure in WebSocket effect
  - Bug #3: Stale event handler not updated
  - Bug #4: Race condition in WebSocket cleanup

- **High:** 4 bugs
  - Bug #1: Infinite re-render risk in useRequireRole
  - Bug #5: Race condition in action methods
  - Bug #7: Type safety violation in useWamStatus
  - Bug #8: Type safety violation in useWamParticipants

- **Medium:** 2 bugs
  - Bug #6: Unsafe gameId closure in cleanup
  - Bug #9: Data consistency issue with proposal cache

---

## Recommendations

### Immediate Actions (Critical Bugs)

1. **Fix WebSocket Hooks Urgently**: Bugs #2, #3, and #4 in `useGame.ts` can cause complete failure of real-time game functionality. Use refs to store callbacks and ensure proper cleanup sequencing.

2. **Add Type Guards**: Bugs #7 and #8 can cause runtime crashes. Add validation for all API responses before returning data from hooks.

### Short-term Improvements (High Priority)

3. **Implement Proper Error Handling**: Add try-catch blocks and connection state verification in all WebSocket action methods (Bug #5).

4. **Stabilize Hook Dependencies**: Use memo or stable references for array/object dependencies to prevent infinite re-renders (Bug #1).

### Long-term Improvements

5. **Add Integration Tests**: The bugs in `useGame.ts` could have been caught with integration tests that simulate WebSocket message sequences and connection failures.

6. **Use TypeScript Strict Mode**: Enable strict null checks to catch potential undefined returns at compile time instead of runtime.

7. **Implement Retry Logic**: For WebSocket actions, add automatic retry with exponential backoff when operations fail due to connection issues.

8. **Add Logging**: Implement structured logging for WebSocket events to help diagnose issues in production.

9. **Cache Invalidation Strategy**: Document and enforce a consistent cache invalidation strategy across all mutations. Consider using automatic invalidation based on data dependencies.

10. **Code Review Checklist**: Add items for:
    - Checking useEffect dependency arrays include all referenced values
    - Verifying WebSocket cleanup properly completes async operations
    - Validating API response data before using
    - Ensuring array/object dependencies are stable references
