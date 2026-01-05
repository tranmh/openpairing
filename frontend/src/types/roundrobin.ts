// Round Robin / Rutsch-System types

export interface RoundRobinSettings {
  byePosition: number;
  fixedPlayer: number;
  showSeatingChart: boolean;
}

export interface SeatingPosition {
  seatNumber: number;
  startNumber: number;
  playerName?: string;
  rating?: number;
  isFixed: boolean;
  isBye: boolean;
}

export interface SeatingPairing {
  board: number;
  whiteSeat: number;
  blackSeat: number;
  whiteStartNo: number;
  blackStartNo: number;
  whiteName?: string;
  blackName?: string;
  isBye: boolean;
}

export interface SeatingChartData {
  tournamentId: string;
  tournamentName: string;
  numPlayers: number;
  numSeats: number;
  fixedSeat: number;
  byeSeat: number;
  seats: SeatingPosition[];
  pairings: SeatingPairing[];
  roundNumber: number;
}

export interface SeatMovement {
  startNumber: number;
  playerName?: string;
  fromSeat: number;
  toSeat: number;
  isFixed: boolean;
}

export interface RotationData {
  fromRound: number;
  toRound: number;
  beforeSeats: SeatingPosition[];
  afterSeats: SeatingPosition[];
  movements: SeatMovement[];
}

export interface RoundRobinSettingsResponse {
  settings: RoundRobinSettings;
}

export interface UpdateRoundRobinSettingsRequest {
  byePosition?: number;
  fixedPlayer?: number;
  showSeatingChart?: boolean;
}
