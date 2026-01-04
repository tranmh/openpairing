'use client';

import Link from 'next/link';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Calendar, MapPin, Users, Trophy } from 'lucide-react';
import { cn } from '@/lib/utils';

interface Tournament {
  id: string;
  name: string;
  slug?: string;
  type: string;
  mode: string;
  status: string;
  startDate: string;
  endDate?: string;
  rounds: number;
  venueCity?: string;
  maxPlayers?: number;
  participantCount?: number;
  fideRated?: boolean;
  dwzRated?: boolean;
}

interface TournamentCardProps {
  tournament: Tournament;
  href?: string;
  variant?: 'default' | 'compact';
}

const statusColors: Record<string, string> = {
  draft: 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200',
  published: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
  registration: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
  ongoing: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
  completed: 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200',
  cancelled: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
};

const statusLabels: Record<string, string> = {
  draft: 'Entwurf',
  published: 'Veröffentlicht',
  registration: 'Anmeldung',
  ongoing: 'Laufend',
  completed: 'Beendet',
  cancelled: 'Abgesagt',
};

const typeLabels: Record<string, string> = {
  swiss: 'Schweizer System',
  round_robin: 'Rundenturnier',
  knockout: 'K.O.-System',
  team_swiss: 'Mannschaftsturnier',
};

const modeLabels: Record<string, string> = {
  otb: 'Präsenz',
  online: 'Online',
  hybrid: 'Hybrid',
};

export function TournamentCard({
  tournament,
  href,
  variant = 'default',
}: TournamentCardProps) {
  const startDate = new Date(tournament.startDate).toLocaleDateString('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  });

  const endDate = tournament.endDate
    ? new Date(tournament.endDate).toLocaleDateString('de-DE', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
      })
    : null;

  const content = (
    <Card
      className={cn(
        'transition-all duration-200',
        href && 'hover:shadow-lg hover:border-primary/50 cursor-pointer'
      )}
    >
      <CardHeader className={cn(variant === 'compact' && 'pb-2')}>
        <div className="flex justify-between items-start gap-2">
          <CardTitle
            className={cn(
              'line-clamp-2',
              variant === 'compact' ? 'text-base' : 'text-lg'
            )}
          >
            {tournament.name}
          </CardTitle>
          <Badge className={statusColors[tournament.status]}>
            {statusLabels[tournament.status] || tournament.status}
          </Badge>
        </div>
        <CardDescription className="flex items-center gap-2">
          <span>{typeLabels[tournament.type] || tournament.type}</span>
          <span>•</span>
          <span>{modeLabels[tournament.mode] || tournament.mode}</span>
        </CardDescription>
      </CardHeader>
      <CardContent className={cn('space-y-2', variant === 'compact' && 'pt-0')}>
        <div className="flex items-center text-sm text-muted-foreground">
          <Calendar className="mr-2 h-4 w-4 flex-shrink-0" />
          <span>
            {startDate}
            {endDate && endDate !== startDate && ` - ${endDate}`}
          </span>
        </div>

        {tournament.venueCity && (
          <div className="flex items-center text-sm text-muted-foreground">
            <MapPin className="mr-2 h-4 w-4 flex-shrink-0" />
            <span>{tournament.venueCity}</span>
          </div>
        )}

        <div className="flex items-center text-sm text-muted-foreground">
          <Users className="mr-2 h-4 w-4 flex-shrink-0" />
          <span>
            {tournament.participantCount || 0}
            {tournament.maxPlayers && ` / ${tournament.maxPlayers}`} Teilnehmer
          </span>
        </div>

        <div className="flex items-center text-sm text-muted-foreground">
          <Trophy className="mr-2 h-4 w-4 flex-shrink-0" />
          <span>{tournament.rounds} Runden</span>
        </div>

        {(tournament.fideRated || tournament.dwzRated) && (
          <div className="flex gap-2 pt-2">
            {tournament.fideRated && (
              <Badge variant="outline" className="text-xs">
                FIDE
              </Badge>
            )}
            {tournament.dwzRated && (
              <Badge variant="outline" className="text-xs">
                DWZ
              </Badge>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );

  if (href) {
    return <Link href={href}>{content}</Link>;
  }

  return content;
}
