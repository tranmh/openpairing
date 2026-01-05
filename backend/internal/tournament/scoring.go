package tournament

import (
	"fmt"
)

// ResultType represents the type of game result
type ResultType int

const (
	ResultTypeWin ResultType = iota
	ResultTypeDraw
	ResultTypeLoss
)

// GetPointsForResult returns the display points for a given result type.
// This is the core function for converting result types to points.
func GetPointsForResult(resultType ResultType, scoring *ScoringSettings) float64 {
	if scoring == nil {
		scoring = DefaultScoringSettings()
	}

	switch resultType {
	case ResultTypeWin:
		return scoring.WinPoints
	case ResultTypeDraw:
		return scoring.DrawPoints
	case ResultTypeLoss:
		return scoring.LossPoints
	default:
		return 0
	}
}

// GetResultType determines the result type from a game result string.
func GetResultType(result string, forWhite bool) ResultType {
	switch result {
	case ResultWhiteWins, ResultWhiteForfeit:
		if forWhite {
			return ResultTypeWin
		}
		return ResultTypeLoss
	case ResultBlackWins, ResultBlackForfeit:
		if forWhite {
			return ResultTypeLoss
		}
		return ResultTypeWin
	case ResultDraw:
		return ResultTypeDraw
	case ResultDoubleForfeit:
		return ResultTypeLoss // Both players lose
	case ResultBye:
		if forWhite {
			return ResultTypeWin // Bye recipient gets win points
		}
		return ResultTypeLoss
	default:
		return ResultTypeLoss
	}
}

// ParseResultWithScoring converts a result string to scores using the tournament's scoring settings.
// Returns (whiteScore, blackScore) as display points.
func ParseResultWithScoring(result string, scoring *ScoringSettings) (whiteScore, blackScore float64) {
	if scoring == nil {
		scoring = DefaultScoringSettings()
	}

	switch result {
	case ResultWhiteWins, ResultWhiteForfeit:
		return scoring.WinPoints, scoring.LossPoints
	case ResultBlackWins, ResultBlackForfeit:
		return scoring.LossPoints, scoring.WinPoints
	case ResultDraw:
		return scoring.DrawPoints, scoring.DrawPoints
	case ResultDoubleForfeit:
		return scoring.LossPoints, scoring.LossPoints
	case ResultBye:
		return scoring.WinPoints, scoring.LossPoints // Bye recipient (white) gets win
	default:
		return scoring.LossPoints, scoring.LossPoints
	}
}

// IsWin checks if the given score represents a win based on scoring settings.
func IsWin(score float64, scoring *ScoringSettings) bool {
	if scoring == nil {
		scoring = DefaultScoringSettings()
	}
	return score == scoring.WinPoints
}

// IsDraw checks if the given score represents a draw based on scoring settings.
func IsDraw(score float64, scoring *ScoringSettings) bool {
	if scoring == nil {
		scoring = DefaultScoringSettings()
	}
	return score == scoring.DrawPoints
}

// IsLoss checks if the given score represents a loss based on scoring settings.
func IsLoss(score float64, scoring *ScoringSettings) bool {
	if scoring == nil {
		scoring = DefaultScoringSettings()
	}
	return score == scoring.LossPoints
}

// GetResultTypeFromScore determines the result type from a score value.
func GetResultTypeFromScore(score float64, scoring *ScoringSettings) ResultType {
	if scoring == nil {
		scoring = DefaultScoringSettings()
	}

	if score == scoring.WinPoints {
		return ResultTypeWin
	}
	if score == scoring.DrawPoints {
		return ResultTypeDraw
	}
	return ResultTypeLoss
}

// CalculateSonnebornBerger calculates the Sonneborn-Berger tiebreak value for a result.
// The mode determines how draws are weighted:
// - Traditional: draws contribute half of opponent's points
// - Proportional: draws contribute (drawPoints/winPoints) of opponent's points
func CalculateSonnebornBerger(score, opponentPoints float64, scoring *ScoringSettings) float64 {
	if scoring == nil {
		scoring = DefaultScoringSettings()
	}

	resultType := GetResultTypeFromScore(score, scoring)

	switch resultType {
	case ResultTypeWin:
		return opponentPoints
	case ResultTypeDraw:
		if scoring.SonnebornBergerMode == SBModeTraditional {
			// Traditional: always half for draws
			return opponentPoints / 2
		}
		// Proportional: based on point ratio
		return opponentPoints * (scoring.DrawPoints / scoring.WinPoints)
	case ResultTypeLoss:
		return 0
	default:
		return 0
	}
}

// FormatPoints formats points for display based on the scoring system.
// Standard scoring uses 1 decimal place, other systems use integers when possible.
func FormatPoints(points float64, scoring *ScoringSettings) string {
	if scoring == nil {
		scoring = DefaultScoringSettings()
	}

	if scoring.IsStandardScoring() {
		// Standard: always show 1 decimal (e.g., "3.5", "4.0")
		return fmt.Sprintf("%.1f", points)
	}

	// Non-standard: show as integer if whole number
	if points == float64(int(points)) {
		return fmt.Sprintf("%.0f", points)
	}
	return fmt.Sprintf("%.1f", points)
}

// GetByePoints returns the points for different bye types using bye settings.
func GetByePoints(byeType string, byeSettings *ByeSettings, scoring *ScoringSettings) float64 {
	if byeSettings == nil {
		byeSettings = DefaultByeSettings()
	}
	if scoring == nil {
		scoring = DefaultScoringSettings()
	}

	switch byeType {
	case "system":
		return byeSettings.SystemBye.Points
	case "half_point":
		return byeSettings.HalfPointBye.Points
	case "zero_point":
		return 0
	case "forfeit_win":
		return byeSettings.Forfeit.WinnerPoints
	case "forfeit_loss":
		return byeSettings.Forfeit.LoserPoints
	case "double_forfeit":
		return byeSettings.Forfeit.DoubleForfeitPoints
	default:
		return 0
	}
}
