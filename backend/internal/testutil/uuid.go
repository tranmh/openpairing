package testutil

import "github.com/google/uuid"

// Predefined UUIDs for consistent test data.
// Using deterministic UUIDs makes test output reproducible and easier to debug.
var (
	// Tournament UUIDs
	TournamentID1 = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	TournamentID2 = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	TournamentID3 = uuid.MustParse("00000000-0000-0000-0000-000000000003")

	// Player UUIDs
	PlayerID1 = uuid.MustParse("00000000-0000-0000-0001-000000000001")
	PlayerID2 = uuid.MustParse("00000000-0000-0000-0001-000000000002")
	PlayerID3 = uuid.MustParse("00000000-0000-0000-0001-000000000003")
	PlayerID4 = uuid.MustParse("00000000-0000-0000-0001-000000000004")
	PlayerID5 = uuid.MustParse("00000000-0000-0000-0001-000000000005")
	PlayerID6 = uuid.MustParse("00000000-0000-0000-0001-000000000006")
	PlayerID7 = uuid.MustParse("00000000-0000-0000-0001-000000000007")
	PlayerID8 = uuid.MustParse("00000000-0000-0000-0001-000000000008")

	// User UUIDs
	UserID1 = uuid.MustParse("00000000-0000-0000-0002-000000000001")
	UserID2 = uuid.MustParse("00000000-0000-0000-0002-000000000002")
	UserID3 = uuid.MustParse("00000000-0000-0000-0002-000000000003")

	// Organization UUIDs
	OrgID1 = uuid.MustParse("00000000-0000-0000-0003-000000000001")
	OrgID2 = uuid.MustParse("00000000-0000-0000-0003-000000000002")

	// Club UUIDs
	ClubID1 = uuid.MustParse("00000000-0000-0000-0004-000000000001")
	ClubID2 = uuid.MustParse("00000000-0000-0000-0004-000000000002")

	// Round UUIDs
	RoundID1 = uuid.MustParse("00000000-0000-0000-0005-000000000001")
	RoundID2 = uuid.MustParse("00000000-0000-0000-0005-000000000002")
	RoundID3 = uuid.MustParse("00000000-0000-0000-0005-000000000003")

	// Pairing UUIDs
	PairingID1 = uuid.MustParse("00000000-0000-0000-0006-000000000001")
	PairingID2 = uuid.MustParse("00000000-0000-0000-0006-000000000002")
	PairingID3 = uuid.MustParse("00000000-0000-0000-0006-000000000003")
	PairingID4 = uuid.MustParse("00000000-0000-0000-0006-000000000004")

	// Game UUIDs
	GameID1 = uuid.MustParse("00000000-0000-0000-0007-000000000001")
	GameID2 = uuid.MustParse("00000000-0000-0000-0007-000000000002")
	GameID3 = uuid.MustParse("00000000-0000-0000-0007-000000000003")
	GameID4 = uuid.MustParse("00000000-0000-0000-0007-000000000004")

	// Participant UUIDs (tournament participants)
	ParticipantID1 = uuid.MustParse("00000000-0000-0000-0008-000000000001")
	ParticipantID2 = uuid.MustParse("00000000-0000-0000-0008-000000000002")
	ParticipantID3 = uuid.MustParse("00000000-0000-0000-0008-000000000003")
	ParticipantID4 = uuid.MustParse("00000000-0000-0000-0008-000000000004")
	ParticipantID5 = uuid.MustParse("00000000-0000-0000-0008-000000000005")
	ParticipantID6 = uuid.MustParse("00000000-0000-0000-0008-000000000006")
	ParticipantID7 = uuid.MustParse("00000000-0000-0000-0008-000000000007")
	ParticipantID8 = uuid.MustParse("00000000-0000-0000-0008-000000000008")
)

// NewTestUUID generates a new random UUID for tests where uniqueness matters
// but the specific value doesn't.
func NewTestUUID() uuid.UUID {
	return uuid.New()
}

// MustParseUUID parses a UUID string and panics on error.
// Useful for inline UUID creation in test data.
func MustParseUUID(s string) uuid.UUID {
	return uuid.MustParse(s)
}
