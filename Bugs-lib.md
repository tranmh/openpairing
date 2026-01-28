# Bug Report: lib

**Analysis Date:** 2026-01-28
**Directory Analyzed:** `/home/user/openpairing/frontend/src/lib`
**Files Analyzed:** 4 files
**Bugs Found:** 11 bugs

---

## Summary

The lib directory contains the core API client and utility functions for the frontend. Analysis revealed critical bugs in date validation, type safety issues in the API client, error handling gaps, and edge case failures in utility functions. The most critical issues involve unsafe type assertions, missing validation for invalid dates, and potential data loss in error scenarios. Security concerns include silent failure modes that could mask authentication issues.

---

## Bugs

## Bug #1: Unused Tournament ID Parameter in enterResult

**Description:**
The `enterResult` function in the tournaments API accepts a `tournamentId` parameter that is explicitly marked as unused (prefixed with underscore). However, the API endpoint `/results` is called without any tournament context. This could cause issues if multiple tournaments need to track results independently, or if the backend expects tournament context for authorization or validation.

**Location:**
- File: `/home/user/openpairing/frontend/src/lib/api.ts`
- Function: `tournamentsApi.enterResult`
- Lines: 440-441

**Steps to Reproduce:**
1. Call `tournamentsApi.enterResult('tournament-123', 'game-456', '1-0')`
2. Observe that the request goes to `/results` without tournament ID
3. Expected: Tournament ID should be part of the URL or request body
4. Actual: Tournament ID is ignored completely

**Fix:**
Either remove the unused parameter if it's truly not needed, or include it in the API request if it should be validated:

```diff
- enterResult: (_tournamentId: string, gameId: string, result: string) =>
-   api.post<{ game: Game }>(`/results`, { gameId, result }),
+ enterResult: (tournamentId: string, gameId: string, result: string) =>
+   api.post<{ game: Game }>(`/tournaments/${tournamentId}/results`, { gameId, result }),
```

Alternatively, if the backend truly doesn't need tournament ID:

```diff
- enterResult: (_tournamentId: string, gameId: string, result: string) =>
+ enterResult: (gameId: string, result: string) =>
    api.post<{ game: Game }>(`/results`, { gameId, result }),
```

---

## Bug #2: Unsafe Type Assertion for Non-JSON Responses

**Description:**
When the API receives a successful non-JSON response, the code performs an unsafe type assertion `text as T` (line 127). This means a plain text string response is blindly cast to whatever type `T` the caller expects (e.g., `Tournament`, `Player`, etc.). This will cause runtime errors when the calling code tries to access properties on what it thinks is an object but is actually a string.

**Location:**
- File: `/home/user/openpairing/frontend/src/lib/api.ts`
- Function: `request`
- Lines: 115-128

**Steps to Reproduce:**
1. Backend returns a successful response with `Content-Type: text/plain` and body "OK"
2. Frontend calls `api.get<Tournament>('/some-endpoint')`
3. The function returns `{ data: "OK" as Tournament, status: 200 }`
4. Expected: Type error or proper handling
5. Actual: Caller receives a string typed as Tournament, causing runtime errors when accessing `tournament.name`, etc.

**Fix:**
Either explicitly handle text responses or throw an error for unexpected content types:

```diff
  } else {
    // Non-JSON response
    const text = await response.text();
    if (!response.ok) {
      return {
        status,
        error: {
          message: text || `HTTP ${status} error`,
          code: 'NON_JSON_ERROR',
        },
      };
    }
-   return { data: text as T, status };
+   // For successful non-JSON responses, only allow if T is string
+   // Otherwise this is likely a misconfiguration
+   if (typeof text === 'string') {
+     return { data: text as T, status };
+   }
+   return {
+     status,
+     error: {
+       message: 'Expected JSON response but received text',
+       code: 'UNEXPECTED_CONTENT_TYPE',
+     },
+   };
  }
```

Better approach - be explicit about text responses:

```diff
+ getText(path: string, params?: Record<string, string>) {
+   return this.request<string>('GET', path, { params });
+ }
```

---

## Bug #3: No Validation for Invalid Dates in Format Functions

**Description:**
The date formatting functions (`formatDate`, `formatShortDate`, `formatTime`) accept string inputs and convert them to Date objects without validating if the result is a valid date. When passed an invalid date string like "invalid" or "2024-13-99", `new Date()` creates an Invalid Date object, and calling `toLocaleDateString()` on it returns the string "Invalid Date" which is displayed to users.

**Location:**
- File: `/home/user/openpairing/frontend/src/lib/utils.ts`
- Functions: `formatDate`, `formatShortDate`, `formatTime`
- Lines: 14-45

**Steps to Reproduce:**
1. Call `formatDate("invalid-date-string")`
2. Expected: Throw error or return fallback like "-" or "N/A"
3. Actual: Returns "Invalid Date" string which is shown to users

**Fix:**
Add validation to check if the date is valid before formatting:

```diff
export function formatDate(date: Date | string, options?: Intl.DateTimeFormatOptions): string {
  const d = typeof date === 'string' ? new Date(date) : date;
+ if (isNaN(d.getTime())) {
+   return '-';
+ }
  return d.toLocaleDateString('de-DE', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    ...options,
  });
}

export function formatShortDate(date: Date | string): string {
  const d = typeof date === 'string' ? new Date(date) : date;
+ if (isNaN(d.getTime())) {
+   return '-';
+ }
  return d.toLocaleDateString('de-DE', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  });
}

export function formatTime(date: Date | string): string {
  const d = typeof date === 'string' ? new Date(date) : date;
+ if (isNaN(d.getTime())) {
+   return '-';
+ }
  return d.toLocaleTimeString('de-DE', {
    hour: '2-digit',
    minute: '2-digit',
  });
}
```

---

## Bug #4: Negative Time Values Not Handled in formatClockTime

**Description:**
The `formatClockTime` function doesn't handle negative millisecond values, which can occur when a chess clock expires or when calculating time differences. Negative values will produce incorrect formatted strings like "-1:30" or weird values due to `Math.floor` on negative numbers.

**Location:**
- File: `/home/user/openpairing/frontend/src/lib/utils.ts`
- Function: `formatClockTime`
- Lines: 50-60

**Steps to Reproduce:**
1. Call `formatClockTime(-5000)` (expired by 5 seconds)
2. Expected: Return "0:00" or handle gracefully
3. Actual: Returns "-1:-5" or similar malformed time

**Fix:**
Handle negative values explicitly:

```diff
export function formatClockTime(ms: number): string {
+ // Handle negative values (expired time)
+ if (ms < 0) {
+   return '0:00';
+ }
+
  const totalSeconds = Math.floor(ms / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
  }
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
}
```

---

## Bug #5: storage.get Returns Null Instead of Default Value

**Description:**
The `storage.get` function has a logic error where if localStorage contains a JSON-serialized null value (the string `"null"`), it will parse it to JavaScript `null` and return that instead of the provided default value. This breaks the contract of the function which promises to return the default value when the stored value is null or missing.

**Location:**
- File: `/home/user/openpairing/frontend/src/lib/utils.ts`
- Function: `storage.get`
- Lines: 212-220

**Steps to Reproduce:**
1. Store a null value: `localStorage.setItem('test', JSON.stringify(null))`
2. Call `storage.get('test', 'default-value')`
3. Expected: Returns `'default-value'` because stored value is null
4. Actual: Returns `null` because `JSON.parse("null")` returns null which is truthy after parsing

**Fix:**
Check if the parsed value is null and return default in that case:

```diff
get<T>(key: string, defaultValue: T): T {
  if (typeof window === 'undefined') return defaultValue;
  try {
    const item = localStorage.getItem(key);
-   return item ? JSON.parse(item) : defaultValue;
+   if (!item) return defaultValue;
+   const parsed = JSON.parse(item);
+   return parsed !== null ? parsed : defaultValue;
  } catch {
    return defaultValue;
  }
},
```

---

## Bug #6: Rating Parameter Ambiguity in playersApi.search

**Description:**
The `playersApi.search` function converts `minRating` and `maxRating` to strings using `String()` without distinguishing between `undefined` (don't include parameter) and `0` (include as "0"). This means a search with `minRating: 0` will send "0" to the backend, but it's unclear if 0 is a valid minimum rating or if it should be treated as "no minimum". This could cause unexpected filtering behavior.

**Location:**
- File: `/home/user/openpairing/frontend/src/lib/api.ts`
- Function: `playersApi.search`
- Lines: 394-399

**Steps to Reproduce:**
1. Call `playersApi.search('Smith', { minRating: 0 })`
2. The API request includes `minRating=0` in query params
3. Expected: Either exclude the parameter or document that 0 is a valid minimum
4. Actual: Sends 0 which may or may not be handled correctly by backend

**Fix:**
Only include rating parameters if they are defined and non-zero:

```diff
search: (query: string, params?: { federation?: string; minRating?: number; maxRating?: number }) => {
  const queryParams: Record<string, string> = { q: query };
  if (params?.federation) queryParams.federation = params.federation;
- if (params?.minRating !== undefined) queryParams.minRating = String(params.minRating);
- if (params?.maxRating !== undefined) queryParams.maxRating = String(params.maxRating);
+ if (params?.minRating !== undefined && params.minRating > 0) {
+   queryParams.minRating = String(params.minRating);
+ }
+ if (params?.maxRating !== undefined && params.maxRating > 0) {
+   queryParams.maxRating = String(params.maxRating);
+ }
  return api.get<{ players: Player[]; total: number }>('/players', queryParams);
},
```

---

## Bug #7: Unsafe Type Casting in tournamentsApi.list

**Description:**
The `tournamentsApi.list` function accepts parameters with types `status?: string; page?: number; limit?: number` but then casts the entire params object to `Record<string, string>` without converting the numeric values to strings. While JavaScript will coerce numbers to strings in most contexts, this is type-unsafe and could cause issues if the underlying fetch implementation is stricter.

**Location:**
- File: `/home/user/openpairing/frontend/src/lib/api.ts`
- Function: `tournamentsApi.list`
- Lines: 412-413

**Steps to Reproduce:**
1. Call `tournamentsApi.list({ page: 2, limit: 20 })`
2. The params object contains numbers but is cast to `Record<string, string>`
3. Expected: Proper type conversion or error
4. Actual: Relies on implicit type coercion which may fail

**Fix:**
Properly convert numeric parameters to strings:

```diff
list: (params?: { status?: string; page?: number; limit?: number }) => {
- api.get<{ tournaments: Tournament[]; total: number }>('/tournaments', params as Record<string, string>),
+ const queryParams: Record<string, string> = {};
+ if (params?.status) queryParams.status = params.status;
+ if (params?.page !== undefined) queryParams.page = String(params.page);
+ if (params?.limit !== undefined) queryParams.limit = String(params.limit);
+ return api.get<{ tournaments: Tournament[]; total: number }>('/tournaments', queryParams);
```

---

## Bug #8: Error Information Loss on JSON Parse Failure

**Description:**
When JSON parsing fails in the API client (line 105-114), the function catches the error and returns a generic "Invalid JSON response from server" message. However, the actual response text is lost, making it impossible to debug what the server actually returned. This is particularly problematic in production when diagnosing API integration issues.

**Location:**
- File: `/home/user/openpairing/frontend/src/lib/api.ts`
- Function: `request`
- Lines: 103-114

**Steps to Reproduce:**
1. Backend returns `Content-Type: application/json` but body is invalid JSON like `{bad json`
2. Frontend makes a request
3. Expected: Error message includes what was received
4. Actual: Generic error "Invalid JSON response from server" with no details

**Fix:**
Include the response text in the error or log it for debugging:

```diff
if (contentType?.includes('application/json')) {
+ let rawText: string | undefined;
  try {
    data = await response.json();
  } catch {
+   // Try to get the raw response for debugging
+   try {
+     rawText = await response.text();
+   } catch {
+     // Even text reading failed
+   }
    return {
      status,
      error: {
-       message: 'Invalid JSON response from server',
+       message: rawText
+         ? `Invalid JSON response from server: ${rawText.substring(0, 200)}`
+         : 'Invalid JSON response from server',
        code: 'PARSE_ERROR',
      },
    };
  }
}
```

Note: Be careful not to read the response stream twice. Better approach:

```diff
if (contentType?.includes('application/json')) {
+ const rawText = await response.text();
  try {
-   data = await response.json();
+   data = JSON.parse(rawText);
  } catch {
    return {
      status,
      error: {
-       message: 'Invalid JSON response from server',
+       message: `Invalid JSON response: ${rawText.substring(0, 200)}`,
        code: 'PARSE_ERROR',
      },
    };
  }
}
```

---

## Bug #9: truncate Function Edge Case with maxLength = 1

**Description:**
The `truncate` function has an edge case where if `maxLength` is 1, it produces only an ellipsis "…" with no actual text. The calculation `text.slice(0, maxLength - 1)` becomes `text.slice(0, 0)` which returns an empty string, so the result is just "…". While this might be technically correct (1 character = the ellipsis), it's likely unexpected behavior.

**Location:**
- File: `/home/user/openpairing/frontend/src/lib/utils.ts`
- Function: `truncate`
- Lines: 176-179

**Steps to Reproduce:**
1. Call `truncate("Hello", 1)`
2. Expected: Either "H" or reject invalid maxLength
3. Actual: Returns just "…"

**Fix:**
Add a minimum maxLength validation:

```diff
export function truncate(text: string, maxLength: number): string {
+ if (maxLength < 2) {
+   throw new Error('maxLength must be at least 2 to accommodate ellipsis');
+ }
  if (text.length <= maxLength) return text;
  return text.slice(0, maxLength - 1) + '…';
}
```

Or handle the edge case:

```diff
export function truncate(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text;
+ if (maxLength === 1) return '…';
  return text.slice(0, maxLength - 1) + '…';
}
```

---

## Bug #10: debounce Function Loses `this` Context

**Description:**
The `debounce` function doesn't preserve the `this` context when calling the wrapped function. If a debounced function is used as a class method or needs to access `this`, it will fail or have undefined context. This is a common pitfall in JavaScript function utilities.

**Location:**
- File: `/home/user/openpairing/frontend/src/lib/utils.ts`
- Function: `debounce`
- Lines: 196-206

**Steps to Reproduce:**
1. Create a class with a method: `class Foo { value = 42; method = debounce(() => console.log(this.value), 100); }`
2. Call `new Foo().method()`
3. Expected: Logs 42
4. Actual: `this` is undefined or not bound correctly

**Fix:**
Use a proper `this` binding in the returned function:

```diff
export function debounce<T extends (...args: unknown[]) => unknown>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: NodeJS.Timeout | null = null;

- return function (...args: Parameters<T>) {
+ return function (this: unknown, ...args: Parameters<T>) {
+   const context = this;
    if (timeout) clearTimeout(timeout);
-   timeout = setTimeout(() => func(...args), wait);
+   timeout = setTimeout(() => func.apply(context, args), wait);
  };
}
```

---

## Bug #11: Silent localStorage Failure in getAccessToken

**Description:**
In the `getAccessToken` method, when reading from localStorage fails (line 52-54), the error is silently caught and ignored. While defensive programming is good, this could mask real issues like quota exceeded errors, corrupted data, or permission issues. More importantly, if the localStorage data is corrupted, the user remains in a broken authentication state with no indication of what went wrong.

**Location:**
- File: `/home/user/openpairing/frontend/src/lib/api.ts`
- Function: `getAccessToken`
- Lines: 42-54

**Steps to Reproduce:**
1. Corrupt the localStorage auth data: `localStorage.setItem('openpairing-auth', 'invalid{json}')`
2. Make an authenticated API request
3. Expected: User is notified of authentication issue or re-prompted to login
4. Actual: Request proceeds without token, likely returns 401, but root cause is masked

**Fix:**
Log the error for debugging and optionally clear corrupted data:

```diff
if (typeof window !== 'undefined') {
  try {
    const stored = localStorage.getItem('openpairing-auth');
    if (stored) {
      const parsed = JSON.parse(stored);
      if (parsed?.state?.accessToken) {
        // Also update the local cache
        this.accessToken = parsed.state.accessToken;
        return this.accessToken;
      }
    }
- } catch {
-   // Ignore localStorage errors
+ } catch (error) {
+   // Log the error for debugging
+   console.error('Failed to read auth token from localStorage:', error);
+   // Clear corrupted data to prevent repeated failures
+   try {
+     localStorage.removeItem('openpairing-auth');
+   } catch {
+     // Even removal failed, likely storage is broken
+   }
  }
}
```

---

## Statistics

- **Critical:** 3 bugs (#2, #3, #8)
- **High:** 5 bugs (#1, #4, #6, #7, #11)
- **Medium:** 2 bugs (#5, #10)
- **Low:** 1 bug (#9)

---

## Recommendations

### Immediate Actions
1. **Add input validation**: All date formatting functions should validate inputs before processing
2. **Fix type safety**: Replace unsafe type assertions with proper type guards or explicit handling
3. **Improve error reporting**: Preserve error context and original data for debugging
4. **Add unit tests**: Increase test coverage for edge cases (negative values, null values, invalid inputs)

### Code Quality Improvements
1. **Implement proper type conversions**: Replace implicit casts with explicit conversions
2. **Add error logging**: Use a proper logging library to track silent failures
3. **Document edge cases**: Add JSDoc comments explaining how functions handle edge cases
4. **Add runtime validation**: Use zod or similar for validating API responses match expected types

### Security Enhancements
1. **Monitor authentication failures**: Add telemetry for localStorage corruption or token refresh failures
2. **Implement error boundaries**: Add global error handling for API client failures
3. **Add request timeouts**: Prevent hanging requests from blocking the UI
4. **Sanitize error messages**: Ensure error messages shown to users don't leak sensitive information

### Testing Strategy
1. Create comprehensive test suites for all utility functions covering edge cases
2. Add integration tests for API client with mock responses
3. Test error scenarios: invalid JSON, network failures, corrupted localStorage
4. Test concurrent scenarios: multiple simultaneous API calls with token refresh
