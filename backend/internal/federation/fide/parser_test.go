package fide

import (
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestParseName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantFirst string
		wantLast  string
	}{
		{
			name:      "comma separated",
			input:     "CARLSEN, Magnus",
			wantFirst: "Magnus",
			wantLast:  "Carlsen",
		},
		{
			name:      "comma separated lowercase",
			input:     "carlsen, magnus",
			wantFirst: "Magnus",
			wantLast:  "Carlsen",
		},
		{
			name:      "multiple first names",
			input:     "KEYMER, Vincent",
			wantFirst: "Vincent",
			wantLast:  "Keymer",
		},
		{
			name:      "space separated uppercase last",
			input:     "MUELLER Thomas",
			wantFirst: "Thomas",
			wantLast:  "Mueller",
		},
		{
			name:      "German von",
			input:     "VON DER LASA, Tassilo",
			wantFirst: "Tassilo",
			wantLast:  "von der Lasa",
		},
		{
			name:      "Dutch van",
			input:     "VAN FOREEST, Jorden",
			wantFirst: "Jorden",
			wantLast:  "van Foreest",
		},
		{
			name:      "single name",
			input:     "CARLSEN",
			wantFirst: "",
			wantLast:  "Carlsen",
		},
		{
			name:      "extra whitespace",
			input:     "  CARLSEN,   Magnus  ",
			wantFirst: "Magnus",
			wantLast:  "Carlsen",
		},
		{
			name:      "hyphenated name",
			input:     "CRAMLING-BELLON, Pia",
			wantFirst: "Pia",
			wantLast:  "Cramling-bellon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseName(tt.input)
			if got.FirstName != tt.wantFirst {
				t.Errorf("ParseName(%q).FirstName = %q, want %q", tt.input, got.FirstName, tt.wantFirst)
			}
			if got.LastName != tt.wantLast {
				t.Errorf("ParseName(%q).LastName = %q, want %q", tt.input, got.LastName, tt.wantLast)
			}
		})
	}
}

func TestFIDEPlayer_GetTitle(t *testing.T) {
	tests := []struct {
		name   string
		player FIDEPlayer
		want   string
	}{
		{
			name:   "GM title",
			player: FIDEPlayer{Title: "GM"},
			want:   "GM",
		},
		{
			name:   "IM title",
			player: FIDEPlayer{Title: "IM"},
			want:   "IM",
		},
		{
			name:   "WGM title fallback",
			player: FIDEPlayer{WTitle: "WGM"},
			want:   "WGM",
		},
		{
			name:   "Title priority over WTitle",
			player: FIDEPlayer{Title: "IM", WTitle: "WGM"},
			want:   "IM",
		},
		{
			name:   "no title",
			player: FIDEPlayer{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.player.GetTitle(); got != tt.want {
				t.Errorf("GetTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFIDEPlayer_GetGender(t *testing.T) {
	tests := []struct {
		name   string
		sex    string
		want   string
	}{
		{"male uppercase", "M", "m"},
		{"male lowercase", "m", "m"},
		{"female uppercase", "F", "f"},
		{"female lowercase", "f", "f"},
		{"empty", "", ""},
		{"unknown", "X", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := FIDEPlayer{Sex: tt.sex}
			if got := p.GetGender(); got != tt.want {
				t.Errorf("GetGender() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFIDEPlayer_GetBirthDate(t *testing.T) {
	tests := []struct {
		name     string
		birthday string
		wantNil  bool
		wantDate string
	}{
		{
			name:     "full date",
			birthday: "1990-11-30",
			wantNil:  false,
			wantDate: "1990-11-30",
		},
		{
			name:     "year only",
			birthday: "1990",
			wantNil:  false,
			wantDate: "1990-01-01",
		},
		{
			name:    "empty",
			birthday: "",
			wantNil: true,
		},
		{
			name:    "invalid",
			birthday: "invalid",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := FIDEPlayer{Birthday: tt.birthday}
			got := p.GetBirthDate()

			if tt.wantNil {
				if got != nil {
					t.Errorf("GetBirthDate() = %q, want nil", *got)
				}
			} else {
				if got == nil {
					t.Error("GetBirthDate() = nil, want date")
				} else if *got != tt.wantDate {
					t.Errorf("GetBirthDate() = %q, want %q", *got, tt.wantDate)
				}
			}
		})
	}
}

func TestParseXML(t *testing.T) {
	// Sample FIDE XML format
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<playerslist>
	<player>
		<fideid>1503014</fideid>
		<name>Carlsen, Magnus</name>
		<country>NOR</country>
		<sex>M</sex>
		<title>GM</title>
		<rating>2830</rating>
		<games>10</games>
		<k>10</k>
		<rapid_rating>2820</rapid_rating>
		<blitz_rating>2885</blitz_rating>
		<birthday>1990</birthday>
	</player>
	<player>
		<fideid>12573981</fideid>
		<name>Keymer, Vincent</name>
		<country>GER</country>
		<sex>M</sex>
		<title>GM</title>
		<rating>2760</rating>
		<rapid_rating>2650</rapid_rating>
		<blitz_rating>2700</blitz_rating>
		<birthday>2004</birthday>
	</player>
</playerslist>`

	parser := NewParser(zerolog.Nop())

	var players []*FIDEPlayer
	handler := func(p *FIDEPlayer) error {
		players = append(players, p)
		return nil
	}

	result, err := parser.ParseXML(strings.NewReader(xmlData), handler)
	if err != nil {
		t.Fatalf("ParseXML() error = %v", err)
	}

	if result.TotalPlayers != 2 {
		t.Errorf("TotalPlayers = %d, want 2", result.TotalPlayers)
	}

	if result.ParsedPlayers != 2 {
		t.Errorf("ParsedPlayers = %d, want 2", result.ParsedPlayers)
	}

	if len(players) != 2 {
		t.Fatalf("got %d players, want 2", len(players))
	}

	// Check first player
	if players[0].FideID != "1503014" {
		t.Errorf("Player[0].FideID = %q, want %q", players[0].FideID, "1503014")
	}
	if players[0].Rating != 2830 {
		t.Errorf("Player[0].Rating = %d, want %d", players[0].Rating, 2830)
	}
	if players[0].Country != "NOR" {
		t.Errorf("Player[0].Country = %q, want %q", players[0].Country, "NOR")
	}

	// Check second player (German)
	if players[1].FideID != "12573981" {
		t.Errorf("Player[1].FideID = %q, want %q", players[1].FideID, "12573981")
	}
	if players[1].Country != "GER" {
		t.Errorf("Player[1].Country = %q, want %q", players[1].Country, "GER")
	}
}

func TestFilterDACH(t *testing.T) {
	var dachPlayers []*FIDEPlayer
	handler := FilterDACH(func(p *FIDEPlayer) error {
		dachPlayers = append(dachPlayers, p)
		return nil
	})

	testPlayers := []FIDEPlayer{
		{FideID: "1", Country: "GER"},
		{FideID: "2", Country: "AUT"},
		{FideID: "3", Country: "SUI"},
		{FideID: "4", Country: "NOR"},
		{FideID: "5", Country: "USA"},
	}

	for _, p := range testPlayers {
		_ = handler(&p)
	}

	if len(dachPlayers) != 3 {
		t.Errorf("got %d DACH players, want 3", len(dachPlayers))
	}

	// Verify only DACH countries
	for _, p := range dachPlayers {
		if p.Country != "GER" && p.Country != "AUT" && p.Country != "SUI" {
			t.Errorf("unexpected country %q in DACH filter", p.Country)
		}
	}
}

func TestFilterMinRating(t *testing.T) {
	var filteredPlayers []*FIDEPlayer
	handler := FilterMinRating(2000, func(p *FIDEPlayer) error {
		filteredPlayers = append(filteredPlayers, p)
		return nil
	})

	testPlayers := []FIDEPlayer{
		{FideID: "1", Rating: 2500},
		{FideID: "2", Rating: 1800},
		{FideID: "3", Rating: 0, RapidRating: 2100},
		{FideID: "4", Rating: 0, BlitzRating: 1900},
		{FideID: "5", Rating: 2000},
	}

	for _, p := range testPlayers {
		_ = handler(&p)
	}

	// Should include: 2500, 2100 (rapid), 2000
	if len(filteredPlayers) != 3 {
		t.Errorf("got %d filtered players, want 3", len(filteredPlayers))
	}
}

func TestCapitalizeWords(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"CARLSEN", "Carlsen"},
		{"magnus carlsen", "Magnus Carlsen"},
		{"VON DER LASA", "von der Lasa"},
		{"VAN FOREEST", "van Foreest"},
		{"DE GROOT", "de Groot"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := capitalizeWords(tt.input); got != tt.want {
				t.Errorf("capitalizeWords(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
