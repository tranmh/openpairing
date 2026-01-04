'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Calendar,
  MapPin,
  Users,
  Trophy,
  Plus,
  Search,
  Filter,
} from 'lucide-react';
import { api } from '@/lib/api';

interface Tournament {
  id: string;
  name: string;
  slug: string;
  type: string;
  mode: string;
  status: string;
  startDate: string;
  endDate?: string;
  rounds: number;
  venueCity?: string;
  maxPlayers?: number;
  participantCount?: number;
  fideRated: boolean;
  dwzRated: boolean;
}

const statusColors: Record<string, string> = {
  draft: 'bg-gray-100 text-gray-800',
  published: 'bg-blue-100 text-blue-800',
  registration: 'bg-green-100 text-green-800',
  ongoing: 'bg-yellow-100 text-yellow-800',
  completed: 'bg-purple-100 text-purple-800',
  cancelled: 'bg-red-100 text-red-800',
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

export default function TournamentsPage() {
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');

  const { data, isLoading, error } = useQuery({
    queryKey: ['tournaments', statusFilter],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (statusFilter && statusFilter !== 'all') {
        params.append('status', statusFilter);
      }
      params.append('limit', '20');
      const response = await api.get(`/tournaments?${params.toString()}`);
      return response.data;
    },
  });

  const tournaments = data?.tournaments || [];

  const filteredTournaments = tournaments.filter((t: Tournament) =>
    t.name.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="container mx-auto py-6 space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Turniere</h1>
          <p className="text-muted-foreground">
            Verwalten Sie Ihre Schachturniere
          </p>
        </div>
        <Link href="/tournaments/new">
          <Button>
            <Plus className="mr-2 h-4 w-4" />
            Neues Turnier
          </Button>
        </Link>
      </div>

      <div className="flex gap-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Turniere suchen..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-10"
          />
        </div>
        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-[200px]">
            <Filter className="mr-2 h-4 w-4" />
            <SelectValue placeholder="Status filtern" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Alle Status</SelectItem>
            <SelectItem value="draft">Entwurf</SelectItem>
            <SelectItem value="published">Veröffentlicht</SelectItem>
            <SelectItem value="registration">Anmeldung</SelectItem>
            <SelectItem value="ongoing">Laufend</SelectItem>
            <SelectItem value="completed">Beendet</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[...Array(6)].map((_, i) => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="h-6 w-3/4" />
                <Skeleton className="h-4 w-1/2" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-4 w-full mb-2" />
                <Skeleton className="h-4 w-2/3" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : error ? (
        <Card>
          <CardContent className="py-10 text-center">
            <p className="text-muted-foreground">
              Fehler beim Laden der Turniere
            </p>
          </CardContent>
        </Card>
      ) : filteredTournaments.length === 0 ? (
        <Card>
          <CardContent className="py-10 text-center">
            <Trophy className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">Keine Turniere gefunden</h3>
            <p className="text-muted-foreground mb-4">
              Erstellen Sie Ihr erstes Turnier
            </p>
            <Link href="/tournaments/new">
              <Button>
                <Plus className="mr-2 h-4 w-4" />
                Turnier erstellen
              </Button>
            </Link>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {filteredTournaments.map((tournament: Tournament) => (
            <TournamentCard key={tournament.id} tournament={tournament} />
          ))}
        </div>
      )}
    </div>
  );
}

function TournamentCard({ tournament }: { tournament: Tournament }) {
  const startDate = new Date(tournament.startDate).toLocaleDateString('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  });

  return (
    <Link href={`/tournaments/${tournament.id}`}>
      <Card className="hover:shadow-lg transition-shadow cursor-pointer h-full">
        <CardHeader>
          <div className="flex justify-between items-start">
            <CardTitle className="text-lg line-clamp-2">{tournament.name}</CardTitle>
            <Badge className={statusColors[tournament.status]}>
              {statusLabels[tournament.status]}
            </Badge>
          </div>
          <CardDescription>
            {typeLabels[tournament.type] || tournament.type}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center text-sm text-muted-foreground">
            <Calendar className="mr-2 h-4 w-4" />
            {startDate}
            {tournament.endDate && (
              <>
                {' - '}
                {new Date(tournament.endDate).toLocaleDateString('de-DE', {
                  day: '2-digit',
                  month: '2-digit',
                  year: 'numeric',
                })}
              </>
            )}
          </div>

          {tournament.venueCity && (
            <div className="flex items-center text-sm text-muted-foreground">
              <MapPin className="mr-2 h-4 w-4" />
              {tournament.venueCity}
            </div>
          )}

          <div className="flex items-center text-sm text-muted-foreground">
            <Users className="mr-2 h-4 w-4" />
            {tournament.participantCount || 0}
            {tournament.maxPlayers && ` / ${tournament.maxPlayers}`} Teilnehmer
          </div>

          <div className="flex items-center text-sm text-muted-foreground">
            <Trophy className="mr-2 h-4 w-4" />
            {tournament.rounds} Runden
          </div>

          <div className="flex gap-2 pt-2">
            {tournament.fideRated && (
              <Badge variant="outline" className="text-xs">FIDE</Badge>
            )}
            {tournament.dwzRated && (
              <Badge variant="outline" className="text-xs">DWZ</Badge>
            )}
          </div>
        </CardContent>
      </Card>
    </Link>
  );
}
