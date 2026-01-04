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
import { Trophy, Medal, Award } from 'lucide-react';

export interface TeamStanding {
  rank: number;
  teamParticipantId: string;
  teamId: string;
  teamName: string;
  matchPoints: number;
  boardPoints: number;
  buchholz: number;
  wins: number;
  draws: number;
  losses: number;
  matchesPlayed: number;
}

interface TeamStandingsProps {
  standings: TeamStanding[];
  highlightTeamId?: string;
  onTeamClick?: (standing: TeamStanding) => void;
  compact?: boolean;
}

export function TeamStandings({
  standings,
  highlightTeamId,
  onTeamClick,
  compact = false,
}: TeamStandingsProps) {
  const getRankIcon = (rank: number) => {
    switch (rank) {
      case 1:
        return <Trophy className="h-5 w-5 text-yellow-500" />;
      case 2:
        return <Medal className="h-5 w-5 text-gray-400" />;
      case 3:
        return <Award className="h-5 w-5 text-amber-600" />;
      default:
        return null;
    }
  };

  const getRankBadge = (rank: number) => {
    if (rank === 1) return 'default';
    if (rank <= 3) return 'secondary';
    return 'outline';
  };

  if (standings.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        No standings available
      </div>
    );
  }

  const sortedStandings = [...standings].sort((a, b) => a.rank - b.rank);

  if (compact) {
    return (
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-12">#</TableHead>
            <TableHead>Team</TableHead>
            <TableHead className="w-16 text-right">MP</TableHead>
            <TableHead className="w-16 text-right">BP</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {sortedStandings.slice(0, 10).map((standing) => (
            <TableRow
              key={standing.teamParticipantId}
              className={`
                ${onTeamClick ? 'cursor-pointer hover:bg-muted/50' : ''}
                ${standing.teamId === highlightTeamId ? 'bg-primary/10' : ''}
              `}
              onClick={() => onTeamClick?.(standing)}
            >
              <TableCell>
                <Badge variant={getRankBadge(standing.rank)}>{standing.rank}</Badge>
              </TableCell>
              <TableCell className="font-medium">{standing.teamName}</TableCell>
              <TableCell className="text-right">{standing.matchPoints}</TableCell>
              <TableCell className="text-right">{standing.boardPoints}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Trophy className="h-5 w-5" />
          Team Standings
        </CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-16">Rank</TableHead>
              <TableHead>Team</TableHead>
              <TableHead className="w-20 text-center">Played</TableHead>
              <TableHead className="w-24 text-center">W/D/L</TableHead>
              <TableHead className="w-20 text-right">MP</TableHead>
              <TableHead className="w-20 text-right">BP</TableHead>
              <TableHead className="w-20 text-right">BH</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sortedStandings.map((standing) => (
              <TableRow
                key={standing.teamParticipantId}
                className={`
                  ${onTeamClick ? 'cursor-pointer hover:bg-muted/50' : ''}
                  ${standing.teamId === highlightTeamId ? 'bg-primary/10' : ''}
                `}
                onClick={() => onTeamClick?.(standing)}
              >
                <TableCell>
                  <div className="flex items-center gap-2">
                    {getRankIcon(standing.rank)}
                    <Badge variant={getRankBadge(standing.rank)}>{standing.rank}</Badge>
                  </div>
                </TableCell>
                <TableCell className="font-medium">{standing.teamName}</TableCell>
                <TableCell className="text-center">{standing.matchesPlayed}</TableCell>
                <TableCell className="text-center">
                  <span className="text-green-600">{standing.wins}</span>
                  <span className="text-muted-foreground">/</span>
                  <span className="text-yellow-600">{standing.draws}</span>
                  <span className="text-muted-foreground">/</span>
                  <span className="text-red-600">{standing.losses}</span>
                </TableCell>
                <TableCell className="text-right font-medium">{standing.matchPoints}</TableCell>
                <TableCell className="text-right">{standing.boardPoints}</TableCell>
                <TableCell className="text-right text-muted-foreground">
                  {standing.buchholz.toFixed(1)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

interface StandingsPodiumProps {
  standings: TeamStanding[];
  onTeamClick?: (standing: TeamStanding) => void;
}

export function StandingsPodium({ standings, onTeamClick }: StandingsPodiumProps) {
  const top3 = standings.filter((s) => s.rank <= 3).sort((a, b) => a.rank - b.rank);

  if (top3.length === 0) return null;

  const [first, second, third] = top3;

  return (
    <div className="flex items-end justify-center gap-4 py-8">
      {/* Second place */}
      {second && (
        <PodiumCard
          standing={second}
          height="h-32"
          icon={<Medal className="h-8 w-8 text-gray-400" />}
          onClick={() => onTeamClick?.(second)}
        />
      )}
      {/* First place */}
      {first && (
        <PodiumCard
          standing={first}
          height="h-40"
          icon={<Trophy className="h-10 w-10 text-yellow-500" />}
          onClick={() => onTeamClick?.(first)}
        />
      )}
      {/* Third place */}
      {third && (
        <PodiumCard
          standing={third}
          height="h-24"
          icon={<Award className="h-6 w-6 text-amber-600" />}
          onClick={() => onTeamClick?.(third)}
        />
      )}
    </div>
  );
}

interface PodiumCardProps {
  standing: TeamStanding;
  height: string;
  icon: React.ReactNode;
  onClick?: () => void;
}

function PodiumCard({ standing, height, icon, onClick }: PodiumCardProps) {
  return (
    <Card
      className={`w-40 ${height} flex flex-col cursor-pointer hover:shadow-lg transition-shadow`}
      onClick={onClick}
    >
      <CardContent className="flex flex-col items-center justify-center h-full p-4">
        {icon}
        <div className="text-center mt-2">
          <div className="font-bold text-sm truncate max-w-full">{standing.teamName}</div>
          <div className="text-xs text-muted-foreground mt-1">
            {standing.matchPoints} pts
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
