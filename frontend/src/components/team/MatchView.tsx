'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Swords, Trophy, Minus, Circle } from 'lucide-react';

export interface TeamMatchBoard {
  id: string;
  teamMatchId: string;
  gameId?: string;
  boardNumber: number;
  homePlayerId?: string;
  awayPlayerId?: string;
  homeRating?: number;
  awayRating?: number;
  homeScore?: number;
  awayScore?: number;
  homePlayerName?: string;
  awayPlayerName?: string;
}

export interface TeamMatch {
  id: string;
  tournamentId: string;
  roundId: string;
  matchNumber: number;
  homeTeamId?: string;
  awayTeamId?: string;
  homeMatchPoints?: number;
  awayMatchPoints?: number;
  homeBoardPoints?: number;
  awayBoardPoints?: number;
  status: 'scheduled' | 'ongoing' | 'completed';
  createdAt: string;
  homeTeamName?: string;
  awayTeamName?: string;
}

interface MatchViewProps {
  match: TeamMatch;
  boards: TeamMatchBoard[];
  onBoardClick?: (board: TeamMatchBoard) => void;
}

export function MatchView({ match, boards, onBoardClick }: MatchViewProps) {
  const statusColors = {
    scheduled: 'secondary',
    ongoing: 'default',
    completed: 'outline',
  } as const;

  const sortedBoards = [...boards].sort((a, b) => a.boardNumber - b.boardNumber);

  const getResultIcon = (homeScore?: number, awayScore?: number) => {
    if (homeScore === undefined || awayScore === undefined) {
      return <Circle className="h-4 w-4 text-muted-foreground" />;
    }
    if (homeScore > awayScore) {
      return <Trophy className="h-4 w-4 text-green-500" />;
    }
    if (homeScore < awayScore) {
      return <Minus className="h-4 w-4 text-red-500" />;
    }
    return <Swords className="h-4 w-4 text-yellow-500" />;
  };

  const formatScore = (score?: number) => {
    if (score === undefined) return '-';
    if (score === 1) return '1';
    if (score === 0.5) return '½';
    return '0';
  };

  const getMatchResult = () => {
    if (match.status !== 'completed') return null;

    const home = match.homeBoardPoints ?? 0;
    const away = match.awayBoardPoints ?? 0;

    if (home > away) return 'home';
    if (away > home) return 'away';
    return 'draw';
  };

  const result = getMatchResult();

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <Swords className="h-5 w-5" />
            Match #{match.matchNumber}
          </CardTitle>
          <Badge variant={statusColors[match.status]}>
            {match.status.charAt(0).toUpperCase() + match.status.slice(1)}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Match header with team names and score */}
        <div className="flex items-center justify-between p-4 bg-muted/50 rounded-lg">
          <div className={`flex-1 text-center ${result === 'home' ? 'font-bold' : ''}`}>
            <div className="text-lg">{match.homeTeamName ?? 'Home Team'}</div>
            {match.homeBoardPoints !== undefined && (
              <div className="text-2xl font-bold mt-1">{match.homeBoardPoints}</div>
            )}
          </div>
          <div className="px-4 text-muted-foreground">vs</div>
          <div className={`flex-1 text-center ${result === 'away' ? 'font-bold' : ''}`}>
            <div className="text-lg">{match.awayTeamName ?? 'Away Team'}</div>
            {match.awayBoardPoints !== undefined && (
              <div className="text-2xl font-bold mt-1">{match.awayBoardPoints}</div>
            )}
          </div>
        </div>

        {/* Match points if available */}
        {match.homeMatchPoints !== undefined && (
          <div className="text-center text-sm text-muted-foreground">
            Match Points: {match.homeMatchPoints} - {match.awayMatchPoints}
          </div>
        )}

        {/* Board results */}
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-16">Board</TableHead>
              <TableHead>Home</TableHead>
              <TableHead className="w-16 text-center">Rating</TableHead>
              <TableHead className="w-24 text-center">Result</TableHead>
              <TableHead className="w-16 text-center">Rating</TableHead>
              <TableHead className="text-right">Away</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sortedBoards.map((board) => (
              <TableRow
                key={board.id}
                className={onBoardClick ? 'cursor-pointer hover:bg-muted/50' : ''}
                onClick={() => onBoardClick?.(board)}
              >
                <TableCell className="font-medium">{board.boardNumber}</TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    {getResultIcon(board.homeScore, board.awayScore)}
                    {board.homePlayerName ?? '-'}
                  </div>
                </TableCell>
                <TableCell className="text-center text-muted-foreground">
                  {board.homeRating ?? '-'}
                </TableCell>
                <TableCell className="text-center font-mono">
                  {formatScore(board.homeScore)} - {formatScore(board.awayScore)}
                </TableCell>
                <TableCell className="text-center text-muted-foreground">
                  {board.awayRating ?? '-'}
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex items-center justify-end gap-2">
                    {board.awayPlayerName ?? '-'}
                    {getResultIcon(board.awayScore, board.homeScore)}
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

interface MatchListProps {
  matches: TeamMatch[];
  onMatchClick?: (match: TeamMatch) => void;
}

export function MatchList({ matches, onMatchClick }: MatchListProps) {
  if (matches.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        No matches scheduled
      </div>
    );
  }

  const sortedMatches = [...matches].sort((a, b) => a.matchNumber - b.matchNumber);

  return (
    <div className="space-y-4">
      {sortedMatches.map((match) => (
        <Card
          key={match.id}
          className={`${onMatchClick ? 'cursor-pointer hover:shadow-md' : ''}`}
          onClick={() => onMatchClick?.(match)}
        >
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                <Badge variant="outline">#{match.matchNumber}</Badge>
                <div className="font-medium">
                  {match.homeTeamName ?? 'TBD'}
                </div>
                <span className="text-muted-foreground">vs</span>
                <div className="font-medium">
                  {match.awayTeamName ?? 'TBD'}
                </div>
              </div>
              <div className="flex items-center gap-4">
                {match.status === 'completed' && (
                  <span className="font-bold">
                    {match.homeBoardPoints} - {match.awayBoardPoints}
                  </span>
                )}
                <Badge
                  variant={
                    match.status === 'ongoing'
                      ? 'default'
                      : match.status === 'completed'
                      ? 'outline'
                      : 'secondary'
                  }
                >
                  {match.status}
                </Badge>
              </div>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
