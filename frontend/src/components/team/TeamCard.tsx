'use client';

import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Users, MapPin, Trophy } from 'lucide-react';

export interface Team {
  id: string;
  clubId?: string;
  name: string;
  shortName?: string;
  division?: string;
  region?: string;
  createdAt: string;
  updatedAt: string;
}

export interface TeamWithStats extends Team {
  memberCount?: number;
  matchesPlayed?: number;
  wins?: number;
  draws?: number;
  losses?: number;
}

interface TeamCardProps {
  team: TeamWithStats;
  onClick?: () => void;
  selected?: boolean;
  showStats?: boolean;
}

export function TeamCard({ team, onClick, selected, showStats = false }: TeamCardProps) {
  return (
    <Card
      className={`cursor-pointer transition-all hover:shadow-md ${
        selected ? 'ring-2 ring-primary' : ''
      }`}
      onClick={onClick}
    >
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between">
          <div>
            <CardTitle className="text-lg">{team.name}</CardTitle>
            {team.shortName && (
              <p className="text-sm text-muted-foreground">{team.shortName}</p>
            )}
          </div>
          {team.division && (
            <Badge variant="secondary">{team.division}</Badge>
          )}
        </div>
      </CardHeader>
      <CardContent>
        <div className="flex flex-wrap gap-4 text-sm text-muted-foreground">
          {team.region && (
            <div className="flex items-center gap-1">
              <MapPin className="h-4 w-4" />
              <span>{team.region}</span>
            </div>
          )}
          {team.memberCount !== undefined && (
            <div className="flex items-center gap-1">
              <Users className="h-4 w-4" />
              <span>{team.memberCount} players</span>
            </div>
          )}
        </div>
        {showStats && team.matchesPlayed !== undefined && (
          <div className="mt-3 flex items-center gap-2">
            <Trophy className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm">
              {team.wins}W / {team.draws}D / {team.losses}L
            </span>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

interface TeamListProps {
  teams: TeamWithStats[];
  selectedId?: string;
  onSelect?: (team: TeamWithStats) => void;
  showStats?: boolean;
}

export function TeamList({ teams, selectedId, onSelect, showStats }: TeamListProps) {
  if (teams.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        No teams found
      </div>
    );
  }

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {teams.map((team) => (
        <TeamCard
          key={team.id}
          team={team}
          selected={team.id === selectedId}
          onClick={() => onSelect?.(team)}
          showStats={showStats}
        />
      ))}
    </div>
  );
}
