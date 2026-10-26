package reports

import (
	"fmt"
	"testing"
)

var (
	benchIntSink   int
	benchNormSink  *NormResult
	benchBytesSink []byte
	benchErrsSink  []string
)

func benchNormGames(n int) []NormGameDetail {
	feds := []string{"GER", "FRA", "ESP", "NED", "AUT", "SUI"}
	titles := []string{"GM", "IM", "FM", ""}
	results := []float64{1, 0.5, 0, 1, 0.5}
	games := make([]NormGameDetail, n)
	for i := range games {
		games[i] = NormGameDetail{
			Round:          i + 1,
			OpponentName:   fmt.Sprintf("Opponent %d", i),
			OpponentFideID: fmt.Sprintf("%d", 1000000+i),
			OpponentRating: 2300 + (i*37)%300,
			OpponentTitle:  titles[i%len(titles)],
			OpponentFed:    feds[i%len(feds)],
			Color:          []string{"w", "b"}[i%2],
			Result:         results[i%len(results)],
		}
	}
	return games
}

func BenchmarkNormCalculator_CalculatePerformance(b *testing.B) {
	c := NewNormCalculator()
	for _, n := range []int{9, 100, 1000} {
		games := benchNormGames(n)
		b.Run(fmt.Sprintf("games=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchIntSink = c.CalculatePerformance(games)
			}
		})
	}
}

func BenchmarkNormCalculator_CheckNorm(b *testing.B) {
	c := NewNormCalculator()
	for _, n := range []int{9, 100, 1000} {
		games := benchNormGames(n)
		b.Run(fmt.Sprintf("games=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchNormSink = c.CheckNorm(NormGM, 2450, "GER", games)
			}
		})
	}
}

func benchDWZGenerator(players, rounds int) *DWZGenerator {
	g := NewDWZGenerator()
	g.SetTournamentInfo(DWZTournamentInfo{
		TurnierCode:   "B123-456-789",
		Name:          `Open "Stadtmeisterschaft"; <Berlin> & Co`,
		City:          "Berlin",
		Organizer:     "SC Test",
		ChiefArbiter:  "Arbiter",
		TimeControl:   "90+30",
		Rounds:        rounds,
		PairingSystem: "Swiss",
		RatingType:    "Standard",
	})
	resultCodes := []string{"1", "0", "=", "+", "-"}
	gameResults := []string{"1-0", "0-1", "1/2-1/2", "+/-", "-/+"}
	for p := 1; p <= players; p++ {
		rr := make([]DWZRoundResult, rounds)
		for r := range rr {
			rr[r] = DWZRoundResult{
				Round:       r + 1,
				OpponentNum: (p+r)%players + 1,
				Color:       []string{"w", "b"}[(p+r)%2],
				Result:      resultCodes[(p+r)%len(resultCodes)],
			}
		}
		g.AddPlayer(DWZPlayerRecord{
			StartNumber: p,
			PKZ:         fmt.Sprintf("%08d", 10000000+p),
			Name:        fmt.Sprintf("Müller, Hans %d", p),
			FirstName:   "Hans",
			LastName:    fmt.Sprintf("Müller;%d", p),
			ClubID:      fmt.Sprintf("C%04d", p%50),
			ClubName:    `SK "Turm" & Co`,
			DWZ:         1200 + (p*13)%1200,
			DWZIndex:    p % 40,
			FideID:      fmt.Sprintf("%d", 2000000+p),
			FideRating:  1300 + (p*17)%1100,
			BirthYear:   1950 + p%60,
			Gender:      []string{"M", "F"}[p%2],
			Score:       float64(p%(rounds+1)) / 2,
			Rank:        p,
			Rounds:      rr,
		})
	}
	for r := 1; r <= rounds; r++ {
		for p := 1; p+1 <= players; p += 2 {
			g.AddGame(DWZGameRecord{
				Round:     r,
				WhitePKZ:  fmt.Sprintf("%08d", 10000000+p),
				WhiteName: fmt.Sprintf("Player %d", p),
				WhiteDWZ:  1500,
				BlackPKZ:  fmt.Sprintf("%08d", 10000000+p+1),
				BlackName: fmt.Sprintf("Player %d", p+1),
				BlackDWZ:  1600,
				Result:    gameResults[(p+r)%len(gameResults)],
			})
		}
	}
	return g
}

var benchDWZSizes = []struct{ players, rounds int }{{20, 5}, {200, 9}, {1000, 11}}

func BenchmarkDWZGenerator_Validate(b *testing.B) {
	for _, s := range benchDWZSizes {
		g := benchDWZGenerator(s.players, s.rounds)
		b.Run(fmt.Sprintf("players=%d", s.players), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchErrsSink = g.Validate()
			}
		})
	}
}

func BenchmarkDWZGenerator_GenerateCSV(b *testing.B) {
	for _, s := range benchDWZSizes {
		g := benchDWZGenerator(s.players, s.rounds)
		b.Run(fmt.Sprintf("players=%d", s.players), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				out, err := g.GenerateCSV()
				if err != nil {
					b.Fatal(err)
				}
				benchBytesSink = out
			}
		})
	}
}

func BenchmarkDWZGenerator_GenerateXML(b *testing.B) {
	for _, s := range benchDWZSizes {
		g := benchDWZGenerator(s.players, s.rounds)
		b.Run(fmt.Sprintf("players=%d", s.players), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				out, err := g.GenerateXML()
				if err != nil {
					b.Fatal(err)
				}
				benchBytesSink = out
			}
		})
	}
}
