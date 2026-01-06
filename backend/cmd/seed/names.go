package main

import (
	"math/rand"
)

// DACH region name pools

var germanFirstNamesMale = []string{
	"Max", "Felix", "Paul", "Leon", "Lukas", "Jonas", "Tim", "Finn",
	"Niklas", "Jan", "David", "Philipp", "Moritz", "Julian", "Simon",
	"Alexander", "Michael", "Thomas", "Stefan", "Andreas", "Markus",
	"Christian", "Daniel", "Sebastian", "Martin", "Tobias", "Florian",
	"Wolfgang", "Klaus", "Hans", "Peter", "Jürgen", "Helmut", "Heinrich",
	"Matthias", "Patrick", "Benjamin", "Maximilian", "Fabian", "Kevin",
	"Dennis", "Marcel", "Dominik", "Marco", "Sven", "Frank", "Uwe",
	"Rainer", "Bernd", "Christoph", "Oliver", "Robert", "Lars", "Steffen",
	"Erik", "Nils", "Björn", "Thorsten", "Karsten", "Dirk", "Holger",
}

var germanFirstNamesFemale = []string{
	"Anna", "Marie", "Sophie", "Lena", "Laura", "Mia", "Emma", "Hannah",
	"Sophia", "Lisa", "Julia", "Sarah", "Katharina", "Christina",
	"Elisabeth", "Monika", "Petra", "Sabine", "Claudia", "Andrea",
	"Nicole", "Sandra", "Melanie", "Stefanie", "Daniela", "Anja",
	"Susanne", "Birgit", "Heike", "Martina", "Karin", "Gabriele",
	"Franziska", "Johanna", "Lea", "Emily", "Amelie", "Lara", "Leonie",
	"Nele", "Maja", "Charlotte", "Paula", "Emilia", "Clara", "Luisa",
}

var germanLastNames = []string{
	"Müller", "Schmidt", "Schneider", "Fischer", "Weber", "Meyer", "Wagner",
	"Becker", "Schulz", "Hoffmann", "Schäfer", "Koch", "Bauer", "Richter",
	"Klein", "Wolf", "Schröder", "Neumann", "Schwarz", "Zimmermann",
	"Braun", "Krüger", "Hofmann", "Hartmann", "Lange", "Werner", "Krause",
	"König", "Walter", "Mayer", "Peters", "Möller", "Keller", "Frank",
	"Berger", "Winkler", "Roth", "Beck", "Lorenz", "Baumann", "Franke",
	"Albrecht", "Schuster", "Simon", "Ludwig", "Böhm", "Winter", "Kraus",
	"Martin", "Schumacher", "Krämer", "Vogt", "Stein", "Jäger", "Otto",
	"Sommer", "Groß", "Seidel", "Heinrich", "Brandt", "Haas", "Schreiber",
	"Graf", "Schulze", "Dietrich", "Ziegler", "Kuhn", "Kühn", "Pohl",
}

var austrianLastNames = []string{
	"Gruber", "Huber", "Bauer", "Wagner", "Müller", "Pichler", "Steiner",
	"Moser", "Mayer", "Hofer", "Leitner", "Berger", "Fuchs", "Eder",
	"Fischer", "Schmid", "Winkler", "Weber", "Schwarz", "Maier",
	"Reiter", "Mayr", "Wimmer", "Egger", "Brunner", "Lang", "Baumgartner",
	"Auer", "Gabler", "Koller", "Hager", "Rainer", "Schuster", "Wallner",
}

var swissLastNames = []string{
	"Meier", "Müller", "Schmid", "Keller", "Weber", "Huber", "Schneider",
	"Meyer", "Steiner", "Fischer", "Brunner", "Gerber", "Baumann", "Frei",
	"Moser", "Widmer", "Roth", "Graf", "Wyss", "Bühler", "Sutter",
	"Studer", "Kaufmann", "Berger", "Baumgartner", "Hofer", "Kunz", "Wenger",
	"Lüthi", "Flückiger", "Zimmermann", "Hess", "Egli", "Zürcher", "Ammann",
}

// NameGenerator generates realistic DACH region names
type NameGenerator struct {
	rand *rand.Rand
}

// NewNameGenerator creates a new name generator with the given seed
func NewNameGenerator(seed int64) *NameGenerator {
	return &NameGenerator{
		rand: rand.New(rand.NewSource(seed)),
	}
}

// GenerateName generates a random name based on federation and gender
func (g *NameGenerator) GenerateName(federation string, gender string) (firstName, lastName string) {
	// Select first name based on gender
	if gender == "f" {
		firstName = germanFirstNamesFemale[g.rand.Intn(len(germanFirstNamesFemale))]
	} else {
		firstName = germanFirstNamesMale[g.rand.Intn(len(germanFirstNamesMale))]
	}

	// Select last name based on federation
	switch federation {
	case "AUT":
		lastName = austrianLastNames[g.rand.Intn(len(austrianLastNames))]
	case "SUI":
		lastName = swissLastNames[g.rand.Intn(len(swissLastNames))]
	default: // GER or default
		lastName = germanLastNames[g.rand.Intn(len(germanLastNames))]
	}

	return firstName, lastName
}

// GenerateFederation generates a random federation with weighted distribution
// 60% German, 25% Austrian, 15% Swiss
func (g *NameGenerator) GenerateFederation() string {
	r := g.rand.Float64()
	switch {
	case r < 0.60:
		return "GER"
	case r < 0.85:
		return "AUT"
	default:
		return "SUI"
	}
}

// GenerateGender generates a random gender with weighted distribution
// 85% male, 15% female (reflecting actual chess demographics)
func (g *NameGenerator) GenerateGender() string {
	if g.rand.Float64() < 0.15 {
		return "f"
	}
	return "m"
}

// GenerateBirthYear generates a random birth year for a chess player
// Distribution: mix of juniors, adults, and seniors
func (g *NameGenerator) GenerateBirthYear() int {
	r := g.rand.Float64()
	switch {
	case r < 0.15: // 15% juniors (2008-2016)
		return 2008 + g.rand.Intn(9)
	case r < 0.70: // 55% adults (1975-2007)
		return 1975 + g.rand.Intn(33)
	default: // 30% seniors (1945-1974)
		return 1945 + g.rand.Intn(30)
	}
}
