# Bug Report: result

**Analysis Date:** 2026-01-28
**Directory Analyzed:** `/home/user/openpairing/backend/internal/result`
**Files Analyzed:** 6 files
**Bugs Found:** 12 bugs

---

## Summary

The result module handles chess game result entry, correction, and deletion with participant statistics tracking. Analysis revealed **12 functional bugs** including 2 critical security vulnerabilities (JSON injection), 2 critical data integrity issues (bulk operations without transactions, incorrect stats calculation), and several medium/low severity bugs related to validation, error handling, and edge cases.

**Most Critical Issues:**
1. JSON injection vulnerability in audit log formatting (could corrupt audit trail)
2. Bulk result entry lacks transactional atomicity (creates inconsistent tournament state)
3. Recalculation query incorrectly counts double forfeits as losses
4. Missing validation allows invalid result types for bye games

---

## Bugs

## Bug #1: JSON Injection Vulnerability in Audit Log Formatting

**Severity:** CRITICAL

**Description:**
The `formatAuditNewValue` and `formatAuditOldValue` functions in `repository.go` manually construct JSON strings using `fmt.Sprintf` without escaping special characters. The `audit.Reason` field contains user input (required 10-500 characters) that could include quotes, newlines, or other special JSON characters. This creates invalid JSON and potentially allows injection of malicious content into audit logs stored in the database.

**Location:**
- File: `/home/user/openpairing/backend/internal/result/repository.go`
- Function: `formatAuditNewValue`, `formatAuditOldValue`
- Lines: 458-467, 448-456

**Steps to Reproduce:**
1. Submit a result correction with reason: `Test reason with "quotes" and \n newlines`
2. The audit log will store: `{"result":"1-0","whiteScore":1.0,"blackScore":0.0,"reason":"Test reason with "quotes" and \n newlines"}`
3. Expected: Valid JSON with properly escaped characters
4. Actual: Invalid JSON structure that breaks parsing

**Impact:**
- Corrupted audit logs that cannot be parsed
- Potential for JSON injection if audit logs are displayed in web UI
- Legal compliance issues if audit trail is compromised
- Database integrity issues with JSONB columns

**Fix:**
Replace manual string concatenation with `json.Marshal` for type-safe JSON generation:

```diff
+import "encoding/json"
+
+type auditValue struct {
+	Result     string  `json:"result,omitempty"`
+	WhiteScore float64 `json:"whiteScore,omitempty"`
+	BlackScore float64 `json:"blackScore,omitempty"`
+	Reason     string  `json:"reason,omitempty"`
+}
+
 func formatAuditOldValue(audit *ResultAudit) interface{} {
 	if audit.OldResult == nil {
 		return nil
 	}
-	return fmt.Sprintf(`{"result":"%s","whiteScore":%v,"blackScore":%v}`,
-		*audit.OldResult,
-		floatPtrToString(audit.OldWhiteScore),
-		floatPtrToString(audit.OldBlackScore))
+	val := auditValue{
+		Result:     *audit.OldResult,
+		WhiteScore: floatPtrToFloat(audit.OldWhiteScore),
+		BlackScore: floatPtrToFloat(audit.OldBlackScore),
+	}
+	jsonBytes, _ := json.Marshal(val)
+	return string(jsonBytes)
 }

 func formatAuditNewValue(audit *ResultAudit) interface{} {
 	if audit.NewResult == nil {
 		return nil
 	}
-	return fmt.Sprintf(`{"result":"%s","whiteScore":%v,"blackScore":%v,"reason":"%s"}`,
-		*audit.NewResult,
-		floatPtrToString(audit.NewWhiteScore),
-		floatPtrToString(audit.NewBlackScore),
-		stringPtrToValue(audit.Reason))
+	val := auditValue{
+		Result:     *audit.NewResult,
+		WhiteScore: floatPtrToFloat(audit.NewWhiteScore),
+		BlackScore: floatPtrToFloat(audit.NewBlackScore),
+		Reason:     stringPtrToValue(audit.Reason),
+	}
+	jsonBytes, _ := json.Marshal(val)
+	return string(jsonBytes)
 }
+
+func floatPtrToFloat(f *float64) float64 {
+	if f == nil {
+		return 0.0
+	}
+	return *f
+}
```

---

## Bug #2: Bulk Result Entry Lacks Transactional Atomicity

**Severity:** CRITICAL

**Description:**
The `EnterBulkResults` function processes multiple game results by calling `EnterResult` in a loop. Each `EnterResult` call uses its own separate transaction. If result entry #5 fails, results #1-4 are already committed to the database. This creates an inconsistent tournament state where some games have results and others don't, affecting standings calculations, round completion status, and pairing for the next round.

**Location:**
- File: `/home/user/openpairing/backend/internal/result/service.go`
- Function: `EnterBulkResults`
- Lines: 334-348

**Steps to Reproduce:**
1. Create a tournament with round 1 having 10 games
2. Submit bulk results with 10 entries where entry #6 has an invalid game ID
3. Observe that games 1-5 have results saved
4. Game 6 fails with "game not found"
5. Games 7-10 are never processed
6. Expected: All 10 results saved together or none at all
7. Actual: Partial results saved, round in inconsistent state

**Impact:**
- Tournament standings show incorrect/incomplete results
- Round completion status is wrong (5/10 complete instead of 0/10 or 10/10)
- Next round pairing cannot start due to incomplete round
- Manual intervention required to fix data consistency
- User confusion about which results were saved

**Fix:**
Wrap the entire bulk operation in a single database transaction:

```diff
 func (s *Service) EnterBulkResults(ctx context.Context, bulk *BulkResultEntry) ([]ResultEntryResponse, []error) {
+	// Start a single transaction for all results
+	tx, err := s.repo.BeginTx(ctx)
+	if err != nil {
+		return nil, []error{fmt.Errorf("failed to start transaction: %w", err)}
+	}
+	defer tx.Rollback(ctx)
+
 	var responses []ResultEntryResponse
 	var errors []error

 	for _, entry := range bulk.Results {
-		resp, err := s.EnterResult(ctx, &entry)
+		// Validate first
+		validation := s.ValidateResult(ctx, &entry)
+		if !validation.Valid {
+			errors = append(errors, fmt.Errorf("game %s: %v", entry.GameID, validation.Errors))
+			continue
+		}
+
+		// Get game
+		game, err := s.repo.GetGame(ctx, entry.GameID)
 		if err != nil {
 			errors = append(errors, fmt.Errorf("game %s: %w", entry.GameID, err))
-		} else {
-			responses = append(responses, *resp)
+			continue
 		}
+
+		// Check existing result
+		if game.Result != nil && *game.Result != "" {
+			errors = append(errors, fmt.Errorf("game %s: already has a result", entry.GameID))
+			continue
+		}
+
+		// Parse scores
+		whiteScore, blackScore := ParseResult(entry.Result)
+
+		// Update game result (within transaction)
+		err = s.repo.UpdateResultWithTx(ctx, tx, entry.GameID, string(entry.Result), whiteScore, blackScore, entry.EnteredBy)
+		if err != nil {
+			errors = append(errors, fmt.Errorf("game %s: failed to update result: %w", entry.GameID, err))
+			continue
+		}
+
+		// Update participant stats (within transaction)
+		if game.WhiteParticipantID != nil {
+			err = s.updateParticipantStatsWithTx(ctx, tx, *game.WhiteParticipantID, whiteScore)
+			if err != nil {
+				errors = append(errors, fmt.Errorf("game %s: failed to update white stats: %w", entry.GameID, err))
+				continue
+			}
+		}
+		if game.BlackParticipantID != nil {
+			err = s.updateParticipantStatsWithTx(ctx, tx, *game.BlackParticipantID, blackScore)
+			if err != nil {
+				errors = append(errors, fmt.Errorf("game %s: failed to update black stats: %w", entry.GameID, err))
+				continue
+			}
+		}
+
+		// Update round count (within transaction)
+		err = s.repo.UpdateRoundCompletedCountWithTx(ctx, tx, game.RoundID)
+		if err != nil {
+			errors = append(errors, fmt.Errorf("game %s: failed to update round count: %w", entry.GameID, err))
+			continue
+		}
+
+		responses = append(responses, ResultEntryResponse{
+			GameID:          entry.GameID,
+			Result:          string(entry.Result),
+			WhiteScore:      whiteScore,
+			BlackScore:      blackScore,
+			StandingsUpdate: true,
+		})
 	}

+	// If any errors occurred, rollback everything
+	if len(errors) > 0 {
+		return nil, errors
+	}
+
+	// Commit all changes together
+	if err := tx.Commit(ctx); err != nil {
+		return nil, []error{fmt.Errorf("failed to commit bulk results: %w", err)}
+	}
+
+	// Create audit logs after commit (non-critical)
+	for _, resp := range responses {
+		resultStr := resp.Result
+		audit := &ResultAudit{
+			GameID:        resp.GameID,
+			Action:        "entered",
+			NewResult:     &resultStr,
+			NewWhiteScore: &resp.WhiteScore,
+			NewBlackScore: &resp.BlackScore,
+			PerformedBy:   bulk.Results[0].EnteredBy, // Assume same user
+		}
+		_ = s.repo.CreateAuditLog(ctx, audit)
+	}
+
 	return responses, errors
 }
```

---

## Bug #3: RecalculateParticipantStats Incorrectly Counts Double Forfeits as Losses

**Severity:** HIGH

**Description:**
The `RecalculateParticipantStats` SQL query counts a participant's losses as any game where their score is 0. However, double forfeits (result "0-0F") give both players a score of 0, but neither player should be credited with a loss. The current query incorrectly counts double forfeits as losses for both participants, inflating loss statistics.

**Location:**
- File: `/home/user/openpairing/backend/internal/result/repository.go`
- Function: `RecalculateParticipantStats`
- Lines: 255-259

**Steps to Reproduce:**
1. Create a tournament with 2 participants
2. Create game 1: Participant A vs B, result "0-0F" (double forfeit)
3. Create game 2: Participant A vs C, result "0-1" (A loses normally)
4. Call RecalculateParticipantStats for Participant A
5. Expected: A has 0 wins, 0 draws, 1 loss (from game 2)
6. Actual: A has 0 wins, 0 draws, 2 losses (both games counted)

**Impact:**
- Participant statistics show inflated loss counts
- Affects tournament standings and tiebreak calculations
- Inaccurate performance metrics displayed to users
- Statistical reports are incorrect

**Fix:**
Modify the loss calculation to only count games where the opponent scored 1:

```diff
 SELECT
     $1::uuid as participant_id,
     SUM(CASE
         WHEN white_participant_id = $1 THEN COALESCE(white_score, 0)
         WHEN black_participant_id = $1 THEN COALESCE(black_score, 0)
         ELSE 0
     END) as total_points,
     SUM(CASE
         WHEN (white_participant_id = $1 AND white_score = 1) OR
              (black_participant_id = $1 AND black_score = 1) THEN 1
         ELSE 0
     END) as wins,
     SUM(CASE
         WHEN (white_participant_id = $1 AND white_score = 0.5) OR
              (black_participant_id = $1 AND black_score = 0.5) THEN 1
         ELSE 0
     END) as draws,
     SUM(CASE
-        WHEN (white_participant_id = $1 AND white_score = 0) OR
-             (black_participant_id = $1 AND black_score = 0) THEN 1
+        WHEN (white_participant_id = $1 AND white_score = 0 AND black_score = 1) OR
+             (black_participant_id = $1 AND black_score = 0 AND white_score = 1) THEN 1
         ELSE 0
     END) as losses
 FROM games
 WHERE (white_participant_id = $1 OR black_participant_id = $1)
   AND result IS NOT NULL
```

This ensures losses are only counted when the opponent scored a full point (1.0).

---

## Bug #4: Missing Validation for Normal Results on Bye Games

**Severity:** MEDIUM

**Description:**
The `ValidateResult` function checks that bye results cannot be entered for games with two players, but it doesn't check the reverse: normal results (1-0, 0-1, 1/2-1/2) cannot be entered for bye games where one participant is NULL. This allows entering "1-0" for a game where BlackParticipantID is NULL, which is semantically incorrect.

**Location:**
- File: `/home/user/openpairing/backend/internal/result/service.go`
- Function: `ValidateResult`
- Lines: 274-311

**Steps to Reproduce:**
1. Create a bye game with WhiteParticipantID set but BlackParticipantID = NULL
2. Submit ResultEntry with Result = "1-0" (white wins)
3. Validation passes (no error)
4. Result is saved as "White defeats nobody"
5. Expected: Validation error "cannot enter match result for bye game"
6. Actual: Result is accepted and saved

**Impact:**
- Semantically incorrect results in database
- Confusion in tournament management interface
- Bye games appear as "wins against opponent" instead of byes
- Affects tournament result exports (PGN, TRF files)

**Fix:**
Add validation to check if the game is a bye game when a normal result is entered:

```diff
 func (s *Service) ValidateResult(ctx context.Context, entry *ResultEntry) *ResultValidation {
     validation := &ResultValidation{Valid: true}

     // Check if game exists
     game, err := s.repo.GetGame(ctx, entry.GameID)
     if err != nil {
         validation.Valid = false
         validation.Errors = append(validation.Errors, "game not found")
         return validation
     }

     // Check if result is valid
     if !IsValidResult(string(entry.Result)) {
         validation.Valid = false
         validation.Errors = append(validation.Errors, "invalid result value")
     }

     // Warn if game already has a result
     if game.Result != nil && *game.Result != "" {
         validation.Warnings = append(validation.Warnings, "game already has a result")
     }

     // Check for bye result with two players
     if entry.Result == ResultBye {
         if game.WhiteParticipantID != nil && game.BlackParticipantID != nil {
             validation.Valid = false
             validation.Errors = append(validation.Errors, "cannot enter bye result for game with two players")
         }
     }
+
+    // Check for normal result on bye game
+    isByeGame := game.WhiteParticipantID == nil || game.BlackParticipantID == nil
+    isNormalResult := entry.Result == ResultWhiteWins || entry.Result == ResultBlackWins ||
+                      entry.Result == ResultDraw || entry.Result == ResultWhiteForfeit ||
+                      entry.Result == ResultBlackForfeit
+    if isByeGame && isNormalResult {
+        validation.Valid = false
+        validation.Errors = append(validation.Errors, "cannot enter match result for bye game, use BYE result")
+    }

     // Check for forfeit results
     if entry.Result == ResultWhiteForfeit || entry.Result == ResultBlackForfeit {
         validation.Warnings = append(validation.Warnings, "forfeit result entered")
     }

     return validation
 }
```

---

## Bug #5: Player Name Concatenation Creates Empty Space Instead of NULL

**Severity:** MEDIUM

**Description:**
The `GetRoundGames` SQL query uses `COALESCE(first_name, '') || ' ' || COALESCE(last_name, '')` to build player names. When both first_name and last_name are NULL (e.g., for bye games or unregistered players), this produces a single space character `" "` instead of NULL. This empty space is then treated as a valid player name in the application.

**Location:**
- File: `/home/user/openpairing/backend/internal/result/repository.go`
- Function: `GetRoundGames`
- Lines: 312-313

**Steps to Reproduce:**
1. Create a bye game where WhiteParticipantID is NULL
2. Call GetRoundGames for that round
3. Observe that WhitePlayerName is `" "` (a single space character)
4. Expected: WhitePlayerName should be NULL or empty string
5. Actual: WhitePlayerName is a space, displayed as a blank but non-empty field in UI

**Impact:**
- UI displays empty space as player name for bye games
- JSON responses include `"whitePlayerName": " "` instead of null
- Frontend code must handle this edge case specially
- Inconsistent with GetGameWithPlayers which may handle this differently

**Fix:**
Use NULLIF to convert empty strings to NULL after concatenation:

```diff
 SELECT
     g.id,
     g.tournament_id,
     g.round_id,
     tr.round_number,
     g.board_number,
     g.white_participant_id,
     g.black_participant_id,
-    COALESCE(wp.first_name, '') || ' ' || COALESCE(wp.last_name, ''),
-    COALESCE(bp.first_name, '') || ' ' || COALESCE(bp.last_name, ''),
+    NULLIF(TRIM(COALESCE(wp.first_name, '') || ' ' || COALESCE(wp.last_name, '')), ''),
+    NULLIF(TRIM(COALESCE(bp.first_name, '') || ' ' || COALESCE(bp.last_name, '')), ''),
     wtp.starting_rating,
     btp.starting_rating,
     g.result,
     g.white_score,
     g.black_score,
     g.status,
     g.result_entered_by,
     g.result_entered_at,
     g.created_at
 FROM games g
```

The `TRIM` removes leading/trailing spaces, and `NULLIF(..., '')` converts empty strings to NULL.

---

## Bug #6: Float Comparison in Switch Statements for Score Calculations

**Severity:** MEDIUM

**Description:**
The `updateParticipantStatsWithTx` and `reverseParticipantStatsWithTx` functions use switch statements with exact float comparisons (`case 1.0:`, `case 0.5:`, `case 0.0:`). Due to floating-point precision issues, a score calculated as `0.5` might actually be stored as `0.5000000000001` or `0.4999999999999`, causing the switch to fall through to the default case. This would update points but not win/draw/loss counts, creating inconsistent statistics.

**Location:**
- File: `/home/user/openpairing/backend/internal/result/service.go`
- Functions: `updateParticipantStatsWithTx`, `reverseParticipantStatsWithTx`
- Lines: 367-374, 390-397

**Steps to Reproduce:**
1. Perform a series of floating-point calculations that result in score = 0.4999999999999
2. Call updateParticipantStatsWithTx with this score
3. The switch statement doesn't match any case
4. winsDelta, drawsDelta, lossesDelta all remain 0
5. Expected: drawsDelta = 1, pointsDelta = 5
6. Actual: All deltas = 0 except pointsDelta = 4 (due to int conversion truncation)

**Impact:**
- Participant win/draw/loss counts become inconsistent with point totals
- Statistics show 5.0 points but 0 wins/draws/losses
- Tiebreak calculations fail
- Requires manual stats recalculation to fix

**Fix:**
Replace switch statements with tolerance-based comparisons:

```diff
 func (s *Service) updateParticipantStatsWithTx(ctx context.Context, q Querier, participantID uuid.UUID, score float64) error {
     var winsDelta, drawsDelta, lossesDelta int
     pointsDelta := int(score * 10) // Convert to tenths for integer arithmetic

-    switch score {
-    case 1.0:
+    const epsilon = 0.001
+    if score > 1.0-epsilon && score <= 1.0 {
         winsDelta = 1
-    case 0.5:
+    } else if score > 0.5-epsilon && score < 0.5+epsilon {
         drawsDelta = 1
-    case 0.0:
+    } else if score < epsilon {
         lossesDelta = 1
+    } else {
+        // Unexpected score value
+        return fmt.Errorf("unexpected score value: %f (expected 0.0, 0.5, or 1.0)", score)
     }

     if q != nil {
         return s.repo.UpdateParticipantStatsWithTx(ctx, q, participantID, pointsDelta, winsDelta, drawsDelta, lossesDelta)
     }
     return s.repo.UpdateParticipantStats(ctx, participantID, pointsDelta, winsDelta, drawsDelta, lossesDelta)
 }

 func (s *Service) reverseParticipantStatsWithTx(ctx context.Context, q Querier, participantID uuid.UUID, score float64) error {
     var winsDelta, drawsDelta, lossesDelta int
     pointsDelta := -int(score * 10) // Negative to reverse

-    switch score {
-    case 1.0:
+    const epsilon = 0.001
+    if score > 1.0-epsilon && score <= 1.0 {
         winsDelta = -1
-    case 0.5:
+    } else if score > 0.5-epsilon && score < 0.5+epsilon {
         drawsDelta = -1
-    case 0.0:
+    } else if score < epsilon {
         lossesDelta = -1
+    } else {
+        return fmt.Errorf("unexpected score value: %f (expected 0.0, 0.5, or 1.0)", score)
     }

     if q != nil {
         return s.repo.UpdateParticipantStatsWithTx(ctx, q, participantID, pointsDelta, winsDelta, drawsDelta, lossesDelta)
     }
     return s.repo.UpdateParticipantStats(ctx, participantID, pointsDelta, winsDelta, drawsDelta, lossesDelta)
 }
```

---

## Bug #7: Missing Validation for Positive Round Numbers

**Severity:** MEDIUM

**Description:**
The `GetRoundGames` and `GetRoundStatus` handlers parse the round number from URL parameters using `strconv.Atoi` but don't validate that the round number is positive. Negative or zero round numbers are accepted and passed to the database query, which may return empty results but provides no clear error message to the user.

**Location:**
- File: `/home/user/openpairing/backend/internal/result/handler.go`
- Functions: `GetRoundGames`, `GetRoundStatus`
- Lines: 210-215, 244-249

**Steps to Reproduce:**
1. Call GET `/tournaments/{id}/rounds/-5/games`
2. The handler accepts roundNumber = -5
3. Database query returns 0 games
4. Handler returns 200 OK with empty array
5. Expected: 400 Bad Request "invalid round number: must be positive"
6. Actual: 200 OK with empty results, confusing the caller

**Impact:**
- API returns misleading success responses for invalid inputs
- Clients cannot distinguish between "round has no games" vs "invalid round"
- Debugging issues with round numbers is harder
- API contract is unclear about valid round number range

**Fix:**
Add validation for positive round numbers:

```diff
 func (h *Handler) GetRoundGames(c *gin.Context) {
     tournamentIDStr := c.Param("id")
     tournamentID, err := uuid.Parse(tournamentIDStr)
     if err != nil {
         c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
         return
     }

     roundStr := c.Param("round")
     roundNumber, err := strconv.Atoi(roundStr)
     if err != nil {
         c.JSON(http.StatusBadRequest, gin.H{"error": "invalid round number"})
         return
     }
+    if roundNumber < 1 {
+        c.JSON(http.StatusBadRequest, gin.H{"error": "round number must be positive"})
+        return
+    }

     games, err := h.service.GetRoundGames(c.Request.Context(), tournamentID, roundNumber)
     // ... rest of function
 }

 func (h *Handler) GetRoundStatus(c *gin.Context) {
     tournamentIDStr := c.Param("id")
     tournamentID, err := uuid.Parse(tournamentIDStr)
     if err != nil {
         c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
         return
     }

     roundStr := c.Param("round")
     roundNumber, err := strconv.Atoi(roundStr)
     if err != nil {
         c.JSON(http.StatusBadRequest, gin.H{"error": "invalid round number"})
         return
     }
+    if roundNumber < 1 {
+        c.JSON(http.StatusBadRequest, gin.H{"error": "round number must be positive"})
+        return
+    }

     status, err := h.service.GetRoundStatus(c.Request.Context(), tournamentID, roundNumber)
     // ... rest of function
 }
```

---

## Bug #8: ClearResult Always Sets Status to 'scheduled'

**Severity:** MEDIUM

**Description:**
The `ClearResultWithTx` function always sets the game status to 'scheduled' when clearing a result. However, if a game was 'ongoing' (started but not finished), clearing its result should set it back to 'ongoing', not 'scheduled'. This loses information about whether the game had been started.

**Location:**
- File: `/home/user/openpairing/backend/internal/result/repository.go`
- Function: `ClearResultWithTx`
- Lines: 194-206

**Steps to Reproduce:**
1. Create a game with status 'ongoing' (game in progress)
2. Enter a result (status changes to 'completed')
3. Delete the result using DeleteResult
4. Game status is now 'scheduled'
5. Expected: Status should be 'ongoing' (game was started)
6. Actual: Status is 'scheduled' (game appears never started)

**Impact:**
- Loss of information about game progress
- Games that were in progress appear as never started
- Tournament directors lose track of which games were played
- Affects tournament timeline and reporting

**Fix:**
Store the original status before result entry, or use a more nuanced status transition:

```diff
 func (r *Repository) ClearResultWithTx(ctx context.Context, q Querier, gameID uuid.UUID) error {
+    // Get current game to check if we should restore to 'ongoing' or 'scheduled'
+    var hadPGN bool
+    err := q.QueryRow(ctx, `SELECT pgn IS NOT NULL FROM games WHERE id = $1`, gameID).Scan(&hadPGN)
+    if err != nil {
+        return fmt.Errorf("failed to check game state: %w", err)
+    }
+
+    // If game has PGN data, it was ongoing; otherwise it was just scheduled
+    newStatus := "scheduled"
+    if hadPGN {
+        newStatus = "ongoing"
+    }
+
-    _, err := q.Exec(ctx, `
+    _, err = q.Exec(ctx, `
         UPDATE games
         SET result = NULL, white_score = NULL, black_score = NULL,
             result_entered_by = NULL, result_entered_at = NULL,
-            status = 'scheduled'
+            status = $2
         WHERE id = $1
-    `, gameID)
+    `, gameID, newStatus)
     if err != nil {
         return fmt.Errorf("failed to clear result: %w", err)
     }
     return nil
 }
```

Alternatively, add a `previous_status` field to track the state before completion.

---

## Bug #9: Bulk Results Handler Returns 200 OK Even When All Entries Fail

**Severity:** LOW

**Description:**
The `EnterBulkResults` handler always returns HTTP 200 OK, even if all result entries failed. The response includes error counts, but the HTTP status code doesn't reflect the failure. This violates HTTP semantics where 200 OK should indicate success. RFC 7231 suggests using 207 Multi-Status for batch operations with mixed success/failure.

**Location:**
- File: `/home/user/openpairing/backend/internal/result/handler.go`
- Function: `EnterBulkResults`
- Lines: 327-333

**Steps to Reproduce:**
1. Submit bulk results with 10 entries, all with invalid game IDs
2. All 10 entries fail with "game not found"
3. Handler returns: `200 OK {"successful": [], "errors": [...10 errors...], "successCount": 0, "errorCount": 10}`
4. Expected: 400 Bad Request or 207 Multi-Status
5. Actual: 200 OK despite complete failure

**Impact:**
- Clients cannot use HTTP status code to determine success
- Middleware/monitoring that checks status codes sees false positives
- REST API best practices violated
- Client code must parse response body to determine actual result

**Fix:**
Return appropriate HTTP status based on success/failure ratio:

```diff
 func (h *Handler) EnterBulkResults(c *gin.Context) {
     var bulk BulkResultEntry
     if err := c.ShouldBindJSON(&bulk); err != nil {
         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
         return
     }

     // Get user ID from context (set by auth middleware)
     userIDStr, exists := c.Get("user_id")
     if exists {
         var enteredBy uuid.UUID
         if uid, ok := userIDStr.(uuid.UUID); ok {
             enteredBy = uid
         } else if uidStr, ok := userIDStr.(string); ok {
             if uid, err := uuid.Parse(uidStr); err == nil {
                 enteredBy = uid
             }
         }
         for i := range bulk.Results {
             bulk.Results[i].EnteredBy = enteredBy
         }
     }

     responses, errors := h.service.EnterBulkResults(c.Request.Context(), &bulk)

     // Build response with both successes and failures
     errorStrings := make([]string, len(errors))
     for i, err := range errors {
         errorStrings[i] = err.Error()
     }

-    c.JSON(http.StatusOK, gin.H{
+    // Determine appropriate status code
+    statusCode := http.StatusOK
+    if len(errors) > 0 {
+        if len(responses) == 0 {
+            // All failed
+            statusCode = http.StatusBadRequest
+        } else {
+            // Partial success - use Multi-Status
+            statusCode = http.StatusMultiStatus // 207
+        }
+    }
+
+    c.JSON(statusCode, gin.H{
         "successful":   responses,
         "errors":       errorStrings,
         "totalCount":   len(bulk.Results),
         "successCount": len(responses),
         "errorCount":   len(errors),
     })
 }
```

---

## Bug #10: Missing user_id in Context Creates Zero-Value UUID in Audit Trail

**Severity:** LOW

**Description:**
The handlers extract `user_id` from the Gin context but don't handle the case where it doesn't exist or fails to parse. When `user_id` is missing, the `EnteredBy`, `CorrectedBy`, or `deletedBy` fields remain at their zero value (all-zeros UUID: `00000000-0000-0000-0000-000000000000`). This invalid UUID is then stored in audit logs, making it impossible to determine who performed the action.

**Location:**
- File: `/home/user/openpairing/backend/internal/result/handler.go`
- Functions: `EnterResult`, `CorrectResult`, `DeleteResult`
- Lines: 40-49, 83-92, 136-146

**Steps to Reproduce:**
1. Call POST `/results` without authentication middleware (or with broken auth)
2. The `user_id` key doesn't exist in context
3. `entry.EnteredBy` remains `uuid.UUID{}` (zero UUID)
4. Result is saved with zero UUID as performer
5. Expected: 401 Unauthorized or 400 Bad Request "user not authenticated"
6. Actual: Result saved with invalid performer ID in audit trail

**Impact:**
- Audit trail shows invalid user IDs (all zeros)
- Cannot trace who performed actions
- Compliance issues with audit requirements
- Legal disputes cannot be resolved with audit logs

**Fix:**
Validate that user_id exists and is valid before proceeding:

```diff
 func (h *Handler) EnterResult(c *gin.Context) {
     var entry ResultEntry
     if err := c.ShouldBindJSON(&entry); err != nil {
         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
         return
     }

     // Get user ID from context (set by auth middleware)
-    userIDStr, exists := c.Get("user_id")
+    userIDRaw, exists := c.Get("user_id")
-    if exists {
-        if uid, ok := userIDStr.(uuid.UUID); ok {
-            entry.EnteredBy = uid
-        } else if uidStr, ok := userIDStr.(string); ok {
-            if uid, err := uuid.Parse(uidStr); err == nil {
-                entry.EnteredBy = uid
-            }
-        }
+    if !exists {
+        c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
+        return
+    }
+
+    var userID uuid.UUID
+    if uid, ok := userIDRaw.(uuid.UUID); ok {
+        userID = uid
+    } else if uidStr, ok := userIDRaw.(string); ok {
+        var err error
+        userID, err = uuid.Parse(uidStr)
+        if err != nil {
+            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID in context"})
+            return
+        }
+    } else {
+        c.JSON(http.StatusInternalServerError, gin.H{"error": "user ID has invalid type in context"})
+        return
     }
+
+    entry.EnteredBy = userID

     response, err := h.service.EnterResult(c.Request.Context(), &entry)
     // ... rest of function
 }
```

Apply the same fix to `CorrectResult` and `DeleteResult`.

---

## Bug #11: Error Handling Uses Brittle String Comparison

**Severity:** LOW

**Description:**
Multiple handlers check for specific errors by comparing `err.Error()` against hardcoded strings like "game not found" or "game has no result to delete". This is brittle because any change to the error message in the service/repository layer breaks the error detection. Go best practices recommend using sentinel errors or error type checking.

**Location:**
- File: `/home/user/openpairing/backend/internal/result/handler.go`
- Functions: `EnterResult`, `CorrectResult`, `DeleteResult`, `GetGame`
- Lines: 53-56, 96-99, 150-152, 181-184

**Steps to Reproduce:**
1. Change error message in service.go from "game not found" to "game with ID %s not found"
2. Handler on line 53 checks `if err.Error() == "game not found"`
3. Condition no longer matches
4. Handler returns 400 Bad Request instead of 404 Not Found
5. Expected: Error type/sentinel error detection still works
6. Actual: Error detection breaks with message changes

**Impact:**
- Fragile error handling that breaks with message changes
- Wrong HTTP status codes returned to clients
- Difficult to maintain and refactor error messages
- Violates Go error handling best practices

**Fix:**
Define sentinel errors and use `errors.Is()` for checking:

```diff
+// Define sentinel errors in service.go
+package result
+
+import "errors"
+
+var (
+	ErrGameNotFound = errors.New("game not found")
+	ErrGameAlreadyHasResult = errors.New("game already has a result")
+	ErrGameHasNoResult = errors.New("game has no result to delete")
+	ErrResultMismatch = errors.New("current result does not match expected old result")
+)
+
+// Update service methods to return sentinel errors
 func (s *Service) EnterResult(ctx context.Context, entry *ResultEntry) (*ResultEntryResponse, error) {
     game, err := s.repo.GetGame(ctx, entry.GameID)
     if err != nil {
-        return nil, fmt.Errorf("failed to get game: %w", err)
+        if err == pgx.ErrNoRows {
+            return nil, ErrGameNotFound
+        }
+        return nil, fmt.Errorf("failed to get game: %w", err)
     }

     if game.Result != nil && *game.Result != "" {
-        return nil, fmt.Errorf("game already has a result, use correction workflow to change it")
+        return nil, ErrGameAlreadyHasResult
     }
     // ... rest of function
 }

+// Update handlers to use errors.Is()
 func (h *Handler) EnterResult(c *gin.Context) {
     // ... setup code ...

     response, err := h.service.EnterResult(c.Request.Context(), &entry)
     if err != nil {
-        if err.Error() == "game not found" {
+        if errors.Is(err, result.ErrGameNotFound) {
             c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
             return
         }
+        if errors.Is(err, result.ErrGameAlreadyHasResult) {
+            c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
+            return
+        }
         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
         return
     }
     // ... rest of function
 }
```

---

## Bug #12: Missing Proper Logging in Service Layer

**Severity:** LOW

**Description:**
The service layer uses `fmt.Printf` for logging warnings (e.g., when audit log creation fails after committing results). This prints directly to stdout/stderr instead of using a structured logger. According to the project's CLAUDE.md, the backend uses zerolog for logging. Using `fmt.Printf` means these warnings won't be captured by centralized logging, lack structured fields, and cannot be filtered by log level.

**Location:**
- File: `/home/user/openpairing/backend/internal/result/service.go`
- Functions: `EnterResult`, `CorrectResult`, `DeleteResult`
- Lines: 94, 189, 268

**Steps to Reproduce:**
1. Trigger an audit log creation failure (e.g., database connection drops after commit)
2. Observe output: `warning: failed to create audit log: <error>`
3. Check centralized logs (e.g., ELK stack, CloudWatch)
4. Expected: Structured log entry with context fields
5. Actual: Warning not captured, just printed to stdout

**Impact:**
- Audit log failures not visible in production monitoring
- No alerting on audit log issues
- Debugging is harder without structured log context
- Inconsistent with project's logging standards

**Fix:**
Add zerolog logger to Service and use structured logging:

```diff
+import (
+	"github.com/rs/zerolog"
+)
+
 // Service provides result management functionality
 type Service struct {
-	repo *Repository
+	repo   *Repository
+	logger zerolog.Logger
 }

 // NewService creates a new result service
-func NewService(repo *Repository) *Service {
-	return &Service{repo: repo}
+func NewService(repo *Repository, logger zerolog.Logger) *Service {
+	return &Service{
+		repo:   repo,
+		logger: logger.With().Str("component", "result-service").Logger(),
+	}
 }

 func (s *Service) EnterResult(ctx context.Context, entry *ResultEntry) (*ResultEntryResponse, error) {
     // ... transaction and result entry code ...

     // Create audit log (outside transaction - non-critical)
     newResult := string(entry.Result)
     audit := &ResultAudit{
         GameID:        entry.GameID,
         Action:        "entered",
         NewResult:     &newResult,
         NewWhiteScore: &whiteScore,
         NewBlackScore: &blackScore,
         Reason:        entry.Notes,
         PerformedBy:   entry.EnteredBy,
     }
     err = s.repo.CreateAuditLog(ctx, audit)
     if err != nil {
-        // Log but don't fail
-        fmt.Printf("warning: failed to create audit log: %v\n", err)
+        s.logger.Warn().
+            Err(err).
+            Str("game_id", entry.GameID.String()).
+            Str("action", "entered").
+            Msg("failed to create audit log")
     }

     return &ResultEntryResponse{
         // ... response
     }, nil
 }
```

Apply similar changes to `CorrectResult` and `DeleteResult` functions.

---

## Statistics

- **Critical:** 2 bugs
- **High:** 1 bug
- **Medium:** 5 bugs
- **Low:** 4 bugs

---

## Recommendations

### Immediate Actions (Critical/High Priority):
1. **Fix JSON injection vulnerability** - Deploy Bug #1 fix immediately to prevent audit log corruption
2. **Add transaction to bulk operations** - Bug #2 must be fixed before bulk entry is used in production
3. **Fix stats recalculation query** - Bug #3 affects data integrity; run corrected query on existing data

### Short-term Improvements (Medium Priority):
4. Implement comprehensive input validation for all result types and game states
5. Replace all float equality comparisons with epsilon-based tolerance checks
6. Add positive number validation for all integer inputs from API
7. Standardize game status transitions with a state machine pattern

### Long-term Enhancements (Low Priority):
8. Implement proper HTTP status codes following REST best practices (RFC 7231)
9. Migrate to sentinel errors throughout the codebase for robust error handling
10. Integrate zerolog structured logging in all service layer functions
11. Add integration tests for bulk operations and transaction rollback scenarios
12. Implement idempotency keys for result entry to prevent duplicate submissions

### Code Quality:
- Add more comprehensive test coverage for edge cases (double forfeits, bye games, floating-point precision)
- Document the state transition diagram for game status (scheduled → ongoing → completed)
- Create API documentation specifying exact error codes for each endpoint
- Add validation middleware to check user authentication before handler execution
- Consider implementing an event sourcing pattern for result changes to improve audit trail
