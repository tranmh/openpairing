# Bug Report: player

**Analysis Date:** 2026-01-28
**Directory Analyzed:** `/home/user/openpairing/backend/internal/player`
**Files Analyzed:** 6 files
**Bugs Found:** 11 bugs

---

## Summary

The player module contains several critical functional bugs including broken query parameter parsing in HTTP handlers, incomplete federation ID checks, data integrity issues, and validation gaps. The most severe issues are non-functional pagination and limit controls that completely ignore user input, missing validation for duplicate user-player links, and incomplete rating comparison logic.

**Critical Issues:**
- Broken query parameter parsing causing user input to be ignored
- Missing duplicate user link validation (data integrity risk)
- Incomplete federation ID support across multiple methods
- Missing transaction handling in bulk operations

---

## Bugs

## Bug #1: Broken Limit Parameter Parsing in GetRatingHistory Handler

**Severity:** HIGH

**Description:**
The `GetRatingHistory` handler contains incomplete and syntactically incorrect code for parsing the `limit` query parameter. Line 273 uses `c.GetQuery("limit")` which returns `(string, bool)`, not an error, making the condition always false. The parsing logic never executes, and the limit is always hardcoded to 50, completely ignoring user-specified values.

**Location:**
- File: `/home/user/openpairing/backend/internal/player/handler.go`
- Function: `GetRatingHistory`
- Lines: 272-276

**Steps to Reproduce:**
1. Call GET `/api/players/{id}/history?limit=10`
2. Observe that exactly 50 entries are returned (or fewer if less history exists)
3. Try with `/api/players/{id}/history?limit=5` - still returns 50
4. Expected: Should respect the limit parameter
5. Actual: Always uses limit=50, ignoring query parameter

**Current Code:**
```go
limit := 50 // default
if limitStr := c.Query("limit"); limitStr != "" {
    if _, err := c.GetQuery("limit"); err {  // BUG: err is bool, not error
        // Parse limit from query
    }
}
```

**Fix:**
Replace with proper integer parsing:

```diff
+ import "strconv"

  limit := 50 // default
  if limitStr := c.Query("limit"); limitStr != "" {
-     if _, err := c.GetQuery("limit"); err {
-         // Parse limit from query
-     }
+     if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
+         limit = parsedLimit
+         if limit > 1000 {
+             limit = 1000  // Cap at reasonable maximum
+         }
+     }
  }
```

---

## Bug #2: Non-Functional Pagination Parsing in ListClubs Handler

**Severity:** HIGH

**Description:**
The `ListClubs` handler contains placeholder code with TODO comments for parsing `page` and `limit` query parameters, but the implementation is completely missing. The code blocks are empty, causing pagination to always use hardcoded defaults (page=1, limit=20) regardless of client requests. This breaks pagination functionality entirely for the clubs listing endpoint.

**Location:**
- File: `/home/user/openpairing/backend/internal/player/handler.go`
- Function: `ListClubs`
- Lines: 299-308

**Steps to Reproduce:**
1. Call GET `/api/clubs?page=3&limit=50`
2. Observe response contains page=1, limit=20
3. Expected: Should return page 3 with 50 results per page
4. Actual: Always returns page 1 with 20 results

**Current Code:**
```go
page := 1
limit := 20
// Parse pagination params
if p := c.Query("page"); p != "" {
    // parse page
}
if l := c.Query("limit"); l != "" {
    // parse limit
}
```

**Fix:**
Implement the parsing logic:

```diff
+ import "strconv"

  page := 1
  limit := 20
- // Parse pagination params
  if p := c.Query("page"); p != "" {
-     // parse page
+     if parsedPage, err := strconv.Atoi(p); err == nil && parsedPage > 0 {
+         page = parsedPage
+     }
  }
  if l := c.Query("limit"); l != "" {
-     // parse limit
+     if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
+         limit = parsedLimit
+     }
  }
```

---

## Bug #3: GetHighestRating Ignores Rapid and Blitz Ratings

**Severity:** MEDIUM

**Description:**
The `GetHighestRating` method only compares three rating types (FideStandard, DWZ, OesbElo) but completely ignores FideRapid and FideBlitz. This produces incorrect results for players whose highest ratings are in rapid or blitz time controls. For example, Magnus Carlsen historically had blitz ratings higher than his standard rating, but this method would incorrectly report only his standard rating.

**Location:**
- File: `/home/user/openpairing/backend/internal/player/models.go`
- Function: `GetHighestRating`
- Lines: 149-162

**Steps to Reproduce:**
1. Create player: `{fideStandard: 2750, fideRapid: 2850, fideBlitz: 2900}`
2. Call `player.GetHighestRating()`
3. Expected: Returns 2900 (blitz is highest)
4. Actual: Returns 2750 (only checks standard, ignores rapid/blitz)

**Current Code:**
```go
func (p *Player) GetHighestRating() int {
    var highest int
    if p.FideStandard != nil && *p.FideStandard > highest {
        highest = *p.FideStandard
    }
    if p.DWZ != nil && *p.DWZ > highest {
        highest = *p.DWZ
    }
    if p.OesbElo != nil && *p.OesbElo > highest {
        highest = *p.OesbElo
    }
    return highest
}
```

**Fix:**
Include all FIDE rating types:

```diff
func (p *Player) GetHighestRating() int {
    var highest int
    if p.FideStandard != nil && *p.FideStandard > highest {
        highest = *p.FideStandard
    }
+   if p.FideRapid != nil && *p.FideRapid > highest {
+       highest = *p.FideRapid
+   }
+   if p.FideBlitz != nil && *p.FideBlitz > highest {
+       highest = *p.FideBlitz
+   }
    if p.DWZ != nil && *p.DWZ > highest {
        highest = *p.DWZ
    }
    if p.OesbElo != nil && *p.OesbElo > highest {
        highest = *p.OesbElo
    }
    return highest
}
```

---

## Bug #4: HasFederationID Ignores SSB Federation

**Severity:** MEDIUM

**Description:**
The `HasFederationID` method checks FideID, DsbID, and OesbID but ignores SsbID (Swiss Chess Federation). Since the Player model includes SsbID as a field and the repository has UpsertBySsbID methods, this is clearly supported data. This bug causes Swiss federation players to incorrectly report as having no federation ID, which could affect tournament eligibility checks or federation sync logic.

**Location:**
- File: `/home/user/openpairing/backend/internal/player/models.go`
- Function: `HasFederationID`
- Lines: 164-169

**Steps to Reproduce:**
1. Create player with only `{ssbId: "CH-123456"}` (Swiss federation)
2. Call `player.HasFederationID()`
3. Expected: Returns true (player has SSB federation ID)
4. Actual: Returns false (SsbID not checked)

**Current Code:**
```go
func (p *Player) HasFederationID() bool {
    return (p.FideID != nil && *p.FideID != "") ||
        (p.DsbID != nil && *p.DsbID != "") ||
        (p.OesbID != nil && *p.OesbID != "")
}
```

**Fix:**
Add SsbID to the check:

```diff
func (p *Player) HasFederationID() bool {
    return (p.FideID != nil && *p.FideID != "") ||
        (p.DsbID != nil && *p.DsbID != "") ||
-       (p.OesbID != nil && *p.OesbID != "")
+       (p.OesbID != nil && *p.OesbID != "") ||
+       (p.SsbID != nil && *p.SsbID != "")
}
```

---

## Bug #5: Inconsistent ClubName Population Across Lookup Methods

**Severity:** MEDIUM

**Description:**
The methods `GetByFideID`, `GetByDsbID`, `GetByOesbID`, and `GetByUserID` do not join with the clubs table, leaving `ClubName` unpopulated (NULL). However, `GetByID` and `GetByIDs` do perform the join and populate this field. This inconsistency means the API returns different data structures depending on the lookup method used, breaking client expectations and potentially causing null reference errors.

**Location:**
- File: `/home/user/openpairing/backend/internal/player/repository.go`
- Functions: `GetByFideID`, `GetByDsbID`, `GetByOesbID`, `GetByUserID`
- Lines: 187-305

**Steps to Reproduce:**
1. Create player with club_id = "abc-123" referencing existing club "Chess Club Berlin"
2. Fetch via GET `/api/players/{id}` - response includes `"clubName": "Chess Club Berlin"`
3. Fetch same player via FIDE ID - response has `"clubName": null` despite having club_id
4. Expected: ClubName should always be populated when club_id exists
5. Actual: Only populated for GetByID and GetByIDs queries

**Fix:**
Add clubs table join to all lookup methods. Example for `GetByFideID`:

```diff
func (r *PostgresRepository) GetByFideID(ctx context.Context, fideID string) (*Player, error) {
    query := `
        SELECT
-           id, user_id, first_name, last_name, birth_date, gender,
-           fide_id, dsb_id, oesb_id, ssb_id,
-           fide_standard, fide_rapid, fide_blitz, dwz, dwz_index, oesb_elo,
-           fide_title, national_title, club_id, federation,
-           ratings_updated_at, created_at, updated_at
-       FROM players
-       WHERE fide_id = $1`
+           p.id, p.user_id, p.first_name, p.last_name, p.birth_date, p.gender,
+           p.fide_id, p.dsb_id, p.oesb_id, p.ssb_id,
+           p.fide_standard, p.fide_rapid, p.fide_blitz, p.dwz, p.dwz_index, p.oesb_elo,
+           p.fide_title, p.national_title, p.club_id, p.federation,
+           p.ratings_updated_at, p.created_at, p.updated_at,
+           c.name as club_name
+       FROM players p
+       LEFT JOIN clubs c ON p.club_id = c.id
+       WHERE p.fide_id = $1`

    var player Player
    err := r.db.QueryRow(ctx, query, fideID).Scan(
        &player.ID, &player.UserID, &player.FirstName, &player.LastName, &player.BirthDate, &player.Gender,
        &player.FideID, &player.DsbID, &player.OesbID, &player.SsbID,
        &player.FideStandard, &player.FideRapid, &player.FideBlitz, &player.DWZ, &player.DWZIndex, &player.OesbElo,
        &player.FideTitle, &player.NationalTitle, &player.ClubID, &player.Federation,
        &player.RatingsUpdatedAt, &player.CreatedAt, &player.UpdatedAt,
+       &player.ClubName,
    )
```

Apply the same pattern to `GetByDsbID` (lines 217-245), `GetByOesbID` (lines 247-275), and `GetByUserID` (lines 277-305).

---

## Bug #6: Missing Duplicate User Link Validation

**Severity:** CRITICAL

**Description:**
The `LinkPlayerToUser` method allows linking a player profile to a user without checking if (1) the player is already linked to a different user, or (2) the target user already has a different player linked. This creates data integrity issues: a player can be silently reassigned between users, or a user can have their existing player profile replaced, all without validation or error messages. This could cause data loss and confusion in production.

**Location:**
- File: `/home/user/openpairing/backend/internal/player/service.go`
- Function: `LinkPlayerToUser`
- Lines: 294-316

**Steps to Reproduce:**
1. Link player P1 to user U1: `LinkPlayerToUser(P1, U1)` ✓
2. User U1 now has player profile P1
3. Link same player to different user: `LinkPlayerToUser(P1, U2)` ✓ (should fail!)
4. Expected: Error "player already linked to another user"
5. Actual: P1 silently disconnected from U1 and reassigned to U2

**Current Code:**
```go
func (s *Service) LinkPlayerToUser(ctx context.Context, playerID, userID uuid.UUID) error {
    player, err := s.repo.GetByID(ctx, playerID)
    if err != nil {
        return err
    }

    player.UserID = &userID  // No validation!
    if err := s.repo.Update(ctx, player); err != nil {
        // ...
    }
    return nil
}
```

**Fix:**
Add validation checks before linking:

```diff
func (s *Service) LinkPlayerToUser(ctx context.Context, playerID, userID uuid.UUID) error {
    player, err := s.repo.GetByID(ctx, playerID)
    if err != nil {
        return err
    }

+   // Check if player is already linked to a different user
+   if player.UserID != nil && *player.UserID != userID {
+       return fmt.Errorf("player %s is already linked to user %s",
+           playerID, *player.UserID)
+   }
+
+   // Check if target user already has a different player linked
+   existingPlayer, err := s.repo.GetByUserID(ctx, userID)
+   if err == nil && existingPlayer.ID != playerID {
+       return fmt.Errorf("user %s is already linked to player %s",
+           userID, existingPlayer.ID)
+   }
+   if err != nil && !errors.Is(err, ErrPlayerNotFound) {
+       s.logger.Error().Err(err).Msg("Failed to check existing player link")
+       return err
+   }

    player.UserID = &userID
    if err := s.repo.Update(ctx, player); err != nil {
        s.logger.Error().Err(err).
            Str("playerId", playerID.String()).
            Str("userId", userID.String()).
            Msg("Failed to link player to user")
        return err
    }

    s.logger.Info().
        Str("playerId", playerID.String()).
        Str("userId", userID.String()).
        Msg("Player linked to user")

    return nil
}
```

---

## Bug #7: Federation Filter Missing SSB Support

**Severity:** MEDIUM

**Description:**
The `SearchPlayersRequest` validation only allows federation values of "fide", "dsb", and "oesb", but the system fully supports Swiss federation (SSB) as evidenced by the SsbID field and UpsertBySsbID methods. This prevents API clients from filtering players by Swiss federation membership, making Swiss players unsearchable by their federation.

**Location:**
- File: `/home/user/openpairing/backend/internal/player/models.go`
- Struct: `SearchPlayersRequest`
- Line: 116

**Steps to Reproduce:**
1. Send GET `/api/players?federation=ssb`
2. Receive 400 Bad Request: "Key: 'SearchPlayersRequest.Federation' Error:Field validation for 'Federation' failed on the 'oneof' tag"
3. Expected: Should filter for players with SSB IDs
4. Actual: Request rejected by validation

**Current Code:**
```go
type SearchPlayersRequest struct {
    Query      string  `form:"q"`
    Federation *string `form:"federation" binding:"omitempty,oneof=fide dsb oesb"`
    // ...
}
```

**Fix:**
Add "ssb" to allowed values and implement filtering:

```diff
type SearchPlayersRequest struct {
    Query      string  `form:"q"`
-   Federation *string `form:"federation" binding:"omitempty,oneof=fide dsb oesb"`
+   Federation *string `form:"federation" binding:"omitempty,oneof=fide dsb oesb ssb"`
    MinRating  *int    `form:"minRating" binding:"omitempty,min=0,max=3500"`
    MaxRating  *int    `form:"maxRating" binding:"omitempty,min=0,max=3500"`
    ClubID     *string `form:"clubId"`
    Page       int     `form:"page,default=1" binding:"min=1"`
    Limit      int     `form:"limit,default=20" binding:"min=1,max=100"`
}
```

Also update the repository Search method (around line 407-417):

```diff
// Add federation filter
if req.Federation != nil {
    switch *req.Federation {
    case "fide":
        conditions = append(conditions, fmt.Sprintf("p.fide_id IS NOT NULL"))
    case "dsb":
        conditions = append(conditions, fmt.Sprintf("p.dsb_id IS NOT NULL"))
    case "oesb":
        conditions = append(conditions, fmt.Sprintf("p.oesb_id IS NOT NULL"))
+   case "ssb":
+       conditions = append(conditions, fmt.Sprintf("p.ssb_id IS NOT NULL"))
    }
}
```

---

## Bug #8: Missing MinRating vs MaxRating Validation

**Severity:** MEDIUM

**Description:**
The search API accepts both `minRating` and `maxRating` parameters but never validates that min ≤ max. This allows nonsensical queries like `?minRating=2500&maxRating=1500` which will execute expensive database queries that can never match any records. The test file (repository_test.go:157) explicitly tests this validation, indicating it was intended but never implemented.

**Location:**
- File: `/home/user/openpairing/backend/internal/player/service.go`
- Function: `SearchPlayers`
- Lines: 220-251

**Steps to Reproduce:**
1. Call GET `/api/players?minRating=3000&maxRating=2000`
2. Query executes successfully, returns empty results after database scan
3. Expected: 400 Bad Request "minRating cannot exceed maxRating"
4. Actual: Query executes and returns empty result set

**Fix:**
Add validation after setting defaults:

```diff
func (s *Service) SearchPlayers(ctx context.Context, req *SearchPlayersRequest) (*PlayerListResponse, error) {
    // Set defaults
    if req.Page <= 0 {
        req.Page = 1
    }
    if req.Limit <= 0 {
        req.Limit = 20
    }
    if req.Limit > 100 {
        req.Limit = 100
    }

+   // Validate rating range
+   if req.MinRating != nil && req.MaxRating != nil {
+       if *req.MinRating > *req.MaxRating {
+           return nil, errors.New("minRating cannot be greater than maxRating")
+       }
+   }

    results, total, err := s.repo.Search(ctx, req)
    if err != nil {
        s.logger.Error().Err(err).Str("query", req.Query).Msg("Failed to search players")
        return nil, err
    }
```

---

## Bug #9: Bulk Upsert Operations Lack Transaction Boundaries

**Severity:** HIGH

**Description:**
The bulk upsert methods (`BulkUpsertByFideID`, `BulkUpsertByOesbID`, `BulkUpsertBySsbID`) use pgx batches but don't wrap operations in database transactions. If an error occurs midway through a 1000-player import (e.g., constraint violation on player 500), the first 499 players are committed while 501-1000 are not processed, leaving the database in an inconsistent state with no rollback capability. This violates atomicity requirements for bulk operations.

**Location:**
- File: `/home/user/openpairing/backend/internal/player/repository.go`
- Functions: `BulkUpsertByFideID`, `BulkUpsertByOesbID`, `BulkUpsertBySsbID`
- Lines: 746-955

**Steps to Reproduce:**
1. Prepare 100 players for bulk upsert
2. Make player 50 have invalid data (e.g., NULL FIDE ID)
3. Call `BulkUpsertByFideID(players)`
4. Error occurs at player 50
5. Expected: All 100 operations rolled back (atomic)
6. Actual: Players 1-49 are committed, 50-100 are lost

**Fix:**
Wrap batch operations in a transaction (example for BulkUpsertByFideID):

```diff
func (r *PostgresRepository) BulkUpsertByFideID(ctx context.Context, players []*Player) error {
    if len(players) == 0 {
        return nil
    }

+   // Begin transaction for atomicity
+   tx, err := r.db.Begin(ctx)
+   if err != nil {
+       return fmt.Errorf("failed to begin transaction: %w", err)
+   }
+   defer tx.Rollback(ctx)  // Safe to call even after commit
+
    batch := &pgx.Batch{}
    now := time.Now()

    for _, player := range players {
        player.UpdatedAt = now
        if player.ID == uuid.Nil {
            player.ID = uuid.New()
            player.CreatedAt = now
        }

        query := `
            INSERT INTO players (
                id, first_name, last_name, fide_id, fide_standard, fide_rapid, fide_blitz,
                fide_title, federation, ratings_updated_at, created_at, updated_at
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
            ON CONFLICT (fide_id) DO UPDATE SET
                first_name = EXCLUDED.first_name,
                last_name = EXCLUDED.last_name,
                fide_standard = EXCLUDED.fide_standard,
                fide_rapid = EXCLUDED.fide_rapid,
                fide_blitz = EXCLUDED.fide_blitz,
                fide_title = EXCLUDED.fide_title,
                federation = EXCLUDED.federation,
                ratings_updated_at = EXCLUDED.ratings_updated_at,
                updated_at = EXCLUDED.updated_at`

        batch.Queue(query,
            player.ID, player.FirstName, player.LastName, player.FideID,
            player.FideStandard, player.FideRapid, player.FideBlitz,
            player.FideTitle, player.Federation, player.RatingsUpdatedAt,
            player.CreatedAt, player.UpdatedAt,
        )
    }

-   results := r.db.SendBatch(ctx, batch)
+   results := tx.SendBatch(ctx, batch)
    defer results.Close()

    for range players {
        _, err := results.Exec()
        if err != nil {
            return fmt.Errorf("failed to execute batch upsert: %w", err)
        }
    }

+   // Commit transaction
+   if err := tx.Commit(ctx); err != nil {
+       return fmt.Errorf("failed to commit batch upsert: %w", err)
+   }

    return nil
}
```

Apply the same fix to `BulkUpsertByOesbID` (lines 862-907) and `BulkUpsertBySsbID` (lines 910-955).

---

## Bug #10: Foreign Key Violation Not Handled in CreatePlayer

**Severity:** MEDIUM

**Description:**
When creating a player with a `clubId`, the service validates the UUID format but doesn't verify the club exists in the database. If a valid UUID is provided for a non-existent club, the database insert fails with a foreign key constraint violation. This error is caught generically as "failed to create player" instead of the more specific and actionable "club not found" error, providing poor user experience.

**Location:**
- File: `/home/user/openpairing/backend/internal/player/service.go`
- Function: `CreatePlayer`
- Lines: 43-55

**Steps to Reproduce:**
1. Call POST `/api/players` with body:
   ```json
   {
     "firstName": "Test",
     "lastName": "Player",
     "clubId": "12345678-1234-1234-1234-123456789012"
   }
   ```
   where the UUID doesn't exist in clubs table
2. Receive 500 Internal Server Error: "Failed to create player"
3. Expected: 400 Bad Request: "Club not found"
4. Actual: Generic 500 error requiring log analysis to debug

**Fix:**
Validate club existence before creating player:

```diff
// Parse club ID if provided
if req.ClubID != nil && *req.ClubID != "" {
    clubUUID, err := uuid.Parse(*req.ClubID)
    if err != nil {
        return nil, errors.New("invalid club ID format")
    }
+
+   // Verify club exists
+   _, err = s.repo.GetClubByID(ctx, clubUUID)
+   if err != nil {
+       if errors.Is(err, ErrClubNotFound) {
+           return nil, ErrClubNotFound
+       }
+       s.logger.Error().Err(err).Str("clubId", clubUUID.String()).
+           Msg("Failed to verify club existence")
+       return nil, fmt.Errorf("failed to verify club: %w", err)
+   }
+
    player.ClubID = &clubUUID
}
```

Also update handler.go CreatePlayer (around line 150):

```diff
player, err := h.service.CreatePlayer(c.Request.Context(), &req)
if err != nil {
    switch {
+   case errors.Is(err, ErrClubNotFound):
+       response.BadRequest(c, "Club not found")
    case errors.Is(err, ErrDuplicateFideID):
        response.Conflict(c, "Player with this FIDE ID already exists")
    case errors.Is(err, ErrDuplicateDsbID):
        response.Conflict(c, "Player with this DSB ID already exists")
    case errors.Is(err, ErrDuplicateOesbID):
        response.Conflict(c, "Player with this ÖSB ID already exists")
    default:
        response.InternalServerError(c, "Failed to create player")
    }
    return
}
```

---

## Bug #11: Empty String Federation IDs Cause Unique Constraint Violations

**Severity:** MEDIUM

**Description:**
The `UpdatePlayer` method allows updating federation IDs (FideID, DsbID, OesbID) to empty strings without converting them to NULL. In PostgreSQL, empty strings are treated as distinct values for unique constraints, not as NULL. If two players are both updated with `"fideId": ""`, the second update will fail with a unique constraint violation because both would have the same empty string value, which violates the UNIQUE constraint on fide_id.

**Location:**
- File: `/home/user/openpairing/backend/internal/player/service.go`
- Function: `UpdatePlayer`
- Lines: 150-157

**Steps to Reproduce:**
1. Create player P1 with no FIDE ID (NULL)
2. Update P1: PATCH `/api/players/{id1}` with `{"fideId": ""}`
3. Create player P2 with no FIDE ID (NULL)
4. Update P2: PATCH `/api/players/{id2}` with `{"fideId": ""}`
5. Expected: Both succeed (empty string means no ID)
6. Actual: Second update fails with "duplicate key value violates unique constraint players_fide_id_key"

**Current Code:**
```go
if req.FideID != nil {
    player.FideID = req.FideID  // Could be pointer to ""
}
if req.DsbID != nil {
    player.DsbID = req.DsbID  // Could be pointer to ""
}
if req.OesbID != nil {
    player.OesbID = req.OesbID  // Could be pointer to ""
}
```

**Fix:**
Normalize empty strings to nil for federation IDs:

```diff
if req.FideID != nil {
+   if *req.FideID == "" {
+       player.FideID = nil
+   } else {
        player.FideID = req.FideID
+   }
}
if req.DsbID != nil {
+   if *req.DsbID == "" {
+       player.DsbID = nil
+   } else {
        player.DsbID = req.DsbID
+   }
}
if req.OesbID != nil {
+   if *req.OesbID == "" {
+       player.OesbID = nil
+   } else {
        player.OesbID = req.OesbID
+   }
}
```

Alternatively, handle this at the database schema level by adding CHECK constraints:
```sql
ALTER TABLE players ADD CONSTRAINT fide_id_not_empty
    CHECK (fide_id IS NULL OR fide_id != '');
ALTER TABLE players ADD CONSTRAINT dsb_id_not_empty
    CHECK (dsb_id IS NULL OR dsb_id != '');
ALTER TABLE players ADD CONSTRAINT oesb_id_not_empty
    CHECK (oesb_id IS NULL OR oesb_id != '');
```

---

## Statistics

- **Critical:** 1 bug (Bug #6: Missing duplicate user link validation)
- **High:** 3 bugs (Bugs #1, #2, #9: Broken parsing, missing transactions)
- **Medium:** 7 bugs (Bugs #3, #4, #5, #7, #8, #10, #11: Logic errors, validation gaps)
- **Low:** 0 bugs

---

## Recommendations

### Immediate Actions (Critical/High Priority)

1. **Fix handler parsing issues** (Bugs #1, #2): Add proper strconv.Atoi() parsing for all query parameters. Consider creating a helper function for consistent parameter parsing across all handlers.

2. **Implement user-player link validation** (Bug #6): Add bidirectional checks before linking to prevent data corruption. Consider adding database-level constraints (unique index on user_id where not null).

3. **Add transaction support to bulk operations** (Bug #9): Wrap all batch operations in transactions. Consider adding configurable batch sizes to prevent long-running transactions.

### Medium Priority Improvements

4. **Complete SSB federation support** (Bugs #4, #7): Audit all methods for complete federation support across all four federations (FIDE, DSB, ÖSB, SSB).

5. **Standardize data population** (Bug #5): Ensure all lookup methods return consistent data structures. Consider using a single shared query builder or view.

6. **Add comprehensive validation** (Bugs #8, #11): Implement range validation, empty string normalization, and consider adding validation middleware.

7. **Fix rating calculations** (Bug #3): Update GetHighestRating to include all rating types. Consider adding GetHighestRatingByType(type string) for more flexible queries.

### Code Quality Improvements

8. **Add integration tests**: Current tests are mostly unit tests with mocks. Add database integration tests for repository methods.

9. **Standardize error handling**: Create specific error types for different failure scenarios rather than generic errors.

10. **Add query parameter validation middleware**: Create reusable Gin middleware for parsing and validating common query parameters (pagination, limits, ranges).

11. **Consider adding database migrations**: Add CHECK constraints to prevent empty strings in federation ID fields at the database level.

12. **Add API request/response logging**: Log all player modifications for audit trail and debugging.

---

**End of Bug Report**
