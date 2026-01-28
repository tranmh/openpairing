# Bug Report: auth

**Analysis Date:** 2026-01-28
**Directory Analyzed:** `/home/user/openpairing/backend/internal/auth`
**Files Analyzed:** 6 files (handler.go, service.go, jwt.go, middleware.go, handler_test.go, service_test.go)
**Bugs Found:** 12 bugs

---

## Summary

The authentication module contains several critical security vulnerabilities and functional bugs that could compromise the security of the entire application. The most severe issues include:
- **Insecure cookie configuration when behind a reverse proxy** (affects production deployments)
- **Timing attack vulnerability in login** (allows email enumeration)
- **Type assertion panics** (can crash the application)
- **Account enumeration via registration endpoint**
- **Interface signature mismatch** (breaks test compatibility)
- **Context misuse in goroutines** (resource leaks)
- **Missing rate limiting** (brute force vulnerability)

Most critical issues are in the cookie security handling and authentication flow. The timing attack and cookie security bugs are high-priority security vulnerabilities that need immediate attention.

---

## Bugs

## Bug #1: Insecure Cookie Configuration Behind Reverse Proxy

**Description:**
The `setSecureCookie` function and all cookie-setting code use `c.Request.TLS != nil` to determine if the `Secure` flag should be set on cookies. When the application runs behind a reverse proxy (nginx, load balancer, cloud provider), `c.Request.TLS` is always `nil` because the TLS termination happens at the proxy level. This causes all cookies to be set with `Secure: false`, making them vulnerable to interception over insecure HTTP connections. This is a **CRITICAL** security vulnerability in production deployments.

**Location:**
- File: `/home/user/openpairing/backend/internal/auth/handler.go`
- Function: `setSecureCookie`, `Login`, `Logout`, `RefreshToken`
- Lines: 22-37 (function definition), 130, 152, 175, 184 (usage)

**Steps to Reproduce:**
1. Deploy the application behind nginx or any reverse proxy with TLS termination
2. User logs in via HTTPS
3. Check the `Set-Cookie` header in the response
4. Expected: `Secure` flag is set
5. Actual: `Secure` flag is NOT set because `c.Request.TLS == nil`
6. Attacker on same network can intercept cookie over HTTP

**Fix:**
Use a configuration setting or check the `X-Forwarded-Proto` header to determine if the original request was HTTPS:

```diff
-func setSecureCookie(c *gin.Context, name, value string, maxAge int, path string, secure, httpOnly bool) {
+func setSecureCookie(c *gin.Context, name, value string, maxAge int, path string, httpOnly bool, isProduction bool) {
+	// Determine if connection is secure by checking X-Forwarded-Proto header
+	// or using a configuration setting
+	secure := isProduction || c.GetHeader("X-Forwarded-Proto") == "https" || c.Request.TLS != nil
+
 	sameSite := http.SameSiteLaxMode
 	if secure {
 		sameSite = http.SameSiteStrictMode
 	}
```

And update all callers:

```diff
-setSecureCookie(c, "refresh_token", tokens.RefreshToken, 60*60*24*7, "/", c.Request.TLS != nil, true)
+setSecureCookie(c, "refresh_token", tokens.RefreshToken, 60*60*24*7, "/", true, h.isProduction)
```

Alternatively, use a configuration setting that explicitly defines whether cookies should be secure based on the deployment environment.

---

## Bug #2: Timing Attack Vulnerability in Login Function

**Description:**
The `Login` function in `service.go` has a timing attack vulnerability that allows attackers to enumerate valid email addresses. When an email doesn't exist, the function returns immediately after the database query. When an email exists but the password is wrong, it returns after the bcrypt comparison, which takes significantly longer (bcrypt is intentionally slow, ~100ms). An attacker can measure response times to determine if an email exists in the system, even without knowing the password.

**Location:**
- File: `/home/user/openpairing/backend/internal/auth/service.go`
- Function: `Login`
- Lines: 290-328

**Steps to Reproduce:**
1. Send login request with non-existent email: measure response time (~5ms)
2. Send login request with valid email but wrong password: measure response time (~100ms)
3. Repeat multiple times to eliminate network variance
4. Expected: Both responses take similar time
5. Actual: Password validation takes ~20x longer, revealing if email exists

**Fix:**
Always perform bcrypt comparison even when the user doesn't exist, using a dummy hash:

```diff
 func (s *Service) Login(ctx context.Context, req *LoginRequest) (*User, *TokenPair, error) {
+	// Dummy hash for timing attack mitigation
+	dummyHash := "$2a$12$abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
+
 	// Get user
 	var user User
 	err := s.db.QueryRow(ctx, `
 		SELECT id, email, password_hash, role, preferred_language, email_verified,
 		       email_verified_at, last_login_at, created_at, updated_at
 		FROM users WHERE email = $1
 	`, req.Email).Scan(
 		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.PreferredLanguage,
 		&user.EmailVerified, &user.EmailVerifiedAt, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
 	)
+
+	// If user not found, still perform bcrypt to prevent timing attacks
+	passwordHash := user.PasswordHash
 	if err != nil {
 		if errors.Is(err, pgx.ErrNoRows) {
-			return nil, nil, ErrInvalidCredentials
+			passwordHash = dummyHash
+		} else {
+			return nil, nil, err
 		}
-		return nil, nil, err
 	}

 	// Verify password
-	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
+	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
 		return nil, nil, ErrInvalidCredentials
 	}
+
+	// Double-check user was actually found (after bcrypt timing)
+	if user.ID == "" {
+		return nil, nil, ErrInvalidCredentials
+	}
```

This ensures constant-time behavior regardless of whether the email exists.

---

## Bug #3: Unsafe Type Assertion Can Cause Panic

**Description:**
The `GetCurrentUser` handler performs an unsafe type assertion `userID.(string)` without checking if the assertion succeeds. If the middleware is misconfigured or bypassed, or if another piece of code sets `user_id` to a non-string value in the context, the application will panic and crash. While the `exists` check guards against missing values, it doesn't protect against wrong types.

**Location:**
- File: `/home/user/openpairing/backend/internal/auth/handler.go`
- Function: `GetCurrentUser`, `ResendVerificationEmail`
- Lines: 286, 356

**Steps to Reproduce:**
1. Modify middleware to set `user_id` as an integer: `c.Set("user_id", 12345)`
2. Make request to `/auth/me`
3. Expected: Return error message
4. Actual: Application panics with "interface conversion: interface {} is int, not string"

**Fix:**
Use safe type assertion with the comma-ok idiom:

```diff
 func (h *Handler) GetCurrentUser(c *gin.Context) {
 	userID, exists := c.Get("user_id")
 	if !exists {
 		response.Unauthorized(c, "Not authenticated")
 		return
 	}
+
+	userIDStr, ok := userID.(string)
+	if !ok {
+		response.InternalServerError(c, "Invalid user ID in context")
+		return
+	}

-	user, err := h.service.GetUserByID(c.Request.Context(), userID.(string))
+	user, err := h.service.GetUserByID(c.Request.Context(), userIDStr)
```

Apply the same fix to line 356 in `ResendVerificationEmail`.

---

## Bug #4: Account Enumeration via Registration Endpoint

**Description:**
The registration endpoint returns a specific error message "A user with this email already exists" when attempting to register with an existing email. This allows attackers to enumerate valid email addresses in the system, which can be used for targeted phishing attacks or to build a list of users for other attacks. The `ForgotPassword` endpoint correctly hides this information, but `Register` does not.

**Location:**
- File: `/home/user/openpairing/backend/internal/auth/handler.go`
- Function: `Register`
- Lines: 87-90

**Steps to Reproduce:**
1. Register a user with email `existing@example.com`
2. Attempt to register another user with the same email
3. Expected: Generic error message that doesn't reveal if email exists
4. Actual: Specific error "A user with this email already exists"
5. Attacker can enumerate all registered emails

**Fix:**
Return a generic error message and optionally send an email to the existing user:

```diff
 	user, err := h.service.Register(c.Request.Context(), &req)
 	if err != nil {
 		switch err {
 		case ErrUserExists:
-			response.Conflict(c, "A user with this email already exists")
+			// Don't reveal if email exists - return generic message
+			response.Success(c, http.StatusOK, gin.H{
+				"message": "If this email is not already registered, you will receive a confirmation email",
+			})
 		default:
 			response.InternalServerError(c, "Failed to register user")
 		}
 		return
 	}
```

Optionally, send an email to existing users informing them of the registration attempt for security awareness.

---

## Bug #5: Context Leak in Goroutine for Email Sending

**Description:**
In the `Register` function, a goroutine is spawned to send verification emails asynchronously. The goroutine replaces the request context with `context.Background()`, which means the original request's cancellation signal is lost. If the HTTP request is cancelled (client disconnects), the email sending goroutine continues to run, wasting resources. Over time, this can lead to goroutine accumulation and resource exhaustion.

**Location:**
- File: `/home/user/openpairing/backend/internal/auth/service.go`
- Function: `Register`
- Lines: 147-153

**Steps to Reproduce:**
1. Start registration request
2. Client disconnects before email is sent
3. Expected: Email sending is cancelled when request context is cancelled
4. Actual: Email sending continues with `context.Background()`, ignoring cancellation
5. Goroutine runs to completion even though client is gone

**Fix:**
Create a timeout context derived from a background context instead of replacing it entirely:

```diff
 	// Send verification email
 	if s.emailService != nil {
 		go func() {
-			ctx := context.Background()
-			_ = s.sendVerificationEmail(ctx, userID, req.Email, req.FirstName)
+			// Use a timeout context to prevent goroutine leaks
+			// Don't use the request context as it may be cancelled
+			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
+			defer cancel()
+			if err := s.sendVerificationEmail(ctx, userID, req.Email, req.FirstName); err != nil {
+				log.Error().Err(err).Str("user_id", userID).Msg("Failed to send verification email in background")
+			}
 		}()
 	}
```

This ensures the goroutine has a timeout and doesn't run indefinitely, while still being decoupled from the request lifecycle.

---

## Bug #6: Silent Failure When Invalidating Old Verification Tokens

**Description:**
In `ResendVerificationEmail`, when invalidating existing verification tokens, the error from the database operation is silently ignored using `_`. If this operation fails (e.g., database connection issue, permission problem), old tokens remain valid while new ones are created. This allows multiple valid tokens to exist simultaneously, which could be exploited if an attacker obtains an old token.

**Location:**
- File: `/home/user/openpairing/backend/internal/auth/service.go`
- Function: `ResendVerificationEmail`
- Lines: 217-221

**Steps to Reproduce:**
1. Request verification email
2. Database has permission issue or connection failure when updating tokens
3. `UPDATE` fails but error is ignored
4. New token is created
5. Expected: Old token is invalidated OR operation fails
6. Actual: Both old and new tokens are valid

**Fix:**
Check and handle the error properly:

```diff
 	// Invalidate existing tokens
-	_, _ = s.db.Exec(ctx, `
+	_, err = s.db.Exec(ctx, `
 		UPDATE email_verification_tokens SET used_at = NOW()
 		WHERE user_id = $1 AND used_at IS NULL
 	`, userID)
+	if err != nil {
+		log.Error().Err(err).Str("user_id", userID).Msg("Failed to invalidate old verification tokens")
+		return err
+	}
```

This ensures that if token invalidation fails, the operation is aborted rather than creating duplicate valid tokens.

---

## Bug #7: Token Generation Failure After Revoking Refresh Token

**Description:**
In `RefreshTokens`, the old refresh token is revoked and committed to the database before generating new tokens. If token generation fails after the commit (lines 394-401), the user is left without any valid refresh tokens and must re-login. While this is mentioned in a comment as "safer," it creates a poor user experience and could be exploited in a denial-of-service attack by repeatedly triggering token generation failures.

**Location:**
- File: `/home/user/openpairing/backend/internal/auth/service.go`
- Function: `RefreshTokens`
- Lines: 389-401

**Steps to Reproduce:**
1. User has valid refresh token
2. User requests token refresh
3. Old token is revoked and transaction committed (line 390)
4. `generateTokenPair` fails due to database issue (line 395)
5. Expected: User can retry with same token
6. Actual: Old token revoked, new token not created, user must re-login

**Fix:**
Generate new tokens before revoking the old one, using a two-phase approach:

```diff
-	// Commit the transaction (revokes the old token)
-	if err = tx.Commit(ctx); err != nil {
-		return nil, nil, err
-	}
-
-	// Generate new tokens (outside transaction - this creates a new token)
+	// Generate new tokens BEFORE revoking old token to ensure atomicity
+	// If this fails, transaction rolls back and old token remains valid
 	tokens, err := s.generateTokenPair(ctx, &user)
 	if err != nil {
-		// Note: The old token is already revoked at this point.
-		// If token generation fails, user will need to re-login.
-		// This is safer than leaving the old token valid.
+		// Transaction will rollback, old token stays valid
 		return nil, nil, err
 	}
+
+	// Now commit the transaction to revoke the old token
+	// If this fails, token generation is wasted but no data inconsistency
+	if err = tx.Commit(ctx); err != nil {
+		return nil, nil, err
+	}
```

Note: This requires `generateTokenPair` to accept a transaction parameter or be split into token generation and storage phases.

---

## Bug #8: Missing Rate Limiting on Login Attempts

**Description:**
The authentication module has no rate limiting on login attempts. An attacker can make unlimited login attempts, enabling brute force attacks on user accounts. Combined with Bug #2 (timing attack), this makes the system highly vulnerable to credential stuffing and password guessing attacks. Even without Bug #2, brute force is possible with enough attempts.

**Location:**
- File: `/home/user/openpairing/backend/internal/auth/handler.go`
- Function: `Login`
- Lines: 109-137

**Steps to Reproduce:**
1. Attacker identifies valid email (via Bug #4 or #2)
2. Attacker writes script to attempt 10,000 common passwords
3. No rate limiting prevents the attempts
4. Expected: Requests blocked after N failed attempts
5. Actual: All 10,000 attempts processed, one may succeed

**Fix:**
Implement rate limiting using Redis or in-memory store:

```diff
 func (h *Handler) Login(c *gin.Context) {
 	var req LoginRequest
 	if err := c.ShouldBindJSON(&req); err != nil {
 		response.BadRequest(c, "Invalid request body")
 		return
 	}

 	req.Email = validator.SanitizeString(req.Email)
+
+	// Check rate limit: max 5 attempts per email per 15 minutes
+	if h.rateLimiter.IsLimitExceeded(req.Email, 5, 15*time.Minute) {
+		response.TooManyRequests(c, "Too many login attempts. Please try again later.")
+		return
+	}

 	user, tokens, err := h.service.Login(c.Request.Context(), &req)
 	if err != nil {
 		switch err {
 		case ErrInvalidCredentials:
+			h.rateLimiter.IncrementAttempts(req.Email)
 			response.Unauthorized(c, "Invalid email or password")
 		default:
 			response.InternalServerError(c, "Failed to login")
 		}
 		return
 	}
+
+	// Clear rate limit on successful login
+	h.rateLimiter.ClearAttempts(req.Email)
```

Implement a RateLimiter using Redis with sliding window or token bucket algorithm.

---

## Bug #9: Missing Rate Limiting on Password Reset Requests

**Description:**
The `ForgotPassword` endpoint has no rate limiting, allowing attackers to trigger unlimited password reset emails to any email address. This enables email bombing (DoS) attacks against users and can be used to flood email servers. An attacker could weaponize this to harass users or overwhelm the email service.

**Location:**
- File: `/home/user/openpairing/backend/internal/auth/handler.go`
- Function: `ForgotPassword`
- Lines: 201-224

**Steps to Reproduce:**
1. Attacker writes script to request password reset for victim@example.com
2. Script sends 1000 requests per minute
3. No rate limiting prevents the requests
4. Expected: Requests blocked after N attempts
5. Actual: 1000 emails sent to victim, overwhelming their inbox

**Fix:**
Implement IP-based and email-based rate limiting:

```diff
 func (h *Handler) ForgotPassword(c *gin.Context) {
 	var req struct {
 		Email string `json:"email"`
 	}
 	if err := c.ShouldBindJSON(&req); err != nil {
 		response.BadRequest(c, "Invalid request body")
 		return
 	}

 	req.Email = validator.SanitizeString(req.Email)

 	if !validator.ValidateEmail(req.Email) {
 		response.BadRequest(c, "Invalid email address")
 		return
 	}
+
+	// Rate limit: max 3 requests per email per hour AND max 10 per IP per hour
+	clientIP := c.ClientIP()
+	if h.rateLimiter.IsLimitExceeded(req.Email, 3, time.Hour) ||
+	   h.rateLimiter.IsLimitExceeded(clientIP, 10, time.Hour) {
+		// Still return success to prevent email enumeration
+		response.Success(c, http.StatusOK, gin.H{
+			"message": "If an account with that email exists, a password reset link has been sent",
+		})
+		return
+	}
+
+	h.rateLimiter.IncrementAttempts(req.Email)
+	h.rateLimiter.IncrementAttempts(clientIP)
```

---

## Bug #10: Interface Signature Mismatch in Tests

**Description:**
The `AuthService` interface in `handler_test.go` defines `RequestPasswordReset` as returning `(*User, error)`, but the actual implementation in `service.go` returns `(string, error)`. This is a **compilation error** - the test interface doesn't match the real implementation. Any test that tries to use the real service with this interface will fail to compile.

**Location:**
- File: `/home/user/openpairing/backend/internal/auth/handler_test.go`
- Function: `RequestPasswordReset` (interface definition)
- Line: 24
- File: `/home/user/openpairing/backend/internal/auth/service.go`
- Function: `RequestPasswordReset` (implementation)
- Line: 434

**Steps to Reproduce:**
1. Try to assign real `Service` to `AuthService` interface: `var svc AuthService = realService`
2. Compilation fails: "method RequestPasswordReset has wrong signature"
3. Expected: Interface matches implementation
4. Actual: Signatures don't match

**Fix:**
Update the test interface to match the actual implementation:

```diff
 type AuthService interface {
 	Register(ctx context.Context, req *RegisterRequest) (*User, error)
 	Login(ctx context.Context, req *LoginRequest) (*User, *TokenPair, error)
 	Logout(ctx context.Context, refreshToken string) error
 	RefreshTokens(ctx context.Context, refreshToken string) (*User, *TokenPair, error)
-	RequestPasswordReset(ctx context.Context, email string) (*User, error)
+	RequestPasswordReset(ctx context.Context, email string) (string, error)
 	ResetPassword(ctx context.Context, token, newPassword string) error
 	GetUserByID(ctx context.Context, id string) (*User, error)
 	VerifyEmail(ctx context.Context, token string) error
 	ResendVerificationEmail(ctx context.Context, userID string) error
 }
```

And update the mock implementation:

```diff
 type MockAuthService struct {
 	RegisterFunc              func(ctx context.Context, req *RegisterRequest) (*User, error)
 	LoginFunc                 func(ctx context.Context, req *LoginRequest) (*User, *TokenPair, error)
 	LogoutFunc                func(ctx context.Context, refreshToken string) error
 	RefreshTokensFunc         func(ctx context.Context, refreshToken string) (*User, *TokenPair, error)
-	RequestPasswordResetFunc  func(ctx context.Context, email string) (*User, error)
+	RequestPasswordResetFunc  func(ctx context.Context, email string) (string, error)
 	ResetPasswordFunc         func(ctx context.Context, token, newPassword string) error
 	GetUserByIDFunc           func(ctx context.Context, id string) (*User, error)
 	VerifyEmailFunc           func(ctx context.Context, token string) error
 	ResendVerificationFunc    func(ctx context.Context, userID string) error
 }
```

---

## Bug #11: No Whitespace Trimming in Token Extraction

**Description:**
The `extractToken` function in middleware doesn't trim whitespace from the extracted token. If a client sends an Authorization header with extra whitespace (e.g., `"Bearer  token123"` with double space, or `"Bearer token123 "` with trailing space), the token string will include that whitespace and fail validation, even though it's a valid token. This creates a poor user experience and unnecessary authentication failures.

**Location:**
- File: `/home/user/openpairing/backend/internal/auth/middleware.go`
- Function: `extractToken`
- Lines: 116-129

**Steps to Reproduce:**
1. Client sends valid JWT token with trailing space: `Authorization: Bearer eyJhbGc...xyz `
2. Token extracted as `"eyJhbGc...xyz "` (with space)
3. JWT validation fails because of extra character
4. Expected: Token trimmed and validated successfully
5. Actual: Authentication fails with "Invalid token"

**Fix:**
Trim whitespace from the extracted token:

```diff
 func extractToken(c *gin.Context) string {
 	authHeader := c.GetHeader("Authorization")
 	if authHeader == "" {
 		return ""
 	}

 	// Check for "Bearer " prefix
 	parts := strings.SplitN(authHeader, " ", 2)
 	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
 		return ""
 	}

-	return parts[1]
+	return strings.TrimSpace(parts[1])
 }
```

---

## Bug #12: Missing CSRF Protection for State-Changing Operations

**Description:**
While the authentication system correctly uses HTTP-only cookies for refresh tokens (preventing XSS attacks), there's no CSRF (Cross-Site Request Forgery) protection for state-changing operations. An attacker could create a malicious website that triggers authenticated requests to change user data, reset passwords, or perform other actions on behalf of logged-in users. The `SameSite` cookie attribute provides partial protection but is not sufficient on its own (SameSite=Lax still allows top-level navigation POST requests).

**Location:**
- Files: Multiple handlers in `/home/user/openpairing/backend/internal/auth/handler.go`
- Functions: All POST endpoints (Login, Logout, Register, ForgotPassword, ResetPassword, etc.)
- Lines: Throughout handler.go

**Steps to Reproduce:**
1. User logs into application at `app.example.com`
2. User visits attacker's website `evil.com`
3. `evil.com` contains form that auto-submits to `app.example.com/auth/forgot-password`
4. Browser includes authentication cookies with the request (even with SameSite=Lax for top-level navigation)
5. Expected: Request rejected due to missing CSRF token
6. Actual: Password reset email sent to user without their knowledge

**Fix:**
Implement CSRF token protection for all state-changing endpoints:

```diff
+// middleware.go
+func (m *Middleware) RequireCSRF() gin.HandlerFunc {
+	return func(c *gin.Context) {
+		// Skip CSRF check for safe methods
+		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
+			c.Next()
+			return
+		}
+
+		// Extract CSRF token from header or form
+		csrfToken := c.GetHeader("X-CSRF-Token")
+		if csrfToken == "" {
+			csrfToken = c.PostForm("csrf_token")
+		}
+
+		// Validate token
+		sessionToken, _ := c.Cookie("session_token")
+		if !m.validateCSRFToken(csrfToken, sessionToken) {
+			response.Forbidden(c, "Invalid CSRF token")
+			c.Abort()
+			return
+		}
+
+		c.Next()
+	}
+}
```

Apply this middleware to all authentication routes and include CSRF token generation in login responses.

---

## Statistics

- **Critical:** 3 bugs (Bug #1: Cookie security, Bug #2: Timing attack, Bug #8: No rate limiting on login)
- **High:** 5 bugs (Bug #3: Type assertion panic, Bug #4: Account enumeration, Bug #7: Token generation order, Bug #9: No rate limiting on password reset, Bug #12: Missing CSRF)
- **Medium:** 3 bugs (Bug #5: Context leak, Bug #6: Silent failure, Bug #10: Interface mismatch)
- **Low:** 1 bug (Bug #11: Whitespace handling)

---

## Recommendations

### Immediate Actions (Critical Priority)
1. **Fix Cookie Security (Bug #1)**: Deploy hotfix to read secure flag from configuration or X-Forwarded-Proto header
2. **Fix Timing Attack (Bug #2)**: Implement constant-time login to prevent email enumeration
3. **Add Rate Limiting (Bug #8, #9)**: Implement Redis-based rate limiting for login and password reset endpoints

### Short-term Improvements (High Priority)
4. **Fix Type Assertions (Bug #3)**: Add safe type assertions to prevent panics
5. **Fix Account Enumeration (Bug #4)**: Change registration endpoint to not reveal existing emails
6. **Implement CSRF Protection (Bug #12)**: Add CSRF tokens to all state-changing operations
7. **Fix Token Generation Order (Bug #7)**: Generate new tokens before revoking old ones

### Medium-term Improvements
8. **Fix Context Handling (Bug #5)**: Use proper timeout contexts in goroutines
9. **Fix Error Handling (Bug #6)**: Check errors when invalidating tokens
10. **Fix Test Interface (Bug #10)**: Update test interfaces to match implementations
11. **Improve Token Parsing (Bug #11)**: Trim whitespace from tokens

### Long-term Security Improvements
- Implement account lockout after N failed login attempts
- Add email notification when account is accessed from new device/location
- Implement 2FA (TOTP or SMS) for sensitive operations
- Add security headers (CSP, HSTS, X-Frame-Options)
- Implement session management with device tracking
- Add honeypot fields to registration to catch bots
- Implement proper audit logging for all authentication events
- Consider using OAuth2/OIDC for better security architecture
