/**
 * Component prop types for Arbiter Console components
 */

import type {
  TournamentStatus,
  ParticipantInfo,
  RoundInfo,
  CurrentRound,
  ActiveGame,
  PairingGame,
} from '@/types/arbiter';

// Re-export types for convenience
export type { TournamentStatus, ParticipantInfo, RoundInfo, CurrentRound, PairingGame };

/**
 * Common props shared by many arbiter components
 */
export interface ArbiterComponentProps {
  tournamentId: string;
}

/**
 * Status Cards props
 */
export interface StatusCardsProps {
  currentRound?: CurrentRound;
  totalRounds: number;
  statistics: TournamentStatus['statistics'];
  activeGamesCount: number;
  pendingResults: number;
}

/**
 * Tab component props
 */
export interface ActiveGamesTabProps extends ArbiterComponentProps {
  games: ActiveGame[];
  roundStatus: string;
}

export interface ParticipantsTabProps extends ArbiterComponentProps {
  participants: ParticipantInfo[];
}

export interface RoundControlTabProps extends ArbiterComponentProps {
  rounds: RoundInfo[];
  currentRound?: CurrentRound;
  pendingResults: number;
}

export interface PairingsManagementTabProps extends ArbiterComponentProps {
  rounds: RoundInfo[];
  participants: ParticipantInfo[];
}

export interface ActionsTabProps extends ArbiterComponentProps {
  currentRound?: CurrentRound;
  participants: ParticipantInfo[];
  tournamentRounds: number;
}

export interface SettingsTabProps extends ArbiterComponentProps {
  tournamentName: string;
  participants: ParticipantInfo[];
}

/**
 * Dialog component props
 */
export interface ResultEntryDialogProps extends ArbiterComponentProps {
  gameId: string;
  boardNumber: number;
  whiteName: string;
  blackName: string;
}

export interface ChangeColorDialogProps extends ArbiterComponentProps {
  gameId: string;
  whiteName: string;
  blackName: string;
}

export interface DeletePairingDialogProps extends ArbiterComponentProps {
  gameId: string;
  boardNumber: number;
}

export interface SwapPlayersDialogProps extends ArbiterComponentProps {
  roundNumber: number;
  pairings: PairingGame[];
}

export interface CreateManualPairingDialogProps extends ArbiterComponentProps {
  roundNumber: number;
  participants: ParticipantInfo[];
  existingPairings: PairingGame[];
}

export interface CreateAnnouncementDialogProps extends ArbiterComponentProps {}

export interface ReactivatePlayerDialogProps extends ArbiterComponentProps {
  participantId: string;
  playerName: string;
}

export interface LateEntryDialogProps extends ArbiterComponentProps {
  currentRoundNumber: number;
  tournamentRounds: number;
}

export interface CloneTournamentDialogProps extends ArbiterComponentProps {
  tournamentName: string;
}

export interface RatingOverrideDialogProps extends ArbiterComponentProps {
  participantId: string;
  playerName: string;
  currentRating: number;
  hasOverride?: boolean;
}

/**
 * Form component props
 */
export interface AssignByeFormProps extends ArbiterComponentProps {
  roundNumber: number;
  participants: ParticipantInfo[];
}

export interface CorrectResultFormProps extends ArbiterComponentProps {
  /** Optional: pre-populated game for correction */
  selectedGame?: PairingGame;
  /** All games in the tournament for selection */
  games?: PairingGame[];
}

/**
 * Card component props
 */
export interface ForbiddenPairsCardProps extends ArbiterComponentProps {
  participants: ParticipantInfo[];
}

export interface SpecialRankingsCardProps extends ArbiterComponentProps {}

/**
 * Hook return types
 */
export interface UseArbiterStatusReturn {
  status: TournamentStatus | undefined;
  isLoading: boolean;
  error: Error | null;
  refetch: () => void;
}

/**
 * Mutation options type for consistent error handling
 */
export interface MutationCallbacks {
  onSuccess?: () => void;
  onError?: (_error: Error) => void;
}
