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
    await page.waitForTimeout(1000);

    // Step 4: Settings (Step 3) - Submit form
    await page.click('button[type="submit"]:has-text("Turnier erstellen")');

    // Wait for redirect to tournament detail page
    await page.waitForURL('**/tournaments/**', { timeout: 10000 });

    // Verify tournament created
    await expect(page.locator('h1')).toContainText(tournamentName);

    // Step 5: Publish tournament
    await page.click('button:has-text("Veröffentlichen")');
    await page.click('button:has-text("Veröffentlichen"):not([disabled])');
    await page.waitForTimeout(1000);

    // Step 6: Open registration
    await page.click('button:has-text("Anmeldung öffnen")');
    await page.waitForTimeout(1000);

    // Get token and tournament ID for API calls
    const token = await getAuthToken(page);
    const tournamentId = page.url().split('/tournaments/')[1]?.split('/')[0];

    // BUG-01: Verify tournament ID is extracted correctly
    expect(tournamentId).toBeTruthy();
    console.log('Tournament ID:', tournamentId);

    // Step 7: Create and register test players via API
    const playerIds = await createTestPlayers(token, 6);
    console.log('Created players:', playerIds.length);

    // BUG-02: Check if players were created
    if (playerIds.length < 4) {
      console.error('BUG-02: Not enough players created. API may have issues.');
    }

    await registerPlayersForTournament(token, tournamentId, playerIds);

    // Refresh page to see participants
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Step 8: Start tournament
    await page.click('button:has-text("Turnier starten")');
    await page.waitForSelector('text=Starten');
    await page.click('button:has-text("Starten")');
    await page.waitForTimeout(2000);

    // Step 9: Go to Arbiter Console
    await page.click('a:has-text("Arbiter-Konsole")');
    await page.waitForURL('**/arbiter/**');
    await page.waitForLoadState('networkidle');

    // Step 10: Generate pairings for round 1
    await page.click('text=Runden');
    await page.waitForTimeout(500);

    // Click generate pairings button
    const generateBtn = page.locator('button:has-text("Paarungen erstellen")');
    if (await generateBtn.isVisible()) {
      await generateBtn.click();
      await page.waitForTimeout(2000);
    }

    // Step 11: Start round
    const startRoundBtn = page.locator('button:has-text("Runde starten")');
    if (await startRoundBtn.isVisible()) {
      await startRoundBtn.click();
      await page.waitForTimeout(1000);
    }

    // Step 12: Enter results via Active Games tab
    await page.click('text=Aktive Partien');
    await page.waitForTimeout(500);

    // Find result entry buttons and enter results
    const resultButtons = page.locator('button[aria-label="Ergebnis eintragen"]');
    const buttonCount = await resultButtons.count();
    console.log('Active games to enter results:', buttonCount);

    for (let i = 0; i < buttonCount && i < 3; i++) {
      const btn = resultButtons.nth(i);
      if (await btn.isVisible()) {
        await btn.click();
        await page.waitForSelector('[role="dialog"]');

        // Click 1-0 result
        await page.click('button:has-text("1-0")');
        await page.click('button:has-text("Speichern")');
        await page.waitForTimeout(1000);
      }
    }

    // Step 13: Complete round
    await page.click('text=Runden');
    await page.waitForTimeout(500);

    const completeBtn = page.locator('button:has-text("Runde beenden")');
    if (await completeBtn.isVisible() && await completeBtn.isEnabled()) {
      await completeBtn.click();
      await page.waitForTimeout(1000);
    }

    // Step 14: Check standings
    await page.goto(`/tournaments/${tournamentId}`);
    await page.waitForLoadState('networkidle');
    await page.click('text=Tabelle');
    await page.waitForTimeout(500);

    // Verify standings table exists
    const standingsTable = page.locator('table');
    await expect(standingsTable).toBeVisible();
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
    await page.waitForTimeout(500);

    // Step 4: Submit
    await page.click('button:has-text("Turnier erstellen")');
    await page.waitForURL('**/tournaments/**', { timeout: 10000 });

    // Verify tournament created
    await expect(page.locator('h1')).toContainText(tournamentName);

    // Step 5: Publish
    await page.click('button:has-text("Veröffentlichen")');
    await page.click('button:has-text("Veröffentlichen"):not([disabled])');
    await page.waitForTimeout(1000);

    // Step 6: Open registration
    await page.click('button:has-text("Anmeldung öffnen")');
    await page.waitForTimeout(1000);

    const token = await getAuthToken(page);
    const tournamentId = page.url().split('/tournaments/')[1]?.split('/')[0];

    // Step 7: Register 4 players for round robin
    const playerIds = await createTestPlayers(token, 4);
    await registerPlayersForTournament(token, tournamentId, playerIds);
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Step 8: Check seating tab visibility (Round Robin specific)
    const seatingTab = page.locator('button:has-text("Sitzplan")');
    const isSeatingVisible = await seatingTab.isVisible();
    console.log('BUG-04 Check: Seating tab visible for round robin:', isSeatingVisible);

    // BUG-04: Seating tab should be visible for round robin
    if (!isSeatingVisible) {
      console.error('BUG-04: Seating tab not visible for Round Robin tournament');
    }

    // Step 9: Start tournament
    await page.click('button:has-text("Turnier starten")');
    await page.waitForSelector('text=Starten');
    await page.click('button:has-text("Starten")');
    await page.waitForTimeout(2000);

    // Step 10: Go to Arbiter Console
    await page.click('a:has-text("Arbiter-Konsole")');
    await page.waitForURL('**/arbiter/**');

    // Step 11: Generate pairings
    await page.click('text=Runden');
    await page.waitForTimeout(500);

    const generateBtn = page.locator('button:has-text("Paarungen erstellen")');
    if (await generateBtn.isVisible()) {
      await generateBtn.click();
      await page.waitForTimeout(2000);
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
    // This test checks the seating configuration for round robin
    const tournamentName = uniqueTournamentName('RR Seating Test');

    await page.click('text=Neues Turnier');
    await page.waitForURL('**/tournaments/new');

    await page.fill('input[name="name"]', tournamentName);
    await page.click('button:has-text("Schweizer System")');
    await page.click('[role="option"]:has-text("Rundenturnier")');
    await page.fill('input[name="rounds"]', '5');

    await page.click('button:has-text("Weiter")');
    await page.waitForTimeout(500);

    await page.click('button:has-text("Datum wählen")');
    await page.waitForSelector('[role="gridcell"]');
    const today = new Date().getDate().toString();
    await page.click(`[role="gridcell"]:has-text("${today}")`);

    await page.click('button:has-text("Weiter")');
    await page.waitForTimeout(500);

    await page.click('button:has-text("Turnier erstellen")');
    await page.waitForURL('**/tournaments/**');

    // Publish and register
    await page.click('button:has-text("Veröffentlichen")');
    await page.click('button:has-text("Veröffentlichen"):not([disabled])');
    await page.waitForTimeout(1000);

    await page.click('button:has-text("Anmeldung öffnen")');
    await page.waitForTimeout(1000);

    const token = await getAuthToken(page);
    const tournamentId = page.url().split('/tournaments/')[1]?.split('/')[0];

    const playerIds = await createTestPlayers(token, 6);
    await registerPlayersForTournament(token, tournamentId, playerIds);
    await page.reload();

    // Check seating tab
    const seatingTab = page.locator('button:has-text("Sitzplan")');
    if (await seatingTab.isVisible()) {
      await seatingTab.click();
      await page.waitForTimeout(1000);

      // Check for seating configuration options
      const settingsCard = page.locator('text=Sitzplan-Einstellungen');
      const hasSettings = await settingsCard.isVisible();
      console.log('BUG-06 Check: Seating settings visible:', hasSettings);

      // BUG-06: Seating settings should be visible
      if (!hasSettings) {
        console.error('BUG-06: Seating settings not visible for Round Robin');
      }
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

    // BUG-07: Check if WAM-specific settings appear
    await page.waitForTimeout(500);

    await page.click('button:has-text("Weiter")');
    await page.waitForTimeout(500);

    // Fill date
    await page.click('button:has-text("Datum wählen")');
    await page.waitForSelector('[role="gridcell"]');
    const today = new Date().getDate().toString();
    await page.click(`[role="gridcell"]:has-text("${today}")`);

    await page.click('button:has-text("Weiter")');
    await page.waitForTimeout(500);

    // Step 2: Check WAM settings on step 3
    const wamSettings = page.locator('text=WAM-Einstellungen');
    const hasWamSettings = await wamSettings.isVisible();
    console.log('BUG-07 Check: WAM settings visible on step 3:', hasWamSettings);

    if (hasWamSettings) {
      // Configure WAM settings
      // Group size
      const groupSizeSelect = page.locator('text=Gruppengröße').locator('..').locator('button');
      if (await groupSizeSelect.isVisible()) {
        await groupSizeSelect.click();
        await page.click('text=4 Spieler');
      }

      // Enable playoffs
      const playoffsCheckbox = page.locator('text=Playoffs aktivieren').locator('..').locator('button[role="checkbox"]');
      if (await playoffsCheckbox.isVisible()) {
        await playoffsCheckbox.click();
      }
    } else {
      console.error('BUG-07: WAM settings not visible when WAM type selected');
    }

    // Submit
    await page.click('button:has-text("Turnier erstellen")');
    await page.waitForURL('**/tournaments/**', { timeout: 10000 });

    // Verify
    await expect(page.locator('h1')).toContainText(tournamentName);

    // Step 3: Publish and register
    await page.click('button:has-text("Veröffentlichen")');
    await page.click('button:has-text("Veröffentlichen"):not([disabled])');
    await page.waitForTimeout(1000);

    await page.click('button:has-text("Anmeldung öffnen")');
    await page.waitForTimeout(1000);

    const token = await getAuthToken(page);
    const tournamentId = page.url().split('/tournaments/')[1]?.split('/')[0];

    // Step 4: Register 8 players for WAM (2 groups of 4)
    const playerIds = await createTestPlayers(token, 8);
    await registerPlayersForTournament(token, tournamentId, playerIds);
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Step 5: Check Groups tab (WAM specific)
    const groupsTab = page.locator('button:has-text("Gruppen")');
    const hasGroupsTab = await groupsTab.isVisible();
    console.log('BUG-08 Check: Groups tab visible for WAM:', hasGroupsTab);

    if (hasGroupsTab) {
      await groupsTab.click();
      await page.waitForTimeout(1000);

      // Check for group assignment UI
      const groupContent = page.locator('text=Gruppe');
      const hasGroups = await groupContent.isVisible();
      console.log('Groups displayed:', hasGroups);
    } else {
      console.error('BUG-08: Groups tab not visible for WAM tournament');
    }

    // Step 6: Start tournament
    await page.click('button:has-text("Turnier starten")');
    await page.waitForSelector('text=Starten');
    await page.click('button:has-text("Starten")');
    await page.waitForTimeout(2000);

    // Step 7: Go to Arbiter Console
    await page.click('a:has-text("Arbiter-Konsole")');
    await page.waitForURL('**/arbiter/**');

    // Step 8: Check WAM-specific functionality in arbiter console
    await page.click('text=Runden');
    await page.waitForTimeout(500);

    const generateBtn = page.locator('button:has-text("Paarungen erstellen")');
    if (await generateBtn.isVisible()) {
      await generateBtn.click();
      await page.waitForTimeout(2000);

      // BUG-09: Check if group-based pairings work
      const errorToast = page.locator('text=Fehler');
      const hasError = await errorToast.isVisible();
      if (hasError) {
        console.error('BUG-09: Error generating WAM pairings');
      }
    }
  });

  test('WAM group assignment verification', async ({ page }) => {
    const tournamentName = uniqueTournamentName('WAM Groups Test');

    // Create WAM tournament
    await page.click('text=Neues Turnier');
    await page.waitForURL('**/tournaments/new');

    await page.fill('input[name="name"]', tournamentName);
    await page.click('button:has-text("Schweizer System")');
    await page.click('[role="option"]:has-text("WAM")');

    await page.click('button:has-text("Weiter")');
    await page.waitForTimeout(500);

    await page.click('button:has-text("Datum wählen")');
    await page.waitForSelector('[role="gridcell"]');
    await page.click(`[role="gridcell"]:has-text("${new Date().getDate()}")`);

    await page.click('button:has-text("Weiter")');
    await page.click('button:has-text("Turnier erstellen")');
    await page.waitForURL('**/tournaments/**');

    // Publish and register
    await page.click('button:has-text("Veröffentlichen")');
    await page.click('button:has-text("Veröffentlichen"):not([disabled])');
    await page.waitForTimeout(1000);

    await page.click('button:has-text("Anmeldung öffnen")');
    await page.waitForTimeout(1000);

    const token = await getAuthToken(page);
    const tournamentId = page.url().split('/tournaments/')[1]?.split('/')[0];

    // Register 12 players
    const playerIds = await createTestPlayers(token, 12);
    await registerPlayersForTournament(token, tournamentId, playerIds);
    await page.reload();

    // Check groups tab
    const groupsTab = page.locator('button:has-text("Gruppen")');
    if (await groupsTab.isVisible()) {
      await groupsTab.click();
      await page.waitForTimeout(1000);

      // BUG-10: Check automatic group assignment
      const groups = page.locator('text=Gruppe A');
      const hasGroupA = await groups.isVisible();
      console.log('BUG-10 Check: Automatic group assignment (Gruppe A visible):', hasGroupA);

      if (!hasGroupA) {
        console.error('BUG-10: Groups not automatically assigned for WAM');
      }
    }
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
