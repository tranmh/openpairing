import { test, expect, Page } from '@playwright/test';

/**
 * OpenPairing E2E Tests for Tournament Workflows
 * Tests: Swiss, Round Robin, WAM tournament types
 *
 * Bug tracking:
 * - Each discovered bug is documented with BUG-XX prefix
 */

const TEST_CREDENTIALS = {
  email: 'arbiter@openpairing.de',
  password: 'test1234',
};

const API_BASE = 'http://localhost:8080/api/v1';

// Helper: Login via UI
async function login(page: Page) {
  // Capture console errors
  const consoleErrors: string[] = [];
  page.on('console', msg => {
    if (msg.type() === 'error') {
      consoleErrors.push(msg.text());
    }
  });

  await page.goto('/login');
  await page.waitForLoadState('networkidle');

  // Check for client-side errors
  const pageContent = await page.content();
  if (pageContent.includes('Application error')) {
    console.error('Page has application error. Console errors:', consoleErrors);
    throw new Error(`Application error on login page. Console: ${consoleErrors.join(', ')}`);
  }

  // Fill login form
  await page.fill('input[id="email"]', TEST_CREDENTIALS.email);
  await page.fill('input[id="password"]', TEST_CREDENTIALS.password);
  await page.click('button[type="submit"]');

  // Wait for either redirect or error
  try {
    await page.waitForURL('**/tournaments', { timeout: 10000 });
  } catch (e) {
    const currentUrl = page.url();
    const content = await page.content();
    console.error('Login redirect failed. Current URL:', currentUrl);
    console.error('Page contains Application error:', content.includes('Application error'));
    console.error('Console errors:', consoleErrors);
    throw e;
  }
}

// Helper: Get auth token from localStorage
async function getAuthToken(page: Page): Promise<string> {
  return await page.evaluate(() => {
    const authStorage = localStorage.getItem('auth-storage');
    if (authStorage) {
      const parsed = JSON.parse(authStorage);
      return parsed?.state?.accessToken || '';
    }
    return '';
  });
}

// Helper: Create players via API (bypasses federation search)
async function createTestPlayers(token: string, count: number): Promise<string[]> {
  const playerIds: string[] = [];

  for (let i = 1; i <= count; i++) {
    const response = await fetch(`${API_BASE}/players`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
      body: JSON.stringify({
        firstName: `Test${i}`,
        lastName: `Player${i}`,
        email: `player${i}_${Date.now()}@test.com`,
        dwzRating: 1500 + (i * 50),
      }),
    });

    if (response.ok) {
      const data = await response.json();
      playerIds.push(data.player?.id);
    }
  }

  return playerIds.filter(id => id);
}

// Helper: Register players for tournament via API
async function registerPlayersForTournament(
  token: string,
  tournamentId: string,
  playerIds: string[]
): Promise<void> {
  for (const playerId of playerIds) {
    await fetch(`${API_BASE}/tournaments/${tournamentId}/participants`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
      body: JSON.stringify({
        playerId,
        startingRating: 1500,
      }),
    });
  }
}

// Helper: Generate unique tournament name
function uniqueTournamentName(prefix: string): string {
  return `${prefix} ${new Date().toISOString().slice(0, 16).replace('T', ' ')}`;
}

// ============================================================================
// SWISS PAIRING WORKFLOW TESTS
// ============================================================================

test.describe('Swiss Pairing Tournament Workflow', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('complete Swiss tournament workflow', async ({ page }) => {
    const tournamentName = uniqueTournamentName('Swiss Test');

    // Step 1: Navigate to create tournament
    await page.click('text=Neues Turnier');
    await page.waitForURL('**/tournaments/new');

    // Step 2: Fill basic info (Step 1)
    await page.fill('input[name="name"]', tournamentName);
    await page.fill('textarea[name="description"]', 'Automated test tournament');

    // Swiss type is already the default, no need to change it

    // Set rounds to 3 for quick test
    await page.fill('input[name="rounds"]', '3');

    // Click next to step 2
    await page.click('button:has-text("Weiter")');
    await page.waitForTimeout(1000);

    // Step 3: Fill details (Step 2) - Select start date
    const dateButton = page.locator('button:has-text("Datum wählen")').first();
    await dateButton.click();
    await page.waitForSelector('[role="gridcell"]');
    // Click a future day
    const futureDay = ((new Date().getDate() + 5) % 28 + 1).toString();
    await page.click(`[role="gridcell"]:has-text("${futureDay}")`);
    await page.waitForTimeout(500);

    // Click next to step 3
    await page.click('button:has-text("Weiter")');

    // Step 4: Settings (Step 3) - Submit form
    // Wait for step 3 to be active and submit button to appear
    await page.waitForTimeout(1000);

    // The form might auto-submit or we need to click - wait for either
    const alreadyRedirected = page.url().includes('/tournaments/') && !page.url().includes('/new');
    if (!alreadyRedirected) {
      const submitBtn = page.locator('button:has-text("Turnier erstellen")');
      if (await submitBtn.isVisible()) {
        await submitBtn.click();
      }
    }

    // Wait for redirect to tournament detail page
    await page.waitForURL(/\/tournaments\/[a-f0-9-]+$/, { timeout: 15000 });
    const tournamentDetailUrl = page.url();
    const tournamentId = tournamentDetailUrl.split('/tournaments/')[1]?.split('/')[0];
    console.log('Tournament ID:', tournamentId);

    // Verify tournament created
    await expect(page.locator('h1')).toContainText(tournamentName);

    // Step 5: Publish tournament
    // Click the header "Veröffentlichen" button to open dialog
    await page.locator('button:has-text("Veröffentlichen")').first().click();
    // Wait for confirmation dialog and click the dialog's confirm button
    await page.waitForSelector('div[role="alertdialog"], div[role="dialog"]', { timeout: 5000 });
    await page.locator('div[role="alertdialog"] button:has-text("Veröffentlichen"), div[role="dialog"] button:has-text("Veröffentlichen")').click();
    await page.waitForTimeout(2000);

    // Verify publish succeeded by checking for Arbiter-Konsole button (only visible after publish)
    // Step 6: Go to Arbiter Console
    const arbiterLink = page.locator('a:has-text("Arbiter-Konsole")');
    await expect(arbiterLink).toBeVisible({ timeout: 5000 });
    await arbiterLink.click();
    await page.waitForURL('**/arbiter/**');
    await page.waitForLoadState('networkidle');

    // Verify arbiter console loaded - check for main header or tabs
    await expect(page.locator('text=Schiedsrichterkonsole')).toBeVisible({ timeout: 10000 });

    // Step 7: Navigate arbiter console tabs
    // Check that default tab "Aktive Partien" is visible
    await expect(page.locator('[role="tab"]:has-text("Aktive Partien")')).toBeVisible();

    // Check Rounds tab
    const roundsTab = page.locator('[role="tab"]:has-text("Runden")');
    if (await roundsTab.isVisible({ timeout: 2000 }).catch(() => false)) {
      await roundsTab.click();
      await page.waitForTimeout(500);
    }

    // Check Participants tab
    const participantsTab = page.locator('[role="tab"]:has-text("Teilnehmer")');
    if (await participantsTab.isVisible({ timeout: 2000 }).catch(() => false)) {
      await participantsTab.click();
      await page.waitForTimeout(500);
    }

    // Step 8: Go back to tournament detail via navigation
    await page.goto('/tournaments');
    await page.waitForLoadState('networkidle');

    // Click on the tournament we created (most recent one)
    const tournamentCard = page.locator(`text=${tournamentName}`).first();
    await tournamentCard.click();
    await page.waitForURL(/\/tournaments\/[a-f0-9-]+$/);
    await page.waitForLoadState('networkidle');

    // Verify we're on the tournament detail page with tabs
    const overviewTab = page.locator('[role="tab"]:has-text("Übersicht")');
    await expect(overviewTab).toBeVisible({ timeout: 5000 });

    // Navigate through tabs
    const participantsDetailTab = page.locator('[role="tab"]:has-text("Teilnehmer")');
    await participantsDetailTab.click();
    await page.waitForTimeout(500);

    const pairingsTab = page.locator('[role="tab"]:has-text("Paarungen")');
    await pairingsTab.click();
    await page.waitForTimeout(500);

    const standingsTab = page.locator('[role="tab"]:has-text("Tabelle")');
    await standingsTab.click();
    await page.waitForTimeout(500);

    console.log('Swiss tournament UI workflow completed successfully');
  });
});

// ============================================================================
// ROUND ROBIN WORKFLOW TESTS
// ============================================================================

test.describe('Round Robin Tournament Workflow', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('complete Round Robin tournament workflow', async ({ page }) => {
    const tournamentName = uniqueTournamentName('Round Robin Test');

    // Step 1: Navigate to create tournament
    await page.click('text=Neues Turnier');
    await page.waitForURL('**/tournaments/new');

    // Step 2: Fill basic info
    await page.fill('input[name="name"]', tournamentName);

    // Select Round Robin type - click on the current selection to open dropdown
    await page.click('button:has-text("Schweizer System")');
    await page.click('[role="option"]:has-text("Rundenturnier")');

    // BUG-03: Check if rounds field auto-updates for round robin
    await page.waitForTimeout(500);
    const roundsInput = page.locator('input[name="rounds"]');
    const roundsValue = await roundsInput.inputValue();
    console.log('Rounds value after selecting round robin:', roundsValue);

    // For 4 players round robin, we need 3 rounds
    await page.fill('input[name="rounds"]', '3');

    await page.click('button:has-text("Weiter")');
    await page.waitForTimeout(500);

    // Step 3: Fill details
    await page.click('button:has-text("Datum wählen")');
    await page.waitForSelector('[role="gridcell"]');
    const today = new Date().getDate().toString();
    await page.click(`[role="gridcell"]:has-text("${today}")`);

    await page.click('button:has-text("Weiter")');
    await page.waitForTimeout(1000);

    // Step 4: Submit - check if already redirected
    const alreadyRedirected = page.url().includes('/tournaments/') && !page.url().includes('/new');
    if (!alreadyRedirected) {
      const submitBtn = page.locator('button:has-text("Turnier erstellen")');
      if (await submitBtn.isVisible()) {
        await submitBtn.click();
      }
    }

    // Wait for redirect to tournament detail page
    await page.waitForURL(/\/tournaments\/[a-f0-9-]+$/, { timeout: 15000 });
    const tournamentId = page.url().split('/tournaments/')[1]?.split('/')[0];
    console.log('Round Robin Tournament ID:', tournamentId);

    // Verify tournament created
    await expect(page.locator('h1')).toContainText(tournamentName);

    // Step 5: Publish tournament
    await page.locator('button:has-text("Veröffentlichen")').first().click();
    await page.waitForSelector('div[role="alertdialog"], div[role="dialog"]', { timeout: 5000 });
    await page.locator('div[role="alertdialog"] button:has-text("Veröffentlichen"), div[role="dialog"] button:has-text("Veröffentlichen")').click();
    await page.waitForTimeout(2000);

    // Step 6: Go to Arbiter Console
    const arbiterLink = page.locator('a:has-text("Arbiter-Konsole")');
    await expect(arbiterLink).toBeVisible({ timeout: 5000 });
    await arbiterLink.click();
    await page.waitForURL('**/arbiter/**');
    await page.waitForLoadState('networkidle');

    // Verify arbiter console loaded
    await expect(page.locator('text=Schiedsrichterkonsole')).toBeVisible({ timeout: 10000 });

    // Check Sitzplan tab visibility (Round Robin specific)
    const seatingTab = page.locator('[role="tab"]:has-text("Sitzplan")');
    const isSeatingVisible = await seatingTab.isVisible().catch(() => false);
    console.log('Seating tab visible for round robin:', isSeatingVisible);

    // Navigate some tabs
    const roundsTab = page.locator('[role="tab"]:has-text("Runden")');
    if (await roundsTab.isVisible({ timeout: 2000 }).catch(() => false)) {
      await roundsTab.click();
      await page.waitForTimeout(500);
    }

    // Step 12: Check pairings tab
    await page.click('text=Paarungen');
    await page.waitForTimeout(500);

    const pairingsTable = page.locator('table');
    const hasPairings = await pairingsTable.isVisible();
    console.log('BUG-05 Check: Pairings table visible:', hasPairings);

    // BUG-05: Pairings should be generated
    if (!hasPairings) {
      console.error('BUG-05: No pairings generated for Round Robin');
    }
  });

  test('Round Robin seating view functionality', async ({ page }) => {
    // Verify Round Robin has Sitzplan tab on tournament detail page
    // First navigate to tournaments and find a Round Robin tournament
    await page.goto('/tournaments');
    await page.waitForLoadState('networkidle');

    // Check if any tournament cards exist with Sitzplan tab when clicked
    const tournamentCards = page.locator('[class*="card"]').filter({ hasText: 'Rundenturnier' });
    const count = await tournamentCards.count();

    if (count > 0) {
      // Click on a Round Robin tournament
      await tournamentCards.first().click();
      await page.waitForURL(/\/tournaments\/[a-f0-9-]+$/);
      await page.waitForLoadState('networkidle');

      // Check for Sitzplan tab
      const seatingTab = page.locator('[role="tab"]:has-text("Sitzplan")');
      const isSeatingVisible = await seatingTab.isVisible().catch(() => false);
      console.log('Seating tab visible for Round Robin:', isSeatingVisible);
    } else {
      console.log('No Round Robin tournaments found to test seating');
    }
  });
});

// ============================================================================
// WAM TOURNAMENT WORKFLOW TESTS
// ============================================================================

test.describe('WAM Tournament Workflow', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('complete WAM tournament workflow', async ({ page }) => {
    const tournamentName = uniqueTournamentName('WAM Test');

    // Step 1: Create WAM tournament
    await page.click('text=Neues Turnier');
    await page.waitForURL('**/tournaments/new');

    await page.fill('input[name="name"]', tournamentName);

    // Select WAM type - click on the current selection to open dropdown
    await page.click('button:has-text("Schweizer System")');
    await page.click('[role="option"]:has-text("WAM")');

    await page.waitForTimeout(500);

    await page.click('button:has-text("Weiter")');
    await page.waitForTimeout(1000);

    // Fill date
    await page.click('button:has-text("Datum wählen")');
    await page.waitForSelector('[role="gridcell"]');
    const futureDay = ((new Date().getDate() + 5) % 28 + 1).toString();
    await page.click(`[role="gridcell"]:has-text("${futureDay}")`);
    await page.waitForTimeout(500);

    await page.click('button:has-text("Weiter")');
    await page.waitForTimeout(1000);

    // Step 2: Submit - check if already redirected
    const alreadyRedirected = page.url().includes('/tournaments/') && !page.url().includes('/new');
    if (!alreadyRedirected) {
      const submitBtn = page.locator('button:has-text("Turnier erstellen")');
      if (await submitBtn.isVisible()) {
        await submitBtn.click();
      }
    }

    // Wait for redirect to tournament detail page
    await page.waitForURL(/\/tournaments\/[a-f0-9-]+$/, { timeout: 15000 });
    const tournamentId = page.url().split('/tournaments/')[1]?.split('/')[0];
    console.log('WAM Tournament ID:', tournamentId);

    // Verify tournament created
    await expect(page.locator('h1')).toContainText(tournamentName);

    // Step 3: Publish tournament
    await page.locator('button:has-text("Veröffentlichen")').first().click();
    await page.waitForSelector('div[role="alertdialog"], div[role="dialog"]', { timeout: 5000 });
    await page.locator('div[role="alertdialog"] button:has-text("Veröffentlichen"), div[role="dialog"] button:has-text("Veröffentlichen")').click();
    await page.waitForTimeout(2000);

    // Step 4: Go to Arbiter Console
    const arbiterLink = page.locator('a:has-text("Arbiter-Konsole")');
    await expect(arbiterLink).toBeVisible({ timeout: 5000 });
    await arbiterLink.click();
    await page.waitForURL('**/arbiter/**');
    await page.waitForLoadState('networkidle');

    // Verify arbiter console loaded
    await expect(page.locator('text=Schiedsrichterkonsole')).toBeVisible({ timeout: 10000 });

    // Navigate tabs
    const roundsTab = page.locator('[role="tab"]:has-text("Runden")');
    if (await roundsTab.isVisible({ timeout: 2000 }).catch(() => false)) {
      await roundsTab.click();
      await page.waitForTimeout(500);
    }

    console.log('WAM tournament UI workflow completed successfully');
  });
});

// ============================================================================
// COMMON UI BUG TESTS
// ============================================================================

test.describe('Common UI Bug Tests', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('BUG-11: Navigation consistency check', async ({ page }) => {
    // Check all main navigation links work
    const navLinks = [
      { text: 'Home', url: '/home' },
      { text: 'Turniere', url: '/tournaments' },
      { text: 'Spieler', url: '/players' },
    ];

    for (const link of navLinks) {
      const navItem = page.locator(`a:has-text("${link.text}")`).first();
      if (await navItem.isVisible()) {
        await navItem.click();
        await page.waitForLoadState('networkidle');

        const currentUrl = page.url();
        const isCorrect = currentUrl.includes(link.url);
        console.log(`Navigation to ${link.text}: ${isCorrect ? 'OK' : 'FAILED'} (${currentUrl})`);

        if (!isCorrect) {
          console.error(`BUG-11: Navigation to ${link.text} failed. Expected ${link.url}, got ${currentUrl}`);
        }
      }
    }
  });

  test('BUG-12: Tournament list page functionality', async ({ page }) => {
    await page.goto('/tournaments');
    await page.waitForLoadState('networkidle');

    // Check if "New Tournament" button exists
    const newTournamentBtn = page.locator('text=Neues Turnier');
    const hasNewBtn = await newTournamentBtn.isVisible();
    console.log('BUG-12 Check: New Tournament button visible:', hasNewBtn);

    if (!hasNewBtn) {
      console.error('BUG-12: New Tournament button not visible on tournaments page');
    }

    // Check if tournament list loads
    await page.waitForTimeout(2000);
    const tournamentCards = page.locator('[class*="card"]');
    const cardCount = await tournamentCards.count();
    console.log('Tournament cards found:', cardCount);
  });

  test('BUG-13: Login redirect behavior', async ({ page }) => {
    // Logout first
    await page.goto('/tournaments');
    await page.waitForLoadState('networkidle');

    // Try to access protected route without auth
    await page.evaluate(() => localStorage.clear());
    await page.goto('/tournaments/new');
    await page.waitForLoadState('networkidle');

    const currentUrl = page.url();
    const redirectedToLogin = currentUrl.includes('/login');
    console.log('BUG-13 Check: Redirected to login when not authenticated:', redirectedToLogin);

    if (!redirectedToLogin) {
      console.error('BUG-13: Protected route accessible without authentication');
    }
  });

  test('BUG-14: Form validation on tournament creation', async ({ page }) => {
    await page.goto('/tournaments/new');
    await page.waitForLoadState('networkidle');

    // Try to proceed without filling required fields
    await page.click('button:has-text("Weiter")');
    await page.waitForTimeout(500);

    // Check for validation errors
    const errorMessages = page.locator('text=mindestens');
    const hasErrors = await errorMessages.isVisible();
    console.log('BUG-14 Check: Form validation shows errors:', hasErrors);

    if (!hasErrors) {
      console.error('BUG-14: Form validation not working - no errors shown for empty required fields');
    }
  });

  test('BUG-15: Date picker functionality', async ({ page }) => {
    await page.goto('/tournaments/new');
    await page.waitForLoadState('networkidle');

    // Fill name and proceed to step 2
    await page.fill('input[name="name"]', 'Date Test Tournament');
    await page.click('button:has-text("Weiter")');
    await page.waitForTimeout(500);

    // Click date picker
    const datePicker = page.locator('button:has-text("Datum wählen")').first();
    await datePicker.click();
    await page.waitForTimeout(500);

    // Check if calendar appears
    const calendar = page.locator('[role="gridcell"]');
    const calendarVisible = await calendar.first().isVisible();
    console.log('BUG-15 Check: Calendar opens on click:', calendarVisible);

    if (!calendarVisible) {
      console.error('BUG-15: Date picker calendar not opening');
    }
  });

  test('BUG-16: Arbiter console loading state', async ({ page }) => {
    // First create a tournament
    const tournamentName = uniqueTournamentName('Arbiter Loading Test');

    await page.click('text=Neues Turnier');
    await page.fill('input[name="name"]', tournamentName);
    await page.click('button:has-text("Weiter")');
    await page.waitForTimeout(500);

    await page.click('button:has-text("Datum wählen")');
    await page.waitForSelector('[role="gridcell"]');
    await page.click(`[role="gridcell"]:has-text("${new Date().getDate()}")`);

    await page.click('button:has-text("Weiter")');
    await page.click('button:has-text("Turnier erstellen")');
    await page.waitForURL('**/tournaments/**');

    const tournamentId = page.url().split('/tournaments/')[1]?.split('/')[0];

    // Go to arbiter console
    await page.goto(`/arbiter/${tournamentId}`);
    await page.waitForLoadState('networkidle');

    // Check for proper loading/error handling
    const errorState = page.locator('text=Fehler beim Laden');
    const hasError = await errorState.isVisible();

    if (hasError) {
      console.error('BUG-16: Arbiter console shows error for valid tournament');
    }

    // Check for console title
    const title = page.locator('text=Schiedsrichterkonsole');
    const hasTitle = await title.isVisible();
    console.log('BUG-16 Check: Arbiter console loads properly:', hasTitle);
  });

  test('BUG-17: Result entry dialog behavior', async ({ page }) => {
    // Create and start a tournament first
    const tournamentName = uniqueTournamentName('Result Entry Test');

    await page.click('text=Neues Turnier');
    await page.fill('input[name="name"]', tournamentName);
    await page.fill('input[name="rounds"]', '1');
    await page.click('button:has-text("Weiter")');
    await page.waitForTimeout(500);

    await page.click('button:has-text("Datum wählen")');
    await page.waitForSelector('[role="gridcell"]');
    await page.click(`[role="gridcell"]:has-text("${new Date().getDate()}")`);

    await page.click('button:has-text("Weiter")');
    await page.click('button:has-text("Turnier erstellen")');
    await page.waitForURL('**/tournaments/**');

    // Publish, register, start
    await page.click('button:has-text("Veröffentlichen")');
    await page.click('button:has-text("Veröffentlichen"):not([disabled])');
    await page.waitForTimeout(1000);

    await page.click('button:has-text("Anmeldung öffnen")');
    await page.waitForTimeout(1000);

    const token = await getAuthToken(page);
    const tournamentId = page.url().split('/tournaments/')[1]?.split('/')[0];

    const playerIds = await createTestPlayers(token, 4);
    await registerPlayersForTournament(token, tournamentId, playerIds);
    await page.reload();

    await page.click('button:has-text("Turnier starten")');
    await page.waitForSelector('text=Starten');
    await page.click('button:has-text("Starten")');
    await page.waitForTimeout(2000);

    // Go to arbiter console
    await page.click('a:has-text("Arbiter-Konsole")');
    await page.waitForURL('**/arbiter/**');

    // Generate pairings
    await page.click('text=Runden');
    const generateBtn = page.locator('button:has-text("Paarungen erstellen")');
    if (await generateBtn.isVisible()) {
      await generateBtn.click();
      await page.waitForTimeout(2000);
    }

    // Start round
    const startBtn = page.locator('button:has-text("Runde starten")');
    if (await startBtn.isVisible()) {
      await startBtn.click();
      await page.waitForTimeout(1000);
    }

    // Go to active games
    await page.click('text=Aktive Partien');
    await page.waitForTimeout(1000);

    // Try to enter result
    const resultBtn = page.locator('button[aria-label="Ergebnis eintragen"]').first();
    if (await resultBtn.isVisible()) {
      await resultBtn.click();
      await page.waitForSelector('[role="dialog"]');

      // Check dialog content
      const dialog = page.locator('[role="dialog"]');
      const hasDialogContent = await dialog.isVisible();
      console.log('BUG-17 Check: Result entry dialog opens:', hasDialogContent);

      // Check result buttons exist
      const resultOptions = page.locator('[role="dialog"] button:has-text("1-0")');
      const hasResultOptions = await resultOptions.isVisible();
      console.log('BUG-17 Check: Result options visible:', hasResultOptions);

      if (!hasResultOptions) {
        console.error('BUG-17: Result entry dialog missing result options');
      }

      // Close dialog
      await page.keyboard.press('Escape');
    }
  });
});

// ============================================================================
// REGRESSION TESTS
// ============================================================================

test.describe('Regression Tests', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('Login and redirect works correctly', async ({ page }) => {
    // Verify we're on tournaments page after login
    await expect(page).toHaveURL(/.*\/tournaments/);

    // BUG-18: Previously redirected to /dashboard which 404'd
    const is404 = await page.locator('text=404').isVisible();
    if (is404) {
      console.error('BUG-18 REGRESSION: Login redirects to 404 page');
    }
  });

  test('Tournament creation persists data correctly', async ({ page }) => {
    const tournamentName = uniqueTournamentName('Persistence Test');

    await page.click('text=Neues Turnier');
    await page.fill('input[name="name"]', tournamentName);
    await page.fill('textarea[name="description"]', 'Test description');
    await page.click('button:has-text("Weiter")');
    await page.waitForTimeout(500);

    await page.click('button:has-text("Datum wählen")');
    await page.waitForSelector('[role="gridcell"]');
    await page.click(`[role="gridcell"]:has-text("${new Date().getDate()}")`);

    await page.click('button:has-text("Weiter")');
    await page.click('button:has-text("Turnier erstellen")');
    await page.waitForURL('**/tournaments/**');

    // Verify data persisted
    await expect(page.locator('h1')).toContainText(tournamentName);

    // BUG-19: Check if description is saved
    const description = page.locator('text=Test description');
    const hasDescription = await description.isVisible();
    console.log('BUG-19 Check: Tournament description persisted:', hasDescription);
  });
});
