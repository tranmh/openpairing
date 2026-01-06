// Arbiter Console types

/**
 * Tournament status response from GET /arbiter/tournaments/:id/status
 */
export interface TournamentStatus {
  tournament: TournamentInfo;
  currentRound?: CurrentRound;
  rounds: RoundInfo[];
  participants: ParticipantInfo[];
  activeGames: ActiveGame[];
  pendingResults: number;
  statistics: TournamentStatistics;
}

export interface TournamentInfo {
  id: string;
  name: string;
  status: string;
  rounds: number;
}

export interface CurrentRound {
  id: string;
  roundNumber: number;
  status: RoundStatus;
  actualStart?: string;
}

export interface RoundInfo {
  id: string;
  roundNumber: number;
  status: RoundStatus;
}

export type RoundStatus = 'pending' | 'pairing' | 'paired' | 'ongoing' | 'completed';

export interface ParticipantInfo {
  id: string;
  playerId: string;
  playerName?: string;
  playerFirstName?: string;
  playerLastName?: string;
  startRank: number;
  startingRating: number;
  manualRatingOverride?: number | null;
  points: number;
  isActive: boolean;
}

export interface ActiveGame {
  game: GameInfo;
  whitePlayer?: PlayerInfo;
  blackPlayer?: PlayerInfo;
  elapsedMinutes: number;
}

export interface GameInfo {
  id: string;
  boardNumber: number;
  result?: string;
}

export interface PlayerInfo {
  participantId: string;
  name: string;
  rating: number;
}

export interface TournamentStatistics {
  totalParticipants: number;
  activeParticipants: number;
  withdrawnCount: number;
  completedRounds: number;
  averageRating: number;
}

/**
 * Pairing game type for pairing management
 */
export interface PairingGame {
  id: string;
  boardNumber: number;
  whiteParticipantId?: string;
  blackParticipantId?: string;
  result?: string;
  status: string;
  whiteName?: string;
  blackName?: string;
  whiteRating?: number;
  blackRating?: number;
}

/**
 * Audit log entry
 */
export interface AuditAction {
  id: string;
  timestamp: string;
  actionType: string;
  entity: string;
  entityId: string;
  oldValue?: Record<string, unknown>;
  newValue?: Record<string, unknown>;
  reason?: string;
  arbiterName: string;
}

/**
 * Result type for game results
 */
export type GameResult =
  | '1-0'
  | '0-1'
  | '1/2-1/2'
  | '1-0F'
  | '0-1F'
  | '0-0F';

/**
 * Bye type
 */
export type ByeType = 'full' | 'half' | 'zero';

/**
 * Announcement priority
 */
export type AnnouncementPriority = 'info' | 'warning' | 'urgent';

/**
 * API request/response types for arbiter endpoints
 */
export interface WithdrawPlayerRequest {
  participantId: string;
  reason?: string;
  effectiveFrom?: number;
}

export interface ReactivatePlayerRequest {
  participantId: string;
  missedRoundPoints?: number;
  reason?: string;
}

export interface AssignByeRequest {
  participantId: string;
  roundNumber: number;
  byeType: ByeType;
}

export interface CorrectResultRequest {
  gameId: string;
  newResult: GameResult;
  reason: string;
}

export interface SwapPairingRequest {
  roundNumber: number;
  game1Id: string;
  player1Id: string;
  game2Id: string;
  player2Id: string;
  reason?: string;
}

export interface ChangeColorRequest {
  gameId: string;
  reason?: string;
}

export interface CreateManualPairingRequest {
  roundNumber: number;
  whiteParticipantId: string;
  blackParticipantId?: string | null;
  boardNumber?: number;
}

export interface CreateManualPairingResponse {
  gameId: string;
  warnings?: Array<{
    type: 'forbidden_pair' | 'previous_encounter';
    message: string;
  }>;
}

export interface CreateAnnouncementRequest {
  title: string;
  message: string;
  priority: AnnouncementPriority;
}

export interface AnnouncementResponse {
  id: string;
  title: string;
  message: string;
  priority: AnnouncementPriority;
  createdAt: string;
}
