package dsb

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestClient_ParsePlayerCSV(t *testing.T) {
	// Sample CSV response from DeWIS
	csvData := `pkz;vorname;nachname;dwz;index;verein;verband;fide_id;fide_elo;fide_titel;geburtsjahr;geschlecht;status
10012345;Magnus;Carlsen;2850;15;SK Test;GER;1503014;2830;GM;1990;M;A`

	client := NewClient(&ClientConfig{}, zerolog.Nop())

	player, err := client.parsePlayerCSV(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("parsePlayerCSV() error = %v", err)
	}

	if player.PKZ != "10012345" {
		t.Errorf("PKZ = %q, want %q", player.PKZ, "10012345")
	}
	if player.FirstName != "Magnus" {
		t.Errorf("FirstName = %q, want %q", player.FirstName, "Magnus")
	}
	if player.LastName != "Carlsen" {
		t.Errorf("LastName = %q, want %q", player.LastName, "Carlsen")
	}
	if player.DWZ != 2850 {
		t.Errorf("DWZ = %d, want %d", player.DWZ, 2850)
	}
	if player.DWZIndex != 15 {
		t.Errorf("DWZIndex = %d, want %d", player.DWZIndex, 15)
	}
	if player.FideID != "1503014" {
		t.Errorf("FideID = %q, want %q", player.FideID, "1503014")
	}
	if player.FideElo != 2830 {
		t.Errorf("FideElo = %d, want %d", player.FideElo, 2830)
	}
	if player.FideTitle != "GM" {
		t.Errorf("FideTitle = %q, want %q", player.FideTitle, "GM")
	}
	if player.BirthYear != 1990 {
		t.Errorf("BirthYear = %d, want %d", player.BirthYear, 1990)
	}
	if player.Gender != "M" {
		t.Errorf("Gender = %q, want %q", player.Gender, "M")
	}
}

func TestClient_ParsePlayersCSV(t *testing.T) {
	// Sample CSV response with multiple players
	csvData := `pkz;vorname;nachname;dwz;index;verein;verband
10012345;Magnus;Carlsen;2850;15;SK Test;GER
10012346;Vincent;Keymer;2760;12;SF Test;GER
10012347;Jan;Nepomnjaschtschi;2790;10;SK Test;RUS`

	client := NewClient(&ClientConfig{}, zerolog.Nop())

	players, err := client.parsePlayersCSV(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("parsePlayersCSV() error = %v", err)
	}

	if len(players) != 3 {
		t.Errorf("got %d players, want 3", len(players))
	}

	// Check first player
	if players[0].PKZ != "10012345" {
		t.Errorf("Player[0].PKZ = %q, want %q", players[0].PKZ, "10012345")
	}

	// Check second player
	if players[1].FirstName != "Vincent" {
		t.Errorf("Player[1].FirstName = %q, want %q", players[1].FirstName, "Vincent")
	}
}

func TestClient_GetPlayer(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/spieler.php" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		pkz := r.URL.Query().Get("pkz")
		if pkz != "10012345" {
			t.Errorf("unexpected pkz: %s", pkz)
		}

		format := r.URL.Query().Get("format")
		if format != "json" {
			t.Errorf("unexpected format: %s", format)
		}

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{
			"pkz": "10012345",
			"vorname": "Magnus",
			"nachname": "Carlsen",
			"dwz": 2850,
			"dwz_index": 15,
			"verein_id": "C1234",
			"verein_name": "SK Test",
			"verband": "GER",
			"fide_id": "1503014",
			"fide_elo": 2830,
			"fide_titel": "GM",
			"geburtsjahr": 1990,
			"geschlecht": "M",
			"status": "A"
		}`)
	}))
	defer server.Close()

	client := NewClient(&ClientConfig{
		BaseURL: server.URL + "/",
	}, zerolog.Nop())

	player, err := client.GetPlayer(context.Background(), "10012345")
	if err != nil {
		t.Fatalf("GetPlayer() error = %v", err)
	}

	if player.PKZ != "10012345" {
		t.Errorf("PKZ = %q, want %q", player.PKZ, "10012345")
	}
	if player.DWZ != 2850 {
		t.Errorf("DWZ = %d, want %d", player.DWZ, 2850)
	}
}

func TestClient_GetPlayer_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(&ClientConfig{
		BaseURL: server.URL + "/",
	}, zerolog.Nop())

	_, err := client.GetPlayer(context.Background(), "99999999")
	if err != ErrPlayerNotFound {
		t.Errorf("GetPlayer() error = %v, want ErrPlayerNotFound", err)
	}
}

func TestClient_SearchPlayers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/suche.php" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		query := r.URL.Query().Get("suche")
		if query != "Carlsen" {
			t.Errorf("unexpected query: %s", query)
		}

		w.Header().Set("Content-Type", "text/csv")
		io.WriteString(w, `pkz;vorname;nachname;dwz;index;verein;verband
10012345;Magnus;Carlsen;2850;15;SK Test;GER
10012346;Sigmund;Carlsen;2450;8;SF Test;GER`)
	}))
	defer server.Close()

	client := NewClient(&ClientConfig{
		BaseURL: server.URL + "/",
	}, zerolog.Nop())

	players, err := client.SearchPlayers(context.Background(), "Carlsen", 10)
	if err != nil {
		t.Fatalf("SearchPlayers() error = %v", err)
	}

	if len(players) != 2 {
		t.Errorf("got %d players, want 2", len(players))
	}
}

func TestConvertToPlayer(t *testing.T) {
	importer := &Importer{}

	dsbPlayer := &DSBPlayer{
		PKZ:       "10012345",
		FirstName: "Magnus",
		LastName:  "Carlsen",
		DWZ:       2850,
		DWZIndex:  15,
		FideID:    "1503014",
		FideElo:   2830,
		FideTitle: "GM",
		BirthYear: 1990,
		Gender:    "M",
	}

	player := importer.convertToPlayer(dsbPlayer)

	if player.FirstName != "Magnus" {
		t.Errorf("FirstName = %q, want %q", player.FirstName, "Magnus")
	}
	if player.LastName != "Carlsen" {
		t.Errorf("LastName = %q, want %q", player.LastName, "Carlsen")
	}
	if *player.DsbID != "10012345" {
		t.Errorf("DsbID = %q, want %q", *player.DsbID, "10012345")
	}
	if *player.DWZ != 2850 {
		t.Errorf("DWZ = %d, want %d", *player.DWZ, 2850)
	}
	if *player.FideID != "1503014" {
		t.Errorf("FideID = %q, want %q", *player.FideID, "1503014")
	}
	if *player.FideStandard != 2830 {
		t.Errorf("FideStandard = %d, want %d", *player.FideStandard, 2830)
	}
	if *player.FideTitle != "GM" {
		t.Errorf("FideTitle = %q, want %q", *player.FideTitle, "GM")
	}
	if *player.BirthDate != "1990-01-01" {
		t.Errorf("BirthDate = %q, want %q", *player.BirthDate, "1990-01-01")
	}
	if *player.Gender != "m" {
		t.Errorf("Gender = %q, want %q", *player.Gender, "m")
	}
}
