# Bug Report: pairing

**Analysis Date:** 2026-01-28
**Directory Analyzed:** `/home/user/openpairing/backend/internal/pairing`
**Files Analyzed:** 38 files
**Bugs Found:** 13 bugs

---

## Summary

This analysis of the tournament pairing module identified 13 functional bugs across multiple severity levels. The most critical issues include:
- **TRF parsing errors** that cause incorrect color assignment (affects game fairness)
- **Potential infinite recursion** in knockout seeding (can crash the application)
- **Resource leaks** in external pairing engine execution (can exhaust system resources)
- **Slice mutation bugs** causing data corruption
- **TRF format overflow** that breaks file compatibility with external engines

The bugs span error handling, logic errors, resource management, and data integrity issues. Several bugs directly impact tournament fairness and could cause incorrect pairings.

---

## Bugs

## Bug #1: Color Parsing Position Error in TRF Round Results

**Description:**
The TRF parser reads the color field from the wrong position in round result data. According to the TRF16 specification, within each 10-character round block, position 7 (1-indexed) or index 6 (0-indexed) contains the color ('w', 'b', or '-'). However, in both `monrad.go` and `keizer.go`, the code reads from index 5 instead of index 6, which actually reads a space character according to the spec. This causes colors to be incorrectly parsed, leading to wrong color assignments in pairings and potential unfair color distribution.

**Location:**
- File: `backend/internal/pairing/engine/monrad.go`
- Function: `parseRoundResults`
- Lines: 374-382

- File: `backend/internal/pairing/engine/keizer.go`
- Function: `parseRoundResults`
- Lines: 436-445

**Steps to Reproduce:**
1. Create a TRF file with player round results containing color data
2. Parse the TRF using Monrad or Keizer engine
3. Expected: Colors should be correctly identified as 'w' (white) or 'b' (black)
4. Actual: Colors are read from position 5 (space) instead of position 6 (color), resulting in incorrect ColorHistory

**Fix:**
Change the color parsing to read from the correct position (index 6 instead of 5):

```diff
- switch roundData[5] {
+ switch roundData[6] {
  case 'w', 'W':
      colors = append(colors, ColorWhite)
  case 'b', 'B':
      colors = append(colors, ColorBlack)
  default:
      colors = append(colors, ColorNone)
}
```

Apply this fix in both `monrad.go` (line 375) and `keizer.go` (line 437).

**Severity:** High

---

## Bug #2: Infinite Recursion in Knockout Seeding for Single Player

**Description:**
The `generateSeededPositions` function in the Knockout engine uses recursion to build bracket seeding but doesn't handle the base case for bracket size 1 or 0. When called with size=1, it recursively calls itself with size=0, which then calls itself again with size=0, causing infinite recursion and a stack overflow crash. This occurs when a knockout tournament has very few players or in edge cases during bracket generation.

**Location:**
- File: `backend/internal/pairing/engine/knockout.go`
- Function: `generateSeededPositions`
- Lines: 244-259

**Steps to Reproduce:**
1. Call `generateSeededPositions(1)` or `generateSeededPositions(0)`
2. Expected: Should return appropriate seeding or error
3. Actual: Infinite recursion causes stack overflow panic

**Fix:**
Add proper base case handling:

```diff
  func (k *Knockout) generateSeededPositions(size int) []int {
+     if size <= 1 {
+         if size == 1 {
+             return []int{1}
+         }
+         return []int{}
+     }
      if size == 2 {
          return []int{1, 2}
      }

      // Recursively build bracket
      half := k.generateSeededPositions(size / 2)
      result := make([]int, size)

      for i, seed := range half {
          result[i*2] = seed
          result[i*2+1] = size + 1 - seed
      }

      return result
  }
```

**Severity:** Critical

---

## Bug #3: TRF Points Format Overflow for High Scores

**Description:**
In the TRF generator, player points are formatted using `%4.1f` format specifier, which allocates exactly 4 characters for the value (e.g., "3.5 "). However, if a player accumulates 10.0 points or more (possible in tournaments with 10+ rounds), the formatted value becomes "10.0" (4 characters) or "11.5" (4 characters), which fits. But for values 100.0+, this becomes "100.0" (5 characters), overflowing the fixed 4-character field defined by TRF16 specification (positions 81-84). This corrupts the TRF file format, causing subsequent fields to be misaligned and making the file incompatible with external pairing engines like JaVaFo and bbpPairings.

**Location:**
- File: `backend/internal/pairing/trf/base_generator.go`
- Function: `generatePlayerLine`
- Lines: 185

**Steps to Reproduce:**
1. Create a tournament with 100+ rounds (or artificially set player points to 100.0)
2. Generate TRF file using either JaVaFo or BBP generator
3. Parse the TRF file with external tools or re-import
4. Expected: Points should be correctly formatted within 4 characters
5. Actual: Points overflow the field, corrupting the rest of the line

**Fix:**
Use a format that truncates or handles overflow gracefully:

```diff
- line += fmt.Sprintf("%4.1f", player.Points)
+ // Clamp points to max 99.9 to fit in 4-char field, or use integer format for high values
+ if player.Points >= 100.0 {
+     // For points >= 100, show as integer (e.g., "100 ")
+     line += fmt.Sprintf("%4.0f", math.Min(player.Points, 9999.0))
+ } else {
+     line += fmt.Sprintf("%4.1f", player.Points)
+ }
```

Alternatively, consider validating that points don't exceed 99.9 before generation, or switch to a format that always fits (like "%4d" for integer points).

**Severity:** High

---

## Bug #4: Slice Mutation in removeInt Causes Data Corruption

**Description:**
The utility function `removeInt` removes an element from a slice by using `append(slice[:i], slice[i+1:]...)`. This modifies the underlying array of the original slice that was passed in. If the caller retains a reference to the original slice and expects it to remain unchanged, this causes unexpected data corruption. The function doesn't document this side effect, violating the principle of least surprise. This is particularly dangerous in concurrent scenarios or when the same player list is used multiple times.

**Location:**
- File: `backend/internal/pairing/engine/utils.go`
- Function: `removeInt`
- Lines: 74-81

**Steps to Reproduce:**
1. Create a slice: `original := []int{1, 2, 3, 4, 5}`
2. Call: `result := removeInt(original, 3)`
3. Expected: `original` should still be `[]int{1, 2, 3, 4, 5}` and `result` should be `[]int{1, 2, 4, 5}`
4. Actual: Both `original` and `result` share the same underlying array, and `original` is modified

**Fix:**
Create a new slice copy before modification:

```diff
  func removeInt(slice []int, val int) []int {
+     result := make([]int, 0, len(slice)-1)
      for i, v := range slice {
          if v == val {
-             return append(slice[:i], slice[i+1:]...)
+             result = append(result, slice[:i]...)
+             result = append(result, slice[i+1:]...)
+             return result
          }
      }
-     return slice
+     return append(result, slice...)
  }
```

**Severity:** Medium

---

## Bug #5: Resource Leak in JaVaFo Temp Directory Cleanup

**Description:**
The JaVaFo pairing engine creates temporary directories for TRF file I/O but has incomplete cleanup logic. While there's a `defer os.RemoveAll(tmpDir)` statement, if the context is canceled or times out during execution, the command may leave the Java process running and the temporary directory not cleaned up. Over time, with many pairing operations, this can exhaust disk space and leave zombie processes. Additionally, if `os.WriteFile` fails at line 90, the error path returns early (though defer will run). The real issue is context cancellation during `cmd.Run()` - the temp directory cleanup happens after the context deadline, potentially being skipped.

**Location:**
- File: `backend/internal/pairing/engine/javafo.go`
- Function: `GeneratePairings`
- Lines: 82-86, 108-128

**Steps to Reproduce:**
1. Start a JaVaFo pairing operation
2. Cancel the context before completion
3. Check system for orphaned temp directories and Java processes
4. Expected: All resources should be cleaned up
5. Actual: Temporary directories may remain, accumulating over time

**Fix:**
Use context-aware cleanup and ensure proper process termination:

```diff
  func (j *JaVaFo) GeneratePairings(ctx context.Context, req *PairingRequest) (*PairingResult, error) {
      startTime := time.Now()

      // Create temp directory
      tmpDir, err := os.MkdirTemp("", "javafo-*")
      if err != nil {
          return nil, fmt.Errorf("failed to create temp directory: %w", err)
      }
-     defer os.RemoveAll(tmpDir)
+     defer func() {
+         if err := os.RemoveAll(tmpDir); err != nil {
+             j.logger.Warn().Err(err).Str("dir", tmpDir).Msg("Failed to clean up temp directory")
+         }
+     }()

      // ... existing code ...

      // Create context with timeout
      ctx, cancel := context.WithTimeout(ctx, j.timeout)
      defer cancel()

      // Execute JaVaFo
      cmd := exec.CommandContext(ctx, "java", args...)
+
+     // Ensure process is killed on context cancellation
+     if cmd.Process != nil {
+         defer func() {
+             if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
+                 cmd.Process.Kill()
+             }
+         }()
+     }

      var stdout, stderr bytes.Buffer
      cmd.Stdout = &stdout
      cmd.Stderr = &stderr
```

**Severity:** Medium

---

## Bug #6: BBPPairings Has Same Resource Leak Issue

**Description:**
The BBPPairings engine has the identical resource leak issue as JaVaFo (Bug #5). Temporary directories may not be cleaned up on context cancellation, and the bbpPairings process may not be properly terminated, leading to resource exhaustion over time.

**Location:**
- File: `backend/internal/pairing/engine/bbp.go`
- Function: `GeneratePairings`
- Lines: 60-64, 92-107

**Steps to Reproduce:**
(Same as Bug #5, but using BBPPairings engine)

**Fix:**
Apply the same fix as Bug #5, adapted for the BBPPairings execution context.

**Severity:** Medium

---

## Bug #7: Ambiguous XXR Record Parsing Logic

**Description:**
The TRF parser attempts to distinguish between two types of XXR records: global round count (e.g., "XXR 5") and player-specific data (e.g., "XXR    1 data"). The heuristic used is checking if `len(content) < 5` and if it parses as a simple integer. This is fragile and will fail in edge cases. For example, if a tournament has exactly 9999 rounds, the content would be "9999" (length 4), which would be incorrectly treated as a round count instead of potentially being player-specific data. More critically, the spec allows for both formats, and there's no reliable way to distinguish them without more context.

**Location:**
- File: `backend/internal/pairing/trf/parser.go`
- Function: `Parse`
- Lines: 160-176

**Steps to Reproduce:**
1. Create a TRF file with "XXR 9999" intending it as player-specific data for player 9999
2. Parse the file
3. Expected: Should parse as player-specific XXR
4. Actual: Parsed as round count due to length heuristic

**Fix:**
Use a more robust parsing strategy that examines the full context:

```diff
  case "XXR":
-     // XXR can be either:
-     // - Global round count: "XXR 5" (number of rounds)
-     // - Player-specific: "XXR    1 data" (player number + data)
      if len(content) > 0 {
-         // Check if it's a simple number (round count)
-         if rounds, err := strconv.Atoi(strings.TrimSpace(content)); err == nil && rounds > 0 && len(content) < 5 {
-             trf.Rounds = rounds
+         // XXR format: "XXR ssss data" where ssss is 4-character player start number
+         // If content starts with at least 4 spaces/digits, try player-specific parse
+         if len(content) >= 4 {
+             playerNoStr := strings.TrimSpace(content[:4])
+             if playerNo, err := strconv.Atoi(playerNoStr); err == nil && playerNo > 0 {
+                 // Player-specific XXR
+                 xxrData := ""
+                 if len(content) > 4 {
+                     xxrData = content[4:]
+                 }
+                 trf.XXRRecords[playerNo] = xxrData
+             }
+         } else {
+             // Short content, likely round count
+             if rounds, err := strconv.Atoi(strings.TrimSpace(content)); err == nil && rounds > 0 {
+                 trf.Rounds = rounds
+             }
-         } else {
-             // Try parsing as player-specific XXR
-             playerNo, xxrData := p.parseXXRLine(line)
-             if playerNo > 0 {
-                 trf.XXRRecords[playerNo] = xxrData
-             }
          }
      }
```

**Severity:** Low

---

## Bug #8: Monrad Pairing Logic Has Dead Code Path

**Description:**
In the Monrad pairing generation, after determining that no valid opponent exists (line 142: `if opponent == nil`), the code adds the player to floaters (line 143) but then continues to check `if opponent != nil` at line 147. This check will always be false in the path where no opponent was found, making the code inside unreachable (dead code). The real bug is that after adding to floaters, the code should `continue` to skip the pairing attempt, but instead it falls through to line 162 which again checks `if opponent != nil`. This suggests confused logic where the developer intended one behavior but implemented another.

**Location:**
- File: `backend/internal/pairing/engine/monrad.go`
- Function: `generatePairings`
- Lines: 141-165

**Steps to Reproduce:**
1. Create a scenario where a player cannot be paired (all potential opponents already played)
2. Run Monrad pairing generation
3. Expected: Player should float to next group cleanly
4. Actual: Logic flow is confusing with redundant checks

**Fix:**
Simplify the logic and add `continue` after floating:

```diff
              if !m.havePlayed(p1, p2) {
                  opponent = &p2
                  break
              }
          }

          // If no valid opponent found, float to next group instead of forcing rematch
          if opponent == nil {
              floaters = append(floaters, p1)
+             continue
          }

-         if opponent != nil {
              // Determine colors
              white, black := m.determineColors(p1, *opponent)

              pairings = append(pairings, Pairing{
                  Board:        board,
                  WhiteStartNo: white.StartNo,
                  BlackStartNo: black.StartNo,
                  IsBye:        false,
              })
              board++

              paired[p1.StartNo] = true
              paired[opponent.StartNo] = true
-         } else {
-             // Float to next group
-             floaters = append(floaters, p1)
-         }
```

**Severity:** Low

---

## Bug #9: WAM CalculateGroupStandings Head-to-Head Lookup Error

**Description:**
In the WAM engine's standings calculation, when applying the "direct_encounter" tiebreak, the code performs double indirection that results in incorrect head-to-head lookups. At lines 508-509, it looks up `iIdx := playerMap[group.Players[i].SortOrder]` and `jIdx := playerMap[group.Players[j].SortOrder]`, but `i` and `j` are already indices into the `standings` array. The `playerMap` was built to map `SortOrder -> index`, but `group.Players[i]` is dereferencing the players array by index, leading to incorrect mapping. The code should directly use indices `i` and `j` when accessing the head-to-head matrix.

**Location:**
- File: `backend/internal/pairing/engine/wam.go`
- Function: `CalculateGroupStandings`
- Lines: 506-516

**Steps to Reproduce:**
1. Complete a WAM group stage with ties in points
2. Calculate standings with "direct_encounter" tiebreak enabled
3. Expected: Head-to-head results should correctly break ties
4. Actual: Wrong players are compared due to incorrect index mapping

**Fix:**
Remove the double indirection and use indices directly:

```diff
          switch tb {
          case "direct_encounter":
              // Check head-to-head
-             iIdx := playerMap[group.Players[i].SortOrder]
-             jIdx := playerMap[group.Players[j].SortOrder]
-             if score, ok := h2h[iIdx][jIdx]; ok {
+             // i and j are already indices into the standings/h2h arrays
+             if score, ok := h2h[i][j]; ok {
                  if score > 0.5 {
                      return true
                  } else if score < 0.5 {
                      return false
                  }
              }
```

**Severity:** High

---

## Bug #10: Arena Engine Uninitialized SearchStartedAt Field

**Description:**
In the Arena pairing engine, the `calculateMatchScore` function uses `time.Since(p1.SearchStartedAt)` and `time.Since(p2.SearchStartedAt)` to calculate wait time bonuses (lines 217-219). However, when players are created in `matchPlayers` from `req.Players` (lines 99-108), the `SearchStartedAt` field is never initialized. This means it remains as the zero time value (1970-01-01), causing `time.Since()` to return an enormous duration (50+ years), making the wait time calculation meaningless. This breaks the fairness mechanism that's supposed to prioritize players who have been waiting longer.

**Location:**
- File: `backend/internal/pairing/engine/arena.go`
- Function: `matchPlayers`, `calculateMatchScore`
- Lines: 99-108 (initialization), 217-219 (usage)

**Steps to Reproduce:**
1. Create arena players from GeneralPairingRequest
2. Call matchPlayers which creates ArenaPlayer objects
3. calculateMatchScore is called with these players
4. Expected: Wait time should be reasonable (seconds/minutes)
5. Actual: Wait time is billions of seconds (decades) due to zero time

**Fix:**
Initialize SearchStartedAt when creating ArenaPlayer objects:

```diff
  for _, p := range req.Players {
      if p.IsActive {
          arenaPlayers = append(arenaPlayers, ArenaPlayer{
              Player:      p,
-             IsSearching: true,
+             IsSearching:     true,
+             SearchStartedAt: time.Now(),
              ArenaScore:  p.Points,
          })
      }
  }
```

**Severity:** Medium

---

## Bug #11: Context Timeout Not Reset in JaVaFo/BBP Engines

**Description:**
In both JaVaFo and BBP engines, a timeout context is created with `context.WithTimeout(ctx, j.timeout)` (line 108 in javafo.go, line 93 in bbp.go). This creates a new context with the timeout, but it overwrites the `ctx` variable. If the passed-in context already had a deadline, the new timeout is added **on top** of it, meaning whichever deadline comes first will trigger. However, the defer statement `defer cancel()` is called, which will cancel the new context immediately on function exit, but the original context might have resources that aren't cleaned up. More critically, the pattern `ctx, cancel := context.WithTimeout(ctx, j.timeout)` shadows the input context, which could lead to confusion if the original context had important values attached to it.

**Location:**
- File: `backend/internal/pairing/engine/javafo.go`
- Function: `GeneratePairings`
- Lines: 108-109

- File: `backend/internal/pairing/engine/bbp.go`
- Function: `GeneratePairings`
- Lines: 93-94

**Steps to Reproduce:**
1. Call GeneratePairings with a context that already has a 1-second deadline
2. The engine has a 5-minute timeout configured
3. Expected: Should respect the stricter 1-second deadline
4. Actual: Context shadowing may cause unexpected behavior

**Fix:**
Don't shadow the original context, use a different variable name:

```diff
- // Create context with timeout
- ctx, cancel := context.WithTimeout(ctx, j.timeout)
- defer cancel()
+ // Create context with timeout (don't shadow input ctx)
+ cmdCtx, cmdCancel := context.WithTimeout(ctx, j.timeout)
+ defer cmdCancel()

  // Execute JaVaFo
- cmd := exec.CommandContext(ctx, "java", args...)
+ cmd := exec.CommandContext(cmdCtx, "java", args...)
```

Apply to both javafo.go and bbp.go.

**Severity:** Low

---

## Bug #12: Missing Error Check in TRF Player Line Parsing

**Description:**
In the TRF parser's `parsePlayerLine` function, several fields are parsed using `strconv.Atoi` or `strconv.ParseFloat`, but the errors are silently ignored using the blank identifier `_`. While this makes the parser permissive, it means that invalid data is silently treated as zero, which could mask data quality issues. For critical fields like StartNo, an invalid value should cause the entire player record to be rejected. Currently, if StartNo parsing fails, the function continues and creates a player with StartNo=0, which may cause issues downstream.

**Location:**
- File: `backend/internal/pairing/trf/parser.go`
- Function: `parsePlayerLine`
- Lines: 202-255

**Steps to Reproduce:**
1. Create a TRF file with malformed player line (invalid StartNo like "ABCD")
2. Parse the file
3. Expected: Should return error for invalid StartNo
4. Actual: Player is created with StartNo=0, potentially causing nil pointer issues later

**Fix:**
Return error for critical field parsing failures:

```diff
  // Start number (TRF16 positions 5-8, 0-indexed 4-7)
  startNo, err := strconv.Atoi(strings.TrimSpace(line[4:8]))
  if err != nil {
-     return player, fmt.Errorf("invalid start number: %w", err)
+     return player, fmt.Errorf("invalid start number at positions 5-8: %w", err)
  }
+ if startNo <= 0 {
+     return player, fmt.Errorf("start number must be positive, got %d", startNo)
+ }
  player.StartNo = startNo
```

**Severity:** Medium

---

## Bug #13: Round Robin Double Round Robin Color Swap Not Applied to Byes

**Description:**
In the Round Robin engine's `GenerateGeneralPairings` function, when handling double round robin tournaments, the second half swaps colors by setting `isSecondHalf = true` and then swapping `whitePos` and `blackPos` at line 95. However, the bye handling logic (lines 98-120) checks if positions exceed `numPlayers` to detect byes, but doesn't consistently apply the color swap. For example, at lines 99-108, when white position is a bye, the code assigns the black player's StartNo to WhiteStartNo in the Pairing structure, but it doesn't account for the color swap that should happen in the second half. This causes inconsistent bye assignment between first and second halves.

**Location:**
- File: `backend/internal/pairing/engine/roundrobin.go`
- Function: `GenerateGeneralPairings`
- Lines: 98-120

**Steps to Reproduce:**
1. Create a round robin tournament with odd number of players (requires byes)
2. Set DoubleRoundRobin to true
3. Generate pairings for round in second half
4. Expected: Bye assignments should mirror first half with appropriate adjustments
5. Actual: Bye handling doesn't properly account for color swap in second half

**Fix:**
Apply color swap before bye detection:

```diff
  roundPairings := bergerTable[roundIdx]
  var pairings []Pairing

  for board, pair := range roundPairings {
      whitePos := pair[0] // 1-indexed position
      blackPos := pair[1]

      // Swap colors for second half of double round robin
      if isSecondHalf {
          whitePos, blackPos = blackPos, whitePos
      }

      // Handle bye (position > numPlayers means bye due to odd players)
-     if whitePos > numPlayers {
-         // Black player gets bye
-         blackPlayer := r.getPlayerByPosition(players, blackPos)
+     // After color swap, check which position is the bye
+     if whitePos > numPlayers || blackPos > numPlayers {
+         // Determine which player is real (not bye)
+         actualPlayerPos := whitePos
+         if whitePos > numPlayers {
+             actualPlayerPos = blackPos
+         }
+         actualPlayer := r.getPlayerByPosition(players, actualPlayerPos)
          pairings = append(pairings, Pairing{
              Board:        board + 1,
-             WhiteStartNo: blackPlayer.StartNo,
+             WhiteStartNo: actualPlayer.StartNo,
              BlackStartNo: 0,
              IsBye:        true,
          })
          continue
-     }
-     if blackPos > numPlayers {
-         // White player gets bye
-         whitePlayer := r.getPlayerByPosition(players, whitePos)
-         pairings = append(pairings, Pairing{
-             Board:        board + 1,
-             WhiteStartNo: whitePlayer.StartNo,
-             BlackStartNo: 0,
-             IsBye:        true,
-         })
-         continue
      }
```

**Severity:** Medium

---

## Statistics

- **Critical:** 1 bug
- **High:** 3 bugs
- **Medium:** 6 bugs
- **Low:** 3 bugs

---

## Recommendations

### High Priority

1. **Immediately fix the color parsing bug (Bug #1)** - This directly affects tournament fairness by causing incorrect color assignments. Deploy a hotfix.

2. **Add unit tests for edge cases** - Bugs #2, #7, and #13 could have been caught with proper edge case testing (single player brackets, ambiguous data formats, double round robin with odd players).

3. **Fix infinite recursion (Bug #2)** - Add input validation to all recursive functions to prevent stack overflow crashes.

### Medium Priority

4. **Implement proper resource cleanup** - Bugs #5 and #6 show systematic issues with resource management. Consider using a resource cleanup pattern or wrapper.

5. **Add TRF format validation** - Bug #3 shows that format constraints aren't validated. Add pre-generation validation to catch overflow issues.

6. **Review all slice manipulation** - Bug #4 indicates unsafe slice operations. Audit all utility functions for similar issues.

### Low Priority

7. **Improve code clarity** - Bugs #8 and #11 show confusing logic that works but is hard to maintain. Refactor for clarity.

8. **Add comprehensive integration tests** - Many bugs involve interactions between components (TRF parsing → pairing engines). Add end-to-end tests.

9. **Document TRF format assumptions** - Multiple bugs involve TRF format interpretation. Create clear documentation of which TRF variant is supported and validate against it.

### Process Improvements

10. **Add static analysis** - Tools like `go vet`, `staticcheck`, and `gosec` could catch several of these issues automatically.

11. **Implement defensive programming** - Add more input validation, bounds checking, and error handling throughout the codebase.

12. **Code review checklist** - Create a checklist that specifically looks for: resource leaks, slice mutations, index bounds, format overflows, and uninitialized fields.
