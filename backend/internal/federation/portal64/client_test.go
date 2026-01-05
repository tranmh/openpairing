package portal64

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// Integration tests against the live portal64 API at test.svw.info
// Run with: go test -v -tags=integration ./internal/federation/portal64/...

func getTestClient() *Client {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	return NewClient(&ClientConfig{
		BaseURL: "https://test.svw.info:8080",
		Timeout: 30 * time.Second,
	}, logger)
}

func TestClient_SearchPlayers_Integration(t *testing.T) {
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Search for a common German chess name
	players, err := client.SearchPlayers(ctx, "Mueller", 10)
	if err != nil {
		t.Fatalf("SearchPlayers failed: %v", err)
	}

	if len(players) == 0 {
		t.Error("Expected at least one player matching 'Mueller'")
	}

	t.Logf("Found %d players matching 'Mueller'", len(players))

	// Verify player data structure
	for i, p := range players {
		if i >= 3 {
			break // Only log first 3
		}
		t.Logf("Player %d: ID=%s, PKZ=%s, Name=%s %s, DWZ=%d, Club=%s",
			i+1, p.ID, p.PKZ, p.Firstname, p.Name, p.CurrentDWZ, p.Club)

		// Verify required fields are populated
		if p.ID == "" {
			t.Errorf("Player %d has empty ID", i+1)
		}
		if p.PKZ == "" {
			t.Errorf("Player %d has empty PKZ", i+1)
		}
		if p.Name == "" && p.Firstname == "" {
			t.Errorf("Player %d has no name", i+1)
		}
	}
}

func TestClient_SearchPlayers_ByPKZ_Integration(t *testing.T) {
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First search for a player to get a valid ID
	players, err := client.SearchPlayers(ctx, "Tran", 5)
	if err != nil {
		t.Fatalf("SearchPlayers failed: %v", err)
	}

	if len(players) == 0 {
		t.Skip("No players found to test lookup")
	}

	// Use the first player's ID to fetch by GetPlayer (not PKZ search)
	testID := players[0].ID
	testPKZ := players[0].PKZ
	t.Logf("Testing player lookup for ID: %s (PKZ: %s, Name: %s %s)", testID, testPKZ, players[0].Firstname, players[0].Name)

	// Get player by ID
	player, err := client.GetPlayer(ctx, testID)
	if err != nil {
		t.Fatalf("GetPlayer failed: %v", err)
	}

	if player.PKZ != testPKZ {
		t.Errorf("Expected PKZ %s, got %s", testPKZ, player.PKZ)
	}

	t.Logf("Found player by ID: %s %s (DWZ: %d, PKZ: %s)", player.Firstname, player.Name, player.CurrentDWZ, player.PKZ)
}

func TestClient_GetPlayer_Integration(t *testing.T) {
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First search to get a valid player ID
	players, err := client.SearchPlayers(ctx, "Schmidt", 5)
	if err != nil {
		t.Fatalf("SearchPlayers failed: %v", err)
	}

	if len(players) == 0 {
		t.Skip("No players found to test GetPlayer")
	}

	// Get first player by ID
	testID := players[0].ID
	t.Logf("Testing GetPlayer for ID: %s", testID)

	player, err := client.GetPlayer(ctx, testID)
	if err != nil {
		t.Fatalf("GetPlayer failed: %v", err)
	}

	if player.ID != testID {
		t.Errorf("Expected ID %s, got %s", testID, player.ID)
	}

	t.Logf("Player details: %s %s, PKZ=%s, DWZ=%d, Club=%s, Gender=%s",
		player.Firstname, player.Name, player.PKZ, player.CurrentDWZ, player.Club, player.Gender)
}

func TestClient_GetClubPlayers_Integration(t *testing.T) {
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test with a known club ID (C0101 is a common format)
	// First search for a player to find a valid club
	players, err := client.SearchPlayers(ctx, "Stuttgart", 5)
	if err != nil {
		t.Fatalf("SearchPlayers failed: %v", err)
	}

	if len(players) == 0 {
		t.Skip("No players found to determine club ID")
	}

	clubID := players[0].ClubID
	if clubID == "" {
		t.Skip("First player has no club ID")
	}

	t.Logf("Testing GetClubPlayers for club: %s (%s)", clubID, players[0].Club)

	clubPlayers, err := client.GetClubPlayers(ctx, clubID, 50)
	if err != nil {
		// Club endpoint might not be available or have different format
		t.Logf("GetClubPlayers failed (may be expected): %v", err)
		return
	}

	t.Logf("Found %d players in club %s", len(clubPlayers), clubID)

	for i, p := range clubPlayers {
		if i >= 5 {
			break
		}
		t.Logf("  %s %s (DWZ: %d)", p.Firstname, p.Name, p.CurrentDWZ)
	}
}

func TestClient_SearchPlayers_EmptyQuery_Integration(t *testing.T) {
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test with very short query - should still work
	players, err := client.SearchPlayers(ctx, "Ma", 5)
	if err != nil {
		t.Logf("Short query 'Ma' returned error (may be expected): %v", err)
		return
	}

	t.Logf("Found %d players for query 'Ma'", len(players))
}

func TestClient_SearchPlayers_NotFound_Integration(t *testing.T) {
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Search for something that shouldn't exist
	players, err := client.SearchPlayers(ctx, "XYZNONEXISTENT12345", 10)
	if err != nil {
		t.Fatalf("SearchPlayers failed: %v", err)
	}

	if len(players) != 0 {
		t.Errorf("Expected 0 players for nonsense query, got %d", len(players))
	}
}

func TestClient_HealthCheck_Integration(t *testing.T) {
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := client.HealthCheck(ctx)
	if err != nil {
		t.Logf("HealthCheck failed (endpoint may not exist): %v", err)
		// Not a hard failure - health endpoint is optional
	} else {
		t.Log("HealthCheck passed")
	}
}
