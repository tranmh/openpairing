'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Calendar,
  MapPin,
  Users,
  Trophy,
  Settings,
  Play,
  Pause,
  Download,
  Share2,
  Edit,
  CheckCircle,
} from 'lucide-react';
import { api } from '@/lib/api';
import { toast } from 'sonner';

interface Tournament {
  id: string;
  name: string;
  description?: string;
  slug: string;
  type: string;
  mode: string;
  status: string;
  startDate: string;
  endDate?: string;
  rounds: number;
  timeControl: string;
  venueCity?: string;
  venueName?: string;
  venueAddress?: string;
  maxPlayers?: number;
  minPlayers: number;
  fideRated: boolean;
  dwzRated: boolean;
  pairingSystem: string;
  tiebreaks: string[];
}

interface Participant {
  id: string;
  playerId: string;
  startRank: number;
  startingRating: number;
  points: number;
  buchholz: number;
  isActive: boolean;
  playerName?: string;
}

interface Round {
  id: string;
  roundNumber: number;
  status: string;
  scheduledStart?: string;
  actualStart?: string;
  completedAt?: string;
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

export default function TournamentDetailPage() {
  const params = useParams();
  const router = useRouter();
  const queryClient = useQueryClient();
  const tournamentId = params.id as string;

  const { data: tournamentData, isLoading: tournamentLoading } = useQuery({
    queryKey: ['tournament', tournamentId],
    queryFn: async () => {
      const response = await api.get(`/tournaments/${tournamentId}`);
      return response.data;
    },
  });

  const { data: participantsData } = useQuery({
    queryKey: ['participants', tournamentId],
    queryFn: async () => {
      const response = await api.get(`/tournaments/${tournamentId}/participants`);
      return response.data;
    },
  });

  const { data: roundsData } = useQuery({
    queryKey: ['rounds', tournamentId],
    queryFn: async () => {
      const response = await api.get(`/tournaments/${tournamentId}/rounds`);
      return response.data;
    },
  });

  const publishMutation = useMutation({
    mutationFn: () => api.post(`/tournaments/${tournamentId}/publish`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tournament', tournamentId] });
      toast.success('Turnier veröffentlicht');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler beim Veröffentlichen');
    },
  });

  const startRegistrationMutation = useMutation({
    mutationFn: () => api.post(`/tournaments/${tournamentId}/start-registration`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tournament', tournamentId] });
      toast.success('Anmeldung geöffnet');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler beim Öffnen der Anmeldung');
    },
  });

  const startTournamentMutation = useMutation({
    mutationFn: () => api.post(`/tournaments/${tournamentId}/start`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tournament', tournamentId] });
      queryClient.invalidateQueries({ queryKey: ['rounds', tournamentId] });
      toast.success('Turnier gestartet');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler beim Starten');
    },
  });

  const tournament: Tournament | undefined = tournamentData?.tournament;
  const participants: Participant[] = participantsData?.participants || [];
  const rounds: Round[] = roundsData?.rounds || [];

  if (tournamentLoading) {
    return (
      <div className="container mx-auto py-6 space-y-6">
        <Skeleton className="h-10 w-1/3" />
        <Skeleton className="h-6 w-1/4" />
        <div className="grid gap-4 md:grid-cols-3">
          <Skeleton className="h-32" />
          <Skeleton className="h-32" />
          <Skeleton className="h-32" />
        </div>
      </div>
    );
  }

  if (!tournament) {
    return (
      <div className="container mx-auto py-6">
        <Card>
          <CardContent className="py-10 text-center">
            <p className="text-muted-foreground">Turnier nicht gefunden</p>
          </CardContent>
        </Card>
      </div>
    );
  }

  const startDate = new Date(tournament.startDate).toLocaleDateString('de-DE', {
    weekday: 'long',
    day: '2-digit',
    month: 'long',
    year: 'numeric',
  });

  return (
    <div className="container mx-auto py-6 space-y-6">
      <div className="flex justify-between items-start">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <h1 className="text-3xl font-bold tracking-tight">{tournament.name}</h1>
            <Badge className={statusColors[tournament.status]}>
              {statusLabels[tournament.status]}
            </Badge>
          </div>
          {tournament.description && (
            <p className="text-muted-foreground">{tournament.description}</p>
          )}
        </div>
        <div className="flex gap-2">
          {tournament.status === 'draft' && (
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button>
                  <CheckCircle className="mr-2 h-4 w-4" />
                  Veröffentlichen
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Turnier veröffentlichen?</AlertDialogTitle>
                  <AlertDialogDescription>
                    Nach der Veröffentlichung können Sie die Anmeldung öffnen.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Abbrechen</AlertDialogCancel>
                  <AlertDialogAction onClick={() => publishMutation.mutate()}>
                    Veröffentlichen
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          )}
          {tournament.status === 'published' && (
            <Button onClick={() => startRegistrationMutation.mutate()}>
              <Users className="mr-2 h-4 w-4" />
              Anmeldung öffnen
            </Button>
          )}
          {tournament.status === 'registration' && (
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button>
                  <Play className="mr-2 h-4 w-4" />
                  Turnier starten
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Turnier starten?</AlertDialogTitle>
                  <AlertDialogDescription>
                    Die Anmeldung wird geschlossen und Runde 1 wird erstellt.
                    Derzeit sind {participants.filter(p => p.isActive).length} Spieler angemeldet.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Abbrechen</AlertDialogCancel>
                  <AlertDialogAction onClick={() => startTournamentMutation.mutate()}>
                    Starten
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          )}
          <Link href={`/tournaments/${tournamentId}/settings`}>
            <Button variant="outline">
              <Settings className="mr-2 h-4 w-4" />
              Einstellungen
            </Button>
          </Link>
          <Button variant="outline" onClick={() => window.open(`/api/tournaments/${tournamentId}/export/trf`)}>
            <Download className="mr-2 h-4 w-4" />
            TRF Export
          </Button>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Datum
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center">
              <Calendar className="mr-2 h-4 w-4 text-muted-foreground" />
              <span className="font-medium">{startDate}</span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Ort
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center">
              <MapPin className="mr-2 h-4 w-4 text-muted-foreground" />
              <span className="font-medium">
                {tournament.venueCity || 'Nicht angegeben'}
              </span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Teilnehmer
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center">
              <Users className="mr-2 h-4 w-4 text-muted-foreground" />
              <span className="font-medium">
                {participants.filter(p => p.isActive).length}
                {tournament.maxPlayers && ` / ${tournament.maxPlayers}`}
              </span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Runden
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center">
              <Trophy className="mr-2 h-4 w-4 text-muted-foreground" />
              <span className="font-medium">
                {rounds.filter(r => r.status === 'completed').length} / {tournament.rounds}
              </span>
            </div>
          </CardContent>
        </Card>
      </div>

      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList>
          <TabsTrigger value="overview">Übersicht</TabsTrigger>
          <TabsTrigger value="participants">Teilnehmer</TabsTrigger>
          <TabsTrigger value="pairings">Paarungen</TabsTrigger>
          <TabsTrigger value="standings">Tabelle</TabsTrigger>
        </TabsList>

        <TabsContent value="overview">
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Turnierinformationen</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">System</span>
                  <span className="font-medium capitalize">
                    {tournament.pairingSystem}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Bedenkzeit</span>
                  <span className="font-medium">{tournament.timeControl}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Runden</span>
                  <span className="font-medium">{tournament.rounds}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Wertung</span>
                  <span className="font-medium">
                    {[
                      tournament.fideRated && 'FIDE',
                      tournament.dwzRated && 'DWZ',
                    ]
                      .filter(Boolean)
                      .join(', ') || 'Keine'}
                  </span>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Feinwertungen</CardTitle>
              </CardHeader>
              <CardContent>
                <ol className="list-decimal list-inside space-y-1">
                  {tournament.tiebreaks.map((tb, i) => (
                    <li key={i} className="text-sm">
                      {tiebreakLabels[tb] || tb}
                    </li>
                  ))}
                </ol>
              </CardContent>
            </Card>

            {tournament.venueName && (
              <Card className="md:col-span-2">
                <CardHeader>
                  <CardTitle>Spielort</CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="font-medium">{tournament.venueName}</p>
                  {tournament.venueAddress && (
                    <p className="text-muted-foreground">{tournament.venueAddress}</p>
                  )}
                  {tournament.venueCity && (
                    <p className="text-muted-foreground">{tournament.venueCity}</p>
                  )}
                </CardContent>
              </Card>
            )}
          </div>
        </TabsContent>

        <TabsContent value="participants">
          <ParticipantsTab
            tournamentId={tournamentId}
            participants={participants}
            canRegister={tournament.status === 'registration'}
          />
        </TabsContent>

        <TabsContent value="pairings">
          <PairingsTab tournamentId={tournamentId} rounds={rounds} />
        </TabsContent>

        <TabsContent value="standings">
          <StandingsTab tournamentId={tournamentId} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

function ParticipantsTab({
  tournamentId,
  participants,
  canRegister,
}: {
  tournamentId: string;
  participants: Participant[];
  canRegister: boolean;
}) {
  const activeParticipants = participants.filter(p => p.isActive);

  return (
    <Card>
      <CardHeader>
        <div className="flex justify-between items-center">
          <CardTitle>Teilnehmerliste ({activeParticipants.length})</CardTitle>
          {canRegister && (
            <Link href={`/tournaments/${tournamentId}/register`}>
              <Button>
                <Users className="mr-2 h-4 w-4" />
                Spieler anmelden
              </Button>
            </Link>
          )}
        </div>
      </CardHeader>
      <CardContent>
        {activeParticipants.length === 0 ? (
          <p className="text-muted-foreground text-center py-8">
            Noch keine Teilnehmer angemeldet
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b">
                  <th className="text-left py-2 px-2">Nr.</th>
                  <th className="text-left py-2 px-2">Name</th>
                  <th className="text-right py-2 px-2">Rating</th>
                  <th className="text-right py-2 px-2">Punkte</th>
                  <th className="text-right py-2 px-2">Buchholz</th>
                </tr>
              </thead>
              <tbody>
                {activeParticipants
                  .sort((a, b) => a.startRank - b.startRank)
                  .map((p) => (
                    <tr key={p.id} className="border-b">
                      <td className="py-2 px-2">{p.startRank}</td>
                      <td className="py-2 px-2">{p.playerName || p.playerId}</td>
                      <td className="py-2 px-2 text-right">{p.startingRating}</td>
                      <td className="py-2 px-2 text-right">{p.points}</td>
                      <td className="py-2 px-2 text-right">{p.buchholz.toFixed(1)}</td>
                    </tr>
                  ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function PairingsTab({
  tournamentId,
  rounds,
}: {
  tournamentId: string;
  rounds: Round[];
}) {
  const [selectedRound, setSelectedRound] = useState(
    rounds.length > 0 ? rounds[rounds.length - 1].roundNumber : 1
  );
  const queryClient = useQueryClient();

  const { data: pairingsData, isLoading } = useQuery({
    queryKey: ['pairings', tournamentId, selectedRound],
    queryFn: async () => {
      const response = await api.get(
        `/tournaments/${tournamentId}/rounds/${selectedRound}/pairings`
      );
      return response.data;
    },
    enabled: rounds.length > 0,
  });

  const generatePairingsMutation = useMutation({
    mutationFn: () =>
      api.post(`/tournaments/${tournamentId}/rounds/${selectedRound}/pair`),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['pairings', tournamentId, selectedRound],
      });
      queryClient.invalidateQueries({ queryKey: ['rounds', tournamentId] });
      toast.success('Paarungen erstellt');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler beim Erstellen der Paarungen');
    },
  });

  const games = pairingsData?.games || [];
  const currentRound = rounds.find(r => r.roundNumber === selectedRound);

  return (
    <Card>
      <CardHeader>
        <div className="flex justify-between items-center">
          <CardTitle>Paarungen</CardTitle>
          <div className="flex gap-2">
            <select
              value={selectedRound}
              onChange={(e) => setSelectedRound(Number(e.target.value))}
              className="border rounded px-3 py-1"
            >
              {Array.from({ length: Math.max(1, rounds.length) }, (_, i) => (
                <option key={i + 1} value={i + 1}>
                  Runde {i + 1}
                </option>
              ))}
            </select>
            {currentRound?.status === 'pending' && (
              <Button onClick={() => generatePairingsMutation.mutate()}>
                Paarungen erstellen
              </Button>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-2">
            {[...Array(5)].map((_, i) => (
              <Skeleton key={i} className="h-10" />
            ))}
          </div>
        ) : games.length === 0 ? (
          <p className="text-muted-foreground text-center py-8">
            Noch keine Paarungen für diese Runde
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b">
                  <th className="text-center py-2 px-2">Brett</th>
                  <th className="text-left py-2 px-2">Weiß</th>
                  <th className="text-center py-2 px-2">Ergebnis</th>
                  <th className="text-right py-2 px-2">Schwarz</th>
                </tr>
              </thead>
              <tbody>
                {games.map((game: any) => (
                  <tr key={game.id} className="border-b">
                    <td className="py-2 px-2 text-center">{game.boardNumber}</td>
                    <td className="py-2 px-2">{game.whiteName || game.whiteParticipantId}</td>
                    <td className="py-2 px-2 text-center font-medium">
                      {game.result || '-'}
                    </td>
                    <td className="py-2 px-2 text-right">
                      {game.blackName || game.blackParticipantId || 'BYE'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function StandingsTab({ tournamentId }: { tournamentId: string }) {
  const { data, isLoading } = useQuery({
    queryKey: ['standings', tournamentId],
    queryFn: async () => {
      const response = await api.get(`/tournaments/${tournamentId}/standings`);
      return response.data;
    },
  });

  const standings = data?.standings || [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Tabelle</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-2">
            {[...Array(10)].map((_, i) => (
              <Skeleton key={i} className="h-10" />
            ))}
          </div>
        ) : standings.length === 0 ? (
          <p className="text-muted-foreground text-center py-8">
            Noch keine Ergebnisse
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b">
                  <th className="text-center py-2 px-2">Platz</th>
                  <th className="text-left py-2 px-2">Name</th>
                  <th className="text-right py-2 px-2">Rating</th>
                  <th className="text-right py-2 px-2">Punkte</th>
                  <th className="text-right py-2 px-2">BH</th>
                  <th className="text-right py-2 px-2">BH-C1</th>
                  <th className="text-right py-2 px-2">SB</th>
                </tr>
              </thead>
              <tbody>
                {standings.map((p: Participant, i: number) => (
                  <tr key={p.id} className="border-b">
                    <td className="py-2 px-2 text-center">{i + 1}</td>
                    <td className="py-2 px-2">{p.playerName || p.playerId}</td>
                    <td className="py-2 px-2 text-right">{p.startingRating}</td>
                    <td className="py-2 px-2 text-right font-medium">{p.points}</td>
                    <td className="py-2 px-2 text-right">{p.buchholz.toFixed(1)}</td>
                    <td className="py-2 px-2 text-right">
                      {(p as any).buchholzCut1?.toFixed(1) || '-'}
                    </td>
                    <td className="py-2 px-2 text-right">
                      {(p as any).sonnebornBerger?.toFixed(2) || '-'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

const tiebreakLabels: Record<string, string> = {
  buchholz: 'Buchholz',
  buchholz_cut1: 'Buchholz Cut 1',
  buchholz_cut2: 'Buchholz Cut 2',
  buchholz_median: 'Buchholz Median',
  sonneborn_berger: 'Sonneborn-Berger',
  direct_encounter: 'Direkter Vergleich',
  wins: 'Anzahl Siege',
  games_black: 'Partien mit Schwarz',
  wins_black: 'Siege mit Schwarz',
  progress: 'Progressive Wertung',
  aroc: 'AROC',
  koya: 'Koya-System',
  rating: 'Wertungszahl',
};
