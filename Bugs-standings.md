# Bug Report: standings

**Analysis Date:** 2026-01-28
**Directory Analyzed:** `/home/user/openpairing/backend/internal/standings`
**Files Analyzed:** 5 files
**Bugs Found:** 8 bugs

---

## Summary

The standings module handles tournament standings calculations, tiebreakers, crosstables, and player progress tracking. Analysis revealed 8 functional bugs across error handling, data integrity, security, and missing functionality. The most critical issues include:

- **Security**: Missing authorization checks on all endpoints
- **Data Integrity**: Performance rating calculation incorrectly includes unrated opponents (rating 0)
- **Missing Functionality**: Player progress API returns empty round ranks array
- **Error Handling**: Fragile string-based error matching and silently ignored errors

Most bugs are Medium to High severity, with one Critical security issue and several High-priority data integrity problems.

---

## Bugs

## Bug #1: Fragile Error Handling Using String Comparison

**Severity:** Medium

**Description:**
The handler uses string comparison (`err.Error() == "tournament not found"`) to identify specific error types for HTTP status code mapping. This is fragile and breaks if error messages change. If the repository changes the error message wording, the handler will incorrectly return 500 Internal Server Error instead of 404 Not Found, causing poor API contracts and debugging nightmares.

**Location:**
- File: `/home/user/openpairing/backend/internal/standings/handler.go`
- Functions: `GetStandings`, `GetCrosstable`, `RecalculateStandings`, `GetPlayerProgress`
- Lines: 40, 71, 102, 141

**Steps to Reproduce:**
1. Change the error message in repository from "tournament not found" to "tournament does not exist"
2. Call the API endpoint `/tournaments/{id}/standings` with an invalid tournament ID
3. Expected: HTTP 404 Not Found
4. Actual: HTTP 500 Internal Server Error (because string comparison fails)

**Fix:**
Use typed errors with `errors.Is()` or create custom error types:

```diff
// In repository.go
+var ErrTournamentNotFound = errors.New("tournament not found")
+var ErrParticipantNotFound = errors.New("participant not found")

 if err == pgx.ErrNoRows {
-    return nil, fmt.Errorf("tournament not found")
+    return nil, ErrTournamentNotFound
 }

// In handler.go
+import "errors"

 standings, err := h.service.GetStandings(c.Request.Context(), id)
 if err != nil {
-    if err.Error() == "tournament not found" {
+    if errors.Is(err, ErrTournamentNotFound) {
         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
         return
     }
```

---

## Bug #2: Silently Ignored Errors in GetStandings

**Severity:** High

**Description:**
In the `GetStandings` method, errors from `GetSpecialRankings` and `GetParticipantEligibilityData` are silently ignored using the blank identifier `_`. If these operations fail (e.g., database connection issues), the standings will be calculated without special ranking badges, and users won't know why their age-group or women's rankings are missing. This can cause confusion in tournaments where these rankings are critical.

**Location:**
- File: `/home/user/openpairing/backend/internal/standings/service.go`
- Function: `GetStandings`
- Lines: 45-46

**Steps to Reproduce:**
1. Set up a tournament with special rankings (e.g., U12, Women)
2. Cause a database error in the `special_rankings` table (permissions, corruption, etc.)
3. Call `GetStandings` for the tournament
4. Expected: Error returned indicating special rankings couldn't be loaded
5. Actual: Standings returned successfully but without any eligible rankings badges

**Fix:**
Handle errors properly, at minimum log them, or return partial data with warnings:

```diff
 // Get special rankings and eligibility data for badges
-specialRankings, _ := s.repo.GetSpecialRankings(ctx, tournamentID)
-eligibilityData, _ := s.repo.GetParticipantEligibilityData(ctx, tournamentID)
+specialRankings, err := s.repo.GetSpecialRankings(ctx, tournamentID)
+if err != nil {
+    // Log the error but continue - special rankings are optional
+    // logger.Warn().Err(err).Msg("failed to get special rankings")
+    specialRankings = []SpecialRankingData{}
+}
+eligibilityData, err := s.repo.GetParticipantEligibilityData(ctx, tournamentID)
+if err != nil {
+    // logger.Warn().Err(err).Msg("failed to get eligibility data")
+    eligibilityData = make(map[uuid.UUID]ParticipantEligibilityData)
+}
```

Note: Inconsistency exists - `GetFilteredStandings` (line 310) properly returns an error if `GetParticipantEligibilityData` fails, but `GetStandings` ignores it.

---

## Bug #3: Missing RoundRanks Calculation in Player Progress

**Severity:** High

**Description:**
The `GetPlayerProgress` function initializes the `RoundRanks` slice with the correct size but never populates it with actual rank data. It remains as all zeros, which is incorrect. The API contract promises to return the player's rank after each round, but this critical feature is completely missing. This breaks any frontend visualization of rank progression over time.

**Location:**
- File: `/home/user/openpairing/backend/internal/standings/service.go`
- Function: `GetPlayerProgress`
- Lines: 529-589 (specifically line 530 initialization, but no subsequent population)

**Steps to Reproduce:**
1. Create a tournament with multiple rounds and participants
2. Call `/tournaments/{id}/participants/{participantId}/progress`
3. Observe the response JSON
4. Expected: `roundRanks: [3, 2, 1, 1, 1]` (rank after each round)
5. Actual: `roundRanks: [0, 0, 0, 0, 0]` (all zeros)

**Fix:**
Calculate standings after each round to determine the player's rank:

```diff
 progress := &PlayerProgress{
     ParticipantID: participantID,
     PlayerName:    participant.PlayerName,
     RoundPoints:   make([]float64, info.TotalRounds),
     RoundRanks:    make([]int, info.TotalRounds),
 }

-// Calculate round-by-round points
+// Calculate round-by-round points and ranks
 cumulativePoints := 0.0
+
+// Build a map of all participants' cumulative points per round
+participantPointsByRound := make(map[uuid.UUID][]float64)
+for _, p := range participants {
+    participantPointsByRound[p.ID] = make([]float64, info.TotalRounds)
+}
+
 for round := 1; round <= info.TotalRounds; round++ {
+    // Calculate points for all participants up to this round
+    for _, p := range participants {
+        roundPoints := 0.0
+        for _, g := range games {
+            if g.RoundNumber <= round {
+                if g.WhiteParticipantID != nil && *g.WhiteParticipantID == p.ID && g.WhiteScore != nil {
+                    roundPoints += *g.WhiteScore
+                } else if g.BlackParticipantID != nil && *g.BlackParticipantID == p.ID && g.BlackScore != nil {
+                    roundPoints += *g.BlackScore
+                }
+            }
+        }
+        participantPointsByRound[p.ID][round-1] = roundPoints
+    }
+
+    // Calculate rank for our participant in this round
+    roundPoints := participantPointsByRound[participantID][round-1]
+    rank := 1
+    for _, p := range participants {
+        if participantPointsByRound[p.ID][round-1] > roundPoints {
+            rank++
+        }
+    }
+    progress.RoundRanks[round-1] = rank
+
+    // Continue with existing opponent logic
     for _, g := range games {
         if g.RoundNumber == round {
             if g.WhiteParticipantID != nil && *g.WhiteParticipantID == participantID {
                 if g.WhiteScore != nil {
                     cumulativePoints += *g.WhiteScore
                 }
                 // ... rest of existing code
```

---

## Bug #4: Manual JSON Parsing Instead of json.Unmarshal

**Severity:** Medium

**Description:**
The `parseTiebreaks` function manually parses JSON array strings instead of using Go's standard `json.Unmarshal`. This custom parser is fragile and fails to handle standard JSON variations like whitespace, escaped characters, or formatted JSON. While `parseScoringSettings` correctly uses `json.Unmarshal`, tiebreaks uses an ad-hoc parser.

**Location:**
- File: `/home/user/openpairing/backend/internal/standings/repository.go`
- Function: `parseTiebreaks`
- Lines: 459-492

**Steps to Reproduce:**
1. Store tiebreaks JSON in database with formatting: `["buchholz_cut1" , "buchholz"]` (extra spaces)
2. Or with newlines: `["buchholz_cut1",\n"buchholz"]`
3. Or with escaped quotes in a string (edge case): `["buchholz_\"special\""]`
4. Call `GetStandings` which calls `GetTournamentInfo`
5. Expected: Tiebreaks parsed correctly
6. Actual: Parser fails silently or returns incorrect results

**Fix:**
Use standard JSON unmarshaling:

```diff
 func parseTiebreaks(data []byte) []string {
-    // Simple parsing - data is like ["buchholz_cut1","buchholz","sonneborn_berger"]
-    s := string(data)
-    if len(s) < 2 {
+    if len(data) == 0 {
         return nil
     }
-    s = s[1 : len(s)-1] // Remove [ and ]
-    if s == "" {
+
+    var result []string
+    if err := json.Unmarshal(data, &result); err != nil {
+        // Log error and return empty array
+        // logger.Error().Err(err).Msg("failed to parse tiebreaks JSON")
         return nil
     }
-
-    var result []string
-    current := ""
-    inQuote := false
-    for _, c := range s {
-        switch c {
-        case '"':
-            inQuote = !inQuote
-        case ',':
-            if !inQuote && current != "" {
-                result = append(result, current)
-                current = ""
-            }
-        default:
-            if inQuote {
-                current += string(c)
-            }
-        }
-    }
-    if current != "" {
-        result = append(result, current)
-    }
+
     return result
 }
```

---

## Bug #5: RecalculateStandings Overwrites All Tiebreaks With Three Hardcoded Values

**Severity:** Medium

**Description:**
The `RecalculateStandings` function only updates three hardcoded tiebreak types (buchholz, buchholz_cut1, sonneborn_berger) in the database, regardless of which tiebreaks the tournament is configured to use. If a tournament uses other tiebreaks (AROC, Koya, Progress, etc.), those values won't be persisted. Worse, if those three tiebreaks aren't in the calculated standings map, the code stores 0.0 values, potentially overwriting previously correct data.

**Location:**
- File: `/home/user/openpairing/backend/internal/standings/service.go`
- Function: `RecalculateStandings`
- Lines: 482-490

**Steps to Reproduce:**
1. Create a tournament configured to use tiebreaks: ["progress", "aroc", "koya"]
2. Call `RecalculateStandings` endpoint
3. Check database `tournament_participants` table
4. Expected: Progress, AROC, and Koya values updated
5. Actual: buchholz, buchholz_cut1, sonneborn_berger set to 0.0 (not in calculated map)
6. The tournament's actual tiebreaks are not persisted at all

**Fix:**
Update database schema to store all tiebreaks generically or update only the tiebreaks configured for the tournament:

```diff
 func (s *Service) RecalculateStandings(ctx context.Context, tournamentID uuid.UUID) error {
     standings, err := s.GetStandings(ctx, tournamentID)
     if err != nil {
         return fmt.Errorf("failed to get standings: %w", err)
     }

+    // Get tournament info to know which tiebreaks are configured
+    info, err := s.repo.GetTournamentInfo(ctx, tournamentID)
+    if err != nil {
+        return fmt.Errorf("failed to get tournament info: %w", err)
+    }
+
     // Update each participant's tiebreaks in the database
     for _, standing := range standings.Standings {
-        buchholz := standing.Tiebreaks["buchholz"]
-        buchholzCut1 := standing.Tiebreaks["buchholz_cut1"]
-        sonnebornBerger := standing.Tiebreaks["sonneborn_berger"]
-
-        err := s.repo.UpdateParticipantTiebreaks(ctx, standing.ParticipantID, buchholz, buchholzCut1, sonnebornBerger)
+        // Only update the tiebreaks defined in the tournament configuration
+        // Alternatively, redesign UpdateParticipantTiebreaks to accept map[string]float64
+        tiebreakValues := make(map[string]float64)
+        for _, tbCode := range info.Tiebreaks {
+            if val, ok := standing.Tiebreaks[tbCode]; ok {
+                tiebreakValues[tbCode] = val
+            }
+        }
+
+        err := s.repo.UpdateParticipantTiebreaksMap(ctx, standing.ParticipantID, tiebreakValues)
         if err != nil {
             return fmt.Errorf("failed to update tiebreaks for participant %s: %w", standing.ParticipantID, err)
         }
     }

     return nil
 }
```

Note: This also requires updating the repository method to handle dynamic tiebreak columns or using a JSONB column.

---

## Bug #6: Missing BYE Handling for Black Player in Crosstable

**Severity:** Low

**Description:**
The `convertResultForBlack` function handles draw, win, and loss results but doesn't handle the "BYE" result. The symmetric function `convertResultForWhite` does handle BYE (returns "+"). While byes are typically assigned to white by convention, there's no enforcement of this in the code. If a black player somehow receives a bye in the database, it would return `nil` instead of the expected "+", breaking crosstable display.

**Location:**
- File: `/home/user/openpairing/backend/internal/standings/repository.go`
- Function: `convertResultForBlack`
- Lines: 547-564

**Steps to Reproduce:**
1. Manually insert a game with: black_participant_id = player1, white_participant_id = NULL, result = "BYE"
2. Call `/tournaments/{id}/crosstable`
3. Expected: Player's cell shows "+" for bye
4. Actual: Player's cell shows null or empty (result is nil)

**Fix:**
Add BYE handling to match white's behavior:

```diff
 func convertResultForBlack(result *string) *string {
     if result == nil {
         return nil
     }
     switch *result {
     case "0-1", "0-1F":
         s := "1"
         return &s
     case "1-0", "1-0F":
         s := "0"
         return &s
     case "1/2-1/2":
         s := "="
         return &s
+    case "BYE":
+        s := "+"
+        return &s
     default:
         return nil
     }
 }
```

---

## Bug #7: Performance Rating Calculation Includes Unrated Opponents as Rating 0

**Severity:** High

**Description:**
The performance rating calculation incorrectly includes opponents with rating 0 in the average opponent rating calculation. In chess, a rating of 0 means "unrated," not "rated at 0 strength." Including these pulls the average down drastically, giving players inflated performance ratings. For example, a player beating a 2000-rated opponent and an unrated opponent would get average opponent rating of (2000 + 0) / 2 = 1000, when it should be 2000 / 1 = 2000.

**Location:**
- File: `/home/user/openpairing/backend/internal/standings/repository.go`
- Line: 215 (`COALESCE(opp_tp.starting_rating, 0)`)
- File: `/home/user/openpairing/backend/internal/standings/service.go`
- Function: `calculatePerformanceRating`
- Lines: 638-658

**Steps to Reproduce:**
1. Create a tournament with 2 participants: Player A (rated 2000), Player B (unrated/rating NULL)
2. Player A wins against Player B
3. Call `/tournaments/{id}/standings`
4. Expected: Player A's performance rating ≈ 2000 + 400 = 2400 (perfect score vs 2000-rated opponent)
5. Actual: Performance rating ≈ 1000 + 400 = 1400 (because opponent average includes 0)

**Fix:**
Exclude unrated opponents from performance calculation:

```diff
 // In repository.go GetOpponentRatings
 func (r *Repository) GetOpponentRatings(ctx context.Context, tournamentID, participantID uuid.UUID) ([]int, error) {
     rows, err := r.db.Query(ctx, `
-        SELECT COALESCE(opp_tp.starting_rating, 0)
+        SELECT opp_tp.starting_rating
         FROM games g
         JOIN tournament_participants opp_tp ON
             (g.white_participant_id = $2 AND opp_tp.id = g.black_participant_id) OR
             (g.black_participant_id = $2 AND opp_tp.id = g.white_participant_id)
-        WHERE g.tournament_id = $1 AND g.result IS NOT NULL
+        WHERE g.tournament_id = $1
+          AND g.result IS NOT NULL
+          AND opp_tp.starting_rating IS NOT NULL
+          AND opp_tp.starting_rating > 0
     `, tournamentID, participantID)
     if err != nil {
         return nil, fmt.Errorf("failed to query opponent ratings: %w", err)
     }
     defer rows.Close()

     var ratings []int
     for rows.Next() {
-        var rating int
+        var rating *int
         if err := rows.Scan(&rating); err != nil {
             return nil, fmt.Errorf("failed to scan rating: %w", err)
         }
-        ratings = append(ratings, rating)
+        if rating != nil && *rating > 0 {
+            ratings = append(ratings, *rating)
+        }
     }

     return ratings, rows.Err()
 }
```

---

## Bug #8: Missing Authorization Checks in All Handler Methods

**Severity:** Critical

**Description:**
None of the handler methods verify that the requesting user has permission to view tournament standings, player progress, or recalculate standings. This is a security vulnerability that allows:
- Any user to view standings of private tournaments
- Unauthorized users to trigger expensive recalculation operations
- Information leakage about tournament participants

While the pairing API likely has authentication middleware, authorization (checking if user can access THIS specific tournament) is missing entirely.

**Location:**
- File: `/home/user/openpairing/backend/internal/standings/handler.go`
- Functions: All handler methods
- Lines: 30-162

**Steps to Reproduce:**
1. Create a private tournament owned by User A
2. Authenticate as User B (different user)
3. Call `/tournaments/{tournamentId}/standings` with the private tournament ID
4. Expected: HTTP 403 Forbidden
5. Actual: HTTP 200 OK with full standings data

**Fix:**
Add authorization middleware or checks:

```diff
 func (h *Handler) GetStandings(c *gin.Context) {
     idStr := c.Param("id")
     id, err := uuid.Parse(idStr)
     if err != nil {
         c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
         return
     }

+    // Check authorization - ensure user can view this tournament
+    userID := c.GetString("userID") // from auth middleware
+    canView, err := h.service.CanUserViewTournament(c.Request.Context(), userID, id)
+    if err != nil {
+        c.JSON(http.StatusInternalServerError, gin.H{"error": "authorization check failed"})
+        return
+    }
+    if !canView {
+        c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
+        return
+    }

     standings, err := h.service.GetStandings(c.Request.Context(), id)
     if err != nil {
         if err.Error() == "tournament not found" {
             c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
             return
         }
         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
         return
     }

     c.JSON(http.StatusOK, standings)
 }

 func (h *Handler) RecalculateStandings(c *gin.Context) {
     idStr := c.Param("id")
     id, err := uuid.Parse(idStr)
     if err != nil {
         c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tournament ID"})
         return
     }

+    // Check authorization - only tournament directors can recalculate
+    userID := c.GetString("userID")
+    canModify, err := h.service.CanUserModifyTournament(c.Request.Context(), userID, id)
+    if err != nil {
+        c.JSON(http.StatusInternalServerError, gin.H{"error": "authorization check failed"})
+        return
+    }
+    if !canModify {
+        c.JSON(http.StatusForbidden, gin.H{"error": "only tournament directors can recalculate standings"})
+        return
+    }

     err = h.service.RecalculateStandings(c.Request.Context(), id)
```

This requires implementing authorization service methods and ensuring auth middleware populates user context.

---

## Statistics

- **Critical:** 1 bug (Missing authorization)
- **High:** 3 bugs (Ignored errors, missing round ranks, unrated opponents in performance rating)
- **Medium:** 3 bugs (String-based error handling, manual JSON parsing, hardcoded tiebreaks)
- **Low:** 1 bug (Missing BYE for black in crosstable)

---

## Recommendations

### Immediate Actions (Critical/High Priority)
1. **Add authorization checks** to all endpoints - this is a security vulnerability
2. **Fix performance rating calculation** to exclude unrated opponents
3. **Implement round ranks calculation** in player progress
4. **Handle errors from special rankings** properly (log or return warnings)

### Short-term Improvements (Medium Priority)
5. **Refactor error handling** to use typed errors with `errors.Is()`
6. **Replace manual JSON parser** with standard library `json.Unmarshal`
7. **Redesign tiebreak persistence** to support dynamic tiebreak systems

### Long-term Enhancements
8. **Add integration tests** for standings calculations with various tiebreak configurations
9. **Add performance tests** for tournaments with 100+ participants
10. **Consider caching** standings calculations for large tournaments
11. **Add audit logging** for recalculation operations (who triggered, when, what changed)

### Code Quality
- Consider adding pagination to standings endpoint for large tournaments
- Add request rate limiting for expensive operations like recalculation
- Document expected tournament states (must have games, participants, etc.)
- Add validation for scoring configuration (win > draw > loss, all >= 0)
