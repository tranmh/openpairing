# OpenPairing Frontend - Bug Report from E2E Tests

**Test Date:** 2026-01-06
**Test Method:** Playwright E2E Tests

## Critical Bugs (Fixed)

### BUG-01: Missing QueryClientProvider in Production
**Status:** FIXED
**Severity:** Critical
**Location:** `/opt/openpairing/frontend/src/app/layout.tsx`
**Description:** Production layout.tsx was missing the `<Providers>` wrapper that includes QueryClientProvider. This caused "No QueryClient set" error on all pages using React Query.
**Fix:** Added `<Providers>{children}</Providers>` wrapper to the root layout.

### BUG-02: Login Redirects to Non-existent /dashboard
**Status:** FIXED
**Severity:** Critical
**Location:** Multiple files
**Description:** After login, users were redirected to `/dashboard` which returned 404.
**Files Fixed:**
- `src/app/(auth)/login/page.tsx` - Changed default returnUrl from `/dashboard` to `/tournaments`
- `src/hooks/useAuth.ts` - Changed useRedirectIfAuthenticated default to `/tournaments`
- `src/hooks/useAuth.ts` - Changed useRequireRole default to `/tournaments`

### BUG-03: Broken Navigation Links on Home Page
**Status:** FIXED
**Severity:** High
**Location:** `src/app/(dashboard)/home/page.tsx`
**Description:** All links pointed to non-existent `/dashboard/*` routes.
**Fix:** Updated all links to correct routes (`/tournaments`, `/profile`, etc.)

### BUG-04: Missing Translation Key nav.home
**Status:** FIXED
**Severity:** Medium
**Location:** `src/i18n/de.json`, `src/i18n/en.json`
**Description:** Navigation showed `nav.home` instead of translated text.
**Fix:** Added `nav.home` translation keys.

## Bugs Found by Tests (Needs Investigation)

### BUG-05: Navigation "Turniere" Link Doesn't Navigate
**Status:** OPEN
**Severity:** Medium
**Test:** BUG-11: Navigation consistency check
**Description:** Clicking "Turniere" navigation item stays on `/home` instead of navigating to `/tournaments`.
**Console Output:**
```
Navigation to Home: OK (http://localhost:3000/home)
Navigation to Turniere: FAILED (http://localhost:3000/home)
```

### BUG-06: Tournament Creation Form - Type Selector
**Status:** OPEN
**Severity:** Medium
**Tests:** All tournament workflow tests
**Description:** Tests couldn't find `button:has-text("Typ wählen")` - the tournament type dropdown appears to use different text or have a default value pre-selected.
**Note:** May be a test locator issue, not a bug.

### BUG-07: Tournament Creation Form - Unstable Button
**Status:** NEEDS INVESTIGATION
**Severity:** Medium
**Tests:** BUG-16, BUG-17, Tournament creation persistence
**Description:** "Turnier erstellen" button becomes detached from DOM during test interaction.
**Playwright Log:**
```
- element was detached from the DOM, retrying
- element is not stable
```

## Test Results Summary

| Test | Status | Issue |
|------|--------|-------|
| Login and redirect | PASS | Fixed QueryClient issue |
| Navigation consistency | PASS (with bugs logged) | nav.home missing |
| Tournament list page | PASS | - |
| Login redirect behavior | PASS | - |
| Swiss tournament workflow | FAIL | Timeout on type selector |
| Round Robin workflow | FAIL | Timeout on type selector |
| WAM workflow | FAIL | Timeout on type selector |
| Form validation | FAIL | Timeout on form elements |
| Date picker | FAIL | Timeout |
| Arbiter console loading | FAIL | Button detached |
| Result entry dialog | FAIL | Button detached |
| Tournament persistence | FAIL | Button detached |

## Production Deployment Issues Found

1. **Out-of-sync source code**: Production directory `/opt/openpairing/frontend/` had older source files than development directory
2. **Missing dependencies**: Production was missing several `@radix-ui/*` packages
3. **Package.json mismatch**: Production package.json didn't match development

## Recommended Actions

1. Set up proper deployment pipeline (CI/CD) to keep production in sync with development
2. Add pre-deployment build verification
3. Fix unstable form components (potential re-render issues)
4. Update test locators to match actual UI text
5. Investigate why "Turniere" navigation click doesn't work

## Files Modified During Bug Fixes

- `src/app/layout.tsx` - Added Providers wrapper
- `src/app/(auth)/login/page.tsx` - Fixed redirect URL
- `src/app/(dashboard)/home/page.tsx` - Fixed all /dashboard links
- `src/hooks/useAuth.ts` - Fixed default redirects
- `src/components/layout/DashboardLayout.tsx` - Fixed navigation links
- `src/i18n/de.json` - Added nav.home translation
- `src/i18n/en.json` - Added nav.home translation
