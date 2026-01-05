'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import {
  AlertTriangle,
  CheckCircle2,
  Clock,
  Edit2,
  MessageSquare,
  PlayCircle,
  RefreshCw,
  Settings,
  Shield,
  Users,
  UserMinus,
  UserPlus,
  Shuffle,
  Trash2,
  ArrowLeftRight,
  Plus,
} from 'lucide-react';
import { api } from '@/lib/api';
import { toast } from 'sonner';
import { cn } from '@/lib/utils';

interface TournamentStatus {
  tournament: {
    id: string;
    name: string;
    status: string;
    rounds: number;
  };
  currentRound?: {
    id: string;
    roundNumber: number;
    status: string;
    actualStart?: string;
  };
  rounds: Array<{
    id: string;
    roundNumber: number;
    status: string;
  }>;
  participants: Array<{
    id: string;
    playerId: string;
    startRank: number;
    startingRating: number;
    points: number;
    isActive: boolean;
  }>;
  activeGames: Array<{
    game: {
      id: string;
      boardNumber: number;
      result?: string;
    };
    whitePlayer?: {
      participantId: string;
      name: string;
      rating: number;
    };
    blackPlayer?: {
      participantId: string;
      name: string;
      rating: number;
    };
    elapsedMinutes: number;
  }>;
  pendingResults: number;
  statistics: {
    totalParticipants: number;
    activeParticipants: number;
    withdrawnCount: number;
    completedRounds: number;
    averageRating: number;
  };
}

export default function ArbiterConsolePage() {
  const params = useParams();
  const tournamentId = params.tournamentId as string;

  const { data: status, isLoading, error, refetch } = useQuery<TournamentStatus>({
    queryKey: ['arbiter-status', tournamentId],
    queryFn: async (): Promise<TournamentStatus> => {
      const response = await api.get(`/arbiter/tournaments/${tournamentId}/status`);
      return response.data as TournamentStatus;
    },
    refetchInterval: 30000, // Refresh every 30 seconds
  });

  if (isLoading) {
    return (
      <div className="container mx-auto py-6 space-y-6">
        <Skeleton className="h-10 w-1/3" />
        <div className="grid gap-4 md:grid-cols-4">
          {[...Array(4)].map((_, i) => (
            <Skeleton key={i} className="h-32" />
          ))}
        </div>
      </div>
    );
  }

  if (error || !status) {
    return (
      <div className="container mx-auto py-6">
        <Card>
          <CardContent className="py-10 text-center">
            <AlertTriangle className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground">
              Fehler beim Laden der Turnierdaten
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-6 space-y-6">
      <div className="flex justify-between items-start">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <Shield className="h-6 w-6" />
            <h1 className="text-3xl font-bold tracking-tight">
              Schiedsrichterkonsole
            </h1>
          </div>
          <p className="text-muted-foreground">{status.tournament.name}</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Aktualisieren
          </Button>
          <CreateAnnouncementDialog tournamentId={tournamentId} />
        </div>
      </div>

      {/* Status Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Aktuelle Runde
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {status.currentRound?.roundNumber || '-'} / {status.tournament.rounds}
            </div>
            <p className="text-xs text-muted-foreground">
              {status.currentRound?.status === 'ongoing'
                ? 'Läuft'
                : status.currentRound?.status === 'paired'
                ? 'Gepaart'
                : 'Ausstehend'}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Teilnehmer
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {status.statistics.activeParticipants}
            </div>
            <p className="text-xs text-muted-foreground">
              {status.statistics.withdrawnCount} zurückgezogen
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Offene Ergebnisse
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{status.pendingResults}</div>
            <p className="text-xs text-muted-foreground">
              {status.activeGames.length} aktive Partien
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Durchschnittsrating
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {Math.round(status.statistics.averageRating)}
            </div>
            <p className="text-xs text-muted-foreground">Elo/DWZ</p>
          </CardContent>
        </Card>
      </div>

      <Tabs defaultValue="games" className="space-y-4">
        <TabsList>
          <TabsTrigger value="games">
            <Clock className="mr-2 h-4 w-4" />
            Aktive Partien
          </TabsTrigger>
          <TabsTrigger value="participants">
            <Users className="mr-2 h-4 w-4" />
            Teilnehmer
          </TabsTrigger>
          <TabsTrigger value="rounds">
            <PlayCircle className="mr-2 h-4 w-4" />
            Runden
          </TabsTrigger>
          <TabsTrigger value="pairings">
            <Shuffle className="mr-2 h-4 w-4" />
            Paarungen
          </TabsTrigger>
          <TabsTrigger value="actions">
            <Settings className="mr-2 h-4 w-4" />
            Aktionen
          </TabsTrigger>
        </TabsList>

        <TabsContent value="games">
          <ActiveGamesTab
            tournamentId={tournamentId}
            games={status.activeGames}
            roundStatus={status.currentRound?.status || 'pending'}
          />
        </TabsContent>

        <TabsContent value="participants">
          <ParticipantsTab
            tournamentId={tournamentId}
            participants={status.participants}
          />
        </TabsContent>

        <TabsContent value="rounds">
          <RoundControlTab
            tournamentId={tournamentId}
            rounds={status.rounds}
            currentRound={status.currentRound}
            pendingResults={status.pendingResults}
          />
        </TabsContent>

        <TabsContent value="pairings">
          <PairingsManagementTab
            tournamentId={tournamentId}
            rounds={status.rounds}
            participants={status.participants}
          />
        </TabsContent>

        <TabsContent value="actions">
          <ActionsTab
            tournamentId={tournamentId}
            currentRound={status.currentRound}
            participants={status.participants}
          />
        </TabsContent>
      </Tabs>
    </div>
  );
}

function ActiveGamesTab({
  tournamentId,
  games,
  roundStatus,
}: {
  tournamentId: string;
  games: TournamentStatus['activeGames'];
  roundStatus: string;
}) {
  if (games.length === 0) {
    return (
      <Card>
        <CardContent className="py-10 text-center">
          <CheckCircle2 className="mx-auto h-12 w-12 text-green-500 mb-4" />
          <p className="text-muted-foreground">
            Alle Partien beendet oder keine Runde aktiv
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Aktive Partien ({games.length})</CardTitle>
        <CardDescription>
          Klicken Sie auf eine Partie, um das Ergebnis einzutragen
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b">
                <th className="text-center py-2 px-2">Brett</th>
                <th className="text-left py-2 px-2">Weiß</th>
                <th className="text-center py-2 px-2">Status</th>
                <th className="text-right py-2 px-2">Schwarz</th>
                <th className="text-center py-2 px-2">Zeit</th>
                <th className="w-20"></th>
              </tr>
            </thead>
            <tbody>
              {games.map((game) => (
                <tr key={game.game.id} className="border-b">
                  <td className="text-center py-3 px-2 font-medium">
                    {game.game.boardNumber}
                  </td>
                  <td className="py-3 px-2">
                    {game.whitePlayer ? (
                      <div>
                        <span>{game.whitePlayer.name || game.whitePlayer.participantId}</span>
                        <span className="text-xs text-muted-foreground ml-2">
                          ({game.whitePlayer.rating})
                        </span>
                      </div>
                    ) : (
                      '-'
                    )}
                  </td>
                  <td className="text-center py-3 px-2">
                    {game.game.result ? (
                      <Badge>{game.game.result}</Badge>
                    ) : (
                      <Badge variant="outline">läuft</Badge>
                    )}
                  </td>
                  <td className="text-right py-3 px-2">
                    {game.blackPlayer ? (
                      <div>
                        <span className="text-xs text-muted-foreground mr-2">
                          ({game.blackPlayer.rating})
                        </span>
                        <span>{game.blackPlayer.name || game.blackPlayer.participantId}</span>
                      </div>
                    ) : (
                      <span className="italic text-muted-foreground">spielfrei</span>
                    )}
                  </td>
                  <td className="text-center py-3 px-2 text-muted-foreground">
                    {game.elapsedMinutes} min
                  </td>
                  <td className="py-3 px-2">
                    {!game.game.result && roundStatus === 'ongoing' && (
                      <ResultEntryDialog
                        tournamentId={tournamentId}
                        gameId={game.game.id}
                        boardNumber={game.game.boardNumber}
                        whiteName={game.whitePlayer?.name || 'Weiß'}
                        blackName={game.blackPlayer?.name || 'Schwarz'}
                      />
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
}

function ResultEntryDialog({
  tournamentId,
  gameId,
  boardNumber,
  whiteName,
  blackName,
}: {
  tournamentId: string;
  gameId: string;
  boardNumber: number;
  whiteName: string;
  blackName: string;
}) {
  const [open, setOpen] = useState(false);
  const [result, setResult] = useState('');
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: () =>
      api.put(`/tournaments/${tournamentId}/games/${gameId}/result`, { result }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['arbiter-status', tournamentId] });
      toast.success('Ergebnis eingetragen');
      setOpen(false);
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler beim Speichern');
    },
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <Edit2 className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Ergebnis eintragen</DialogTitle>
          <DialogDescription>
            Brett {boardNumber}: {whiteName} vs {blackName}
          </DialogDescription>
        </DialogHeader>
        <div className="grid grid-cols-3 gap-2 py-4">
          {['1-0', '1/2-1/2', '0-1'].map((r) => (
            <Button
              key={r}
              variant={result === r ? 'default' : 'outline'}
              onClick={() => setResult(r)}
              className="text-lg"
            >
              {r === '1/2-1/2' ? '½-½' : r}
            </Button>
          ))}
        </div>
        <div className="grid grid-cols-3 gap-2">
          {['1-0F', '0-0F', '0-1F'].map((r) => (
            <Button
              key={r}
              variant={result === r ? 'secondary' : 'ghost'}
              onClick={() => setResult(r)}
              size="sm"
            >
              {r} (k)
            </Button>
          ))}
        </div>
        <DialogFooter className="mt-4">
          <Button variant="outline" onClick={() => setOpen(false)}>
            Abbrechen
          </Button>
          <Button
            onClick={() => mutation.mutate()}
            disabled={!result || mutation.isPending}
          >
            Speichern
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ParticipantsTab({
  tournamentId,
  participants,
}: {
  tournamentId: string;
  participants: TournamentStatus['participants'];
}) {
  const queryClient = useQueryClient();
  const [withdrawing, setWithdrawing] = useState<string | null>(null);

  const withdrawMutation = useMutation({
    mutationFn: (participantId: string) =>
      api.post(`/arbiter/tournaments/${tournamentId}/withdraw-player`, {
        participantId,
        reason: 'Abmeldung durch Schiedsrichter',
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['arbiter-status', tournamentId] });
      toast.success('Spieler zurückgezogen');
      setWithdrawing(null);
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler beim Zurückziehen');
    },
  });

  const activeParticipants = participants.filter((p) => p.isActive);
  const withdrawnParticipants = participants.filter((p) => !p.isActive);

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Aktive Teilnehmer ({activeParticipants.length})</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b">
                  <th className="text-center py-2 px-2">Nr.</th>
                  <th className="text-left py-2 px-2">Spieler</th>
                  <th className="text-right py-2 px-2">Rating</th>
                  <th className="text-right py-2 px-2">Punkte</th>
                  <th className="w-20"></th>
                </tr>
              </thead>
              <tbody>
                {activeParticipants.map((p) => (
                  <tr key={p.id} className="border-b">
                    <td className="text-center py-2 px-2">{p.startRank}</td>
                    <td className="py-2 px-2">{p.playerId}</td>
                    <td className="text-right py-2 px-2">{p.startingRating}</td>
                    <td className="text-right py-2 px-2">{p.points}</td>
                    <td className="py-2 px-2">
                      <Dialog
                        open={withdrawing === p.id}
                        onOpenChange={(open) => setWithdrawing(open ? p.id : null)}
                      >
                        <DialogTrigger asChild>
                          <Button variant="ghost" size="sm">
                            <UserMinus className="h-4 w-4" />
                          </Button>
                        </DialogTrigger>
                        <DialogContent>
                          <DialogHeader>
                            <DialogTitle>Spieler zurückziehen?</DialogTitle>
                            <DialogDescription>
                              Der Spieler wird aus dem Turnier zurückgezogen.
                              Laufende Partien werden als Niederlage gewertet.
                            </DialogDescription>
                          </DialogHeader>
                          <DialogFooter>
                            <Button
                              variant="outline"
                              onClick={() => setWithdrawing(null)}
                            >
                              Abbrechen
                            </Button>
                            <Button
                              variant="destructive"
                              onClick={() => withdrawMutation.mutate(p.id)}
                              disabled={withdrawMutation.isPending}
                            >
                              Zurückziehen
                            </Button>
                          </DialogFooter>
                        </DialogContent>
                      </Dialog>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {withdrawnParticipants.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-muted-foreground">
              Zurückgezogene Spieler ({withdrawnParticipants.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b">
                    <th className="text-center py-2 px-2">Nr.</th>
                    <th className="text-left py-2 px-2">Spieler</th>
                    <th className="text-right py-2 px-2">Rating</th>
                    <th className="text-right py-2 px-2">Punkte</th>
                    <th className="w-20"></th>
                  </tr>
                </thead>
                <tbody>
                  {withdrawnParticipants.map((p) => (
                    <tr key={p.id} className="border-b opacity-60 hover:opacity-100">
                      <td className="text-center py-2 px-2">{p.startRank}</td>
                      <td className="py-2 px-2">{p.playerId}</td>
                      <td className="text-right py-2 px-2">{p.startingRating}</td>
                      <td className="text-right py-2 px-2">{p.points}</td>
                      <td className="py-2 px-2">
                        <ReactivatePlayerDialog
                          tournamentId={tournamentId}
                          participantId={p.id}
                          playerName={p.playerId}
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function ActionsTab({
  tournamentId,
  currentRound,
  participants,
}: {
  tournamentId: string;
  currentRound?: TournamentStatus['currentRound'];
  participants: TournamentStatus['participants'];
}) {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle>Spielfrei vergeben</CardTitle>
          <CardDescription>
            Vergeben Sie einem Spieler ein Spielfrei für die aktuelle Runde
          </CardDescription>
        </CardHeader>
        <CardContent>
          <AssignByeForm
            tournamentId={tournamentId}
            roundNumber={currentRound?.roundNumber || 1}
            participants={participants.filter((p) => p.isActive)}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Ergebnis korrigieren</CardTitle>
          <CardDescription>
            Korrigieren Sie ein fehlerhaft eingetragenes Ergebnis
          </CardDescription>
        </CardHeader>
        <CardContent>
          <CorrectResultForm tournamentId={tournamentId} />
        </CardContent>
      </Card>
    </div>
  );
}

function RoundControlTab({
  tournamentId,
  rounds,
  currentRound,
  pendingResults,
}: {
  tournamentId: string;
  rounds: TournamentStatus['rounds'];
  currentRound?: TournamentStatus['currentRound'];
  pendingResults: number;
}) {
  const [selectedRound, setSelectedRound] = useState<string>(
    currentRound?.roundNumber.toString() || '1'
  );
  const queryClient = useQueryClient();

  const selectedRoundData = rounds.find(
    (r) => r.roundNumber === parseInt(selectedRound)
  );

  const generatePairingsMutation = useMutation({
    mutationFn: () =>
      api.post(`/arbiter/tournaments/${tournamentId}/rounds/${selectedRound}/pair`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['arbiter-status', tournamentId] });
      toast.success('Paarungen erstellt');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler beim Erstellen der Paarungen');
    },
  });

  const startRoundMutation = useMutation({
    mutationFn: () =>
      api.post(`/arbiter/tournaments/${tournamentId}/rounds/${selectedRound}/start`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['arbiter-status', tournamentId] });
      toast.success('Runde gestartet');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler beim Starten der Runde');
    },
  });

  const completeRoundMutation = useMutation({
    mutationFn: () =>
      api.post(`/arbiter/tournaments/${tournamentId}/rounds/${selectedRound}/complete`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['arbiter-status', tournamentId] });
      toast.success('Runde beendet');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler beim Beenden der Runde');
    },
  });

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'pending':
        return <Badge variant="outline">Ausstehend</Badge>;
      case 'paired':
        return <Badge variant="secondary">Gepaart</Badge>;
      case 'ongoing':
        return <Badge>Läuft</Badge>;
      case 'completed':
        return <Badge variant="default" className="bg-green-600">Beendet</Badge>;
      default:
        return <Badge variant="outline">{status}</Badge>;
    }
  };

  const isCurrentRound = currentRound?.roundNumber === parseInt(selectedRound);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Rundensteuerung</CardTitle>
        <CardDescription>
          Verwalten Sie den Ablauf der Turnierrunden
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="flex items-center gap-4">
          <div className="space-y-2">
            <Label>Runde auswählen</Label>
            <Select value={selectedRound} onValueChange={setSelectedRound}>
              <SelectTrigger className="w-[180px]">
                <SelectValue placeholder="Runde wählen" />
              </SelectTrigger>
              <SelectContent>
                {rounds.map((r) => (
                  <SelectItem key={r.id} value={r.roundNumber.toString()}>
                    Runde {r.roundNumber}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>Status</Label>
            <div className="h-10 flex items-center">
              {selectedRoundData && getStatusBadge(selectedRoundData.status)}
            </div>
          </div>
        </div>

        {selectedRoundData && (
          <div className="flex flex-wrap gap-3">
            {selectedRoundData.status === 'pending' && (
              <Button
                onClick={() => generatePairingsMutation.mutate()}
                disabled={generatePairingsMutation.isPending}
              >
                <Shuffle className="mr-2 h-4 w-4" />
                Paarungen erstellen
              </Button>
            )}

            {selectedRoundData.status === 'paired' && (
              <Button
                onClick={() => startRoundMutation.mutate()}
                disabled={startRoundMutation.isPending}
              >
                <PlayCircle className="mr-2 h-4 w-4" />
                Runde starten
              </Button>
            )}

            {selectedRoundData.status === 'ongoing' && (
              <Button
                onClick={() => completeRoundMutation.mutate()}
                disabled={completeRoundMutation.isPending || (isCurrentRound && pendingResults > 0)}
                title={isCurrentRound && pendingResults > 0 ? `Noch ${pendingResults} offene Ergebnisse` : ''}
              >
                <CheckCircle2 className="mr-2 h-4 w-4" />
                Runde beenden
                {isCurrentRound && pendingResults > 0 && (
                  <span className="ml-2 text-xs">({pendingResults} offen)</span>
                )}
              </Button>
            )}

            {selectedRoundData.status === 'completed' && (
              <p className="text-sm text-muted-foreground flex items-center">
                <CheckCircle2 className="mr-2 h-4 w-4 text-green-600" />
                Diese Runde ist abgeschlossen.
              </p>
            )}
          </div>
        )}

        <div className="border-t pt-4">
          <h4 className="font-medium mb-2">Rundenübersicht</h4>
          <div className="flex flex-wrap gap-2">
            {rounds.map((r) => (
              <Button
                key={r.id}
                variant={r.roundNumber === parseInt(selectedRound) ? 'default' : 'outline'}
                size="sm"
                onClick={() => setSelectedRound(r.roundNumber.toString())}
                className={cn(
                  r.status === 'completed' && 'border-green-600',
                  r.status === 'ongoing' && 'border-blue-600'
                )}
              >
                R{r.roundNumber}
                {r.status === 'completed' && <CheckCircle2 className="ml-1 h-3 w-3 text-green-600" />}
              </Button>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

interface PairingGame {
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

function PairingsManagementTab({
  tournamentId,
  rounds,
  participants,
}: {
  tournamentId: string;
  rounds: TournamentStatus['rounds'];
  participants: TournamentStatus['participants'];
}) {
  const [selectedRound, setSelectedRound] = useState<string>(
    rounds.length > 0 ? rounds[rounds.length - 1].roundNumber.toString() : '1'
  );

  const selectedRoundData = rounds.find(
    (r) => r.roundNumber === parseInt(selectedRound)
  );

  const { data: pairingsData, isLoading } = useQuery<{ games: PairingGame[] }>({
    queryKey: ['pairings', tournamentId, selectedRound],
    queryFn: async () => {
      const response = await api.get(
        `/tournaments/${tournamentId}/rounds/${selectedRound}/pairings`
      );
      return response.data as { games: PairingGame[] };
    },
    enabled: !!selectedRound && rounds.length > 0,
  });

  const pairings = pairingsData?.games || [];
  const canEdit = selectedRoundData?.status === 'paired';

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Paarungsverwaltung</CardTitle>
              <CardDescription>
                Bearbeiten Sie die Paarungen der ausgewählten Runde
              </CardDescription>
            </div>
            <div className="flex gap-2">
              <Select value={selectedRound} onValueChange={setSelectedRound}>
                <SelectTrigger className="w-[150px]">
                  <SelectValue placeholder="Runde" />
                </SelectTrigger>
                <SelectContent>
                  {rounds.map((r) => (
                    <SelectItem key={r.id} value={r.roundNumber.toString()}>
                      Runde {r.roundNumber}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {canEdit && (
                <>
                  <SwapPlayersDialog
                    tournamentId={tournamentId}
                    roundNumber={parseInt(selectedRound)}
                    pairings={pairings}
                  />
                  <CreateManualPairingDialog
                    tournamentId={tournamentId}
                    roundNumber={parseInt(selectedRound)}
                    participants={participants}
                    existingPairings={pairings}
                  />
                </>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              {[...Array(5)].map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : pairings.length === 0 ? (
            <div className="text-center py-10">
              <p className="text-muted-foreground">
                Keine Paarungen für diese Runde vorhanden
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b">
                    <th className="text-center py-2 px-2">Brett</th>
                    <th className="text-left py-2 px-2">Weiß</th>
                    <th className="text-center py-2 px-2">Ergebnis</th>
                    <th className="text-right py-2 px-2">Schwarz</th>
                    {canEdit && <th className="w-24 text-center">Aktionen</th>}
                  </tr>
                </thead>
                <tbody>
                  {pairings.map((game) => (
                    <tr key={game.id} className="border-b">
                      <td className="text-center py-2 px-2 font-medium">
                        {game.boardNumber}
                      </td>
                      <td className="py-2 px-2">
                        {game.whiteName || game.whiteParticipantId || '-'}
                        {game.whiteRating && (
                          <span className="text-xs text-muted-foreground ml-1">
                            ({game.whiteRating})
                          </span>
                        )}
                      </td>
                      <td className="text-center py-2 px-2">
                        {game.result ? (
                          <Badge>{game.result}</Badge>
                        ) : (
                          <Badge variant="outline">-</Badge>
                        )}
                      </td>
                      <td className="text-right py-2 px-2">
                        {game.blackParticipantId ? (
                          <>
                            {game.blackRating && (
                              <span className="text-xs text-muted-foreground mr-1">
                                ({game.blackRating})
                              </span>
                            )}
                            {game.blackName || game.blackParticipantId}
                          </>
                        ) : (
                          <span className="italic text-muted-foreground">
                            spielfrei
                          </span>
                        )}
                      </td>
                      {canEdit && (
                        <td className="py-2 px-2">
                          <div className="flex justify-center gap-1">
                            {game.blackParticipantId && !game.result && (
                              <ChangeColorDialog
                                tournamentId={tournamentId}
                                gameId={game.id}
                                whiteName={game.whiteName || 'Weiß'}
                                blackName={game.blackName || 'Schwarz'}
                              />
                            )}
                            {!game.result && (
                              <DeletePairingDialog
                                tournamentId={tournamentId}
                                gameId={game.id}
                                boardNumber={game.boardNumber}
                              />
                            )}
                          </div>
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {!canEdit && selectedRoundData && (
        <p className="text-sm text-muted-foreground text-center">
          Paarungen können nur bearbeitet werden, wenn die Runde den Status
          &quot;Gepaart&quot; hat. Aktueller Status:{' '}
          <Badge variant="outline">{selectedRoundData.status}</Badge>
        </p>
      )}
    </div>
  );
}

function ChangeColorDialog({
  tournamentId,
  gameId,
  whiteName,
  blackName,
}: {
  tournamentId: string;
  gameId: string;
  whiteName: string;
  blackName: string;
}) {
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState('');
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: () =>
      api.post(`/arbiter/tournaments/${tournamentId}/change-color`, {
        gameId,
        reason: reason || 'Farbwechsel durch Schiedsrichter',
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pairings', tournamentId] });
      queryClient.invalidateQueries({ queryKey: ['arbiter-status', tournamentId] });
      toast.success('Farben getauscht');
      setOpen(false);
      setReason('');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler beim Farbwechsel');
    },
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="ghost" size="sm" title="Farben tauschen">
          <ArrowLeftRight className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Farben tauschen</DialogTitle>
          <DialogDescription>
            {whiteName} und {blackName} tauschen die Farben.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label>Begründung (optional)</Label>
            <Textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Grund für den Farbwechsel"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Abbrechen
          </Button>
          <Button onClick={() => mutation.mutate()} disabled={mutation.isPending}>
            Farben tauschen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function DeletePairingDialog({
  tournamentId,
  gameId,
  boardNumber,
}: {
  tournamentId: string;
  gameId: string;
  boardNumber: number;
}) {
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState('');
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: () =>
      api.delete(
        `/arbiter/tournaments/${tournamentId}/pairings/${gameId}?reason=${encodeURIComponent(reason || 'Paarung gelöscht durch Schiedsrichter')}`
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pairings', tournamentId] });
      queryClient.invalidateQueries({ queryKey: ['arbiter-status', tournamentId] });
      toast.success('Paarung gelöscht');
      setOpen(false);
      setReason('');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler beim Löschen');
    },
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="ghost" size="sm" title="Paarung löschen">
          <Trash2 className="h-4 w-4 text-destructive" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Paarung löschen</DialogTitle>
          <DialogDescription>
            Die Paarung auf Brett {boardNumber} wird gelöscht.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label>Begründung (optional)</Label>
            <Textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Grund für das Löschen"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Abbrechen
          </Button>
          <Button
            variant="destructive"
            onClick={() => mutation.mutate()}
            disabled={mutation.isPending}
          >
            Löschen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function SwapPlayersDialog({
  tournamentId,
  roundNumber,
  pairings,
}: {
  tournamentId: string;
  roundNumber: number;
  pairings: PairingGame[];
}) {
  const [open, setOpen] = useState(false);
  const [game1Id, setGame1Id] = useState('');
  const [player1Id, setPlayer1Id] = useState('');
  const [game2Id, setGame2Id] = useState('');
  const [player2Id, setPlayer2Id] = useState('');
  const [reason, setReason] = useState('');
  const queryClient = useQueryClient();

  const game1 = pairings.find((p) => p.id === game1Id);
  const game2 = pairings.find((p) => p.id === game2Id);

  const game1Players = game1
    ? [
        { id: game1.whiteParticipantId, name: game1.whiteName || 'Weiß', color: 'Weiß' },
        { id: game1.blackParticipantId, name: game1.blackName || 'Schwarz', color: 'Schwarz' },
      ].filter((p) => p.id)
    : [];

  const game2Players = game2
    ? [
        { id: game2.whiteParticipantId, name: game2.whiteName || 'Weiß', color: 'Weiß' },
        { id: game2.blackParticipantId, name: game2.blackName || 'Schwarz', color: 'Schwarz' },
      ].filter((p) => p.id)
    : [];

  const mutation = useMutation({
    mutationFn: () =>
      api.post(`/arbiter/tournaments/${tournamentId}/swap-pairing`, {
        roundNumber,
        game1Id,
        player1Id,
        game2Id,
        player2Id,
        reason: reason || 'Spielertausch durch Schiedsrichter',
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pairings', tournamentId] });
      queryClient.invalidateQueries({ queryKey: ['arbiter-status', tournamentId] });
      toast.success('Spieler getauscht');
      setOpen(false);
      resetForm();
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler beim Tauschen');
    },
  });

  const resetForm = () => {
    setGame1Id('');
    setPlayer1Id('');
    setGame2Id('');
    setPlayer2Id('');
    setReason('');
  };

  const availableGames = pairings.filter(
    (p) => !p.result && p.blackParticipantId
  );

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <ArrowLeftRight className="mr-2 h-4 w-4" />
          Spieler tauschen
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Spieler zwischen Paarungen tauschen</DialogTitle>
          <DialogDescription>
            Wählen Sie zwei Partien und jeweils einen Spieler zum Tauschen.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Partie 1</Label>
              <Select value={game1Id} onValueChange={(v) => { setGame1Id(v); setPlayer1Id(''); }}>
                <SelectTrigger>
                  <SelectValue placeholder="Partie wählen" />
                </SelectTrigger>
                <SelectContent>
                  {availableGames
                    .filter((g) => g.id !== game2Id)
                    .map((g) => (
                      <SelectItem key={g.id} value={g.id}>
                        Brett {g.boardNumber}
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
              {game1Id && (
                <Select value={player1Id} onValueChange={setPlayer1Id}>
                  <SelectTrigger>
                    <SelectValue placeholder="Spieler wählen" />
                  </SelectTrigger>
                  <SelectContent>
                    {game1Players.map((p) => (
                      <SelectItem key={p.id} value={p.id!}>
                        {p.name} ({p.color})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </div>
            <div className="space-y-2">
              <Label>Partie 2</Label>
              <Select value={game2Id} onValueChange={(v) => { setGame2Id(v); setPlayer2Id(''); }}>
                <SelectTrigger>
                  <SelectValue placeholder="Partie wählen" />
                </SelectTrigger>
                <SelectContent>
                  {availableGames
                    .filter((g) => g.id !== game1Id)
                    .map((g) => (
                      <SelectItem key={g.id} value={g.id}>
                        Brett {g.boardNumber}
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
              {game2Id && (
                <Select value={player2Id} onValueChange={setPlayer2Id}>
                  <SelectTrigger>
                    <SelectValue placeholder="Spieler wählen" />
                  </SelectTrigger>
                  <SelectContent>
                    {game2Players.map((p) => (
                      <SelectItem key={p.id} value={p.id!}>
                        {p.name} ({p.color})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </div>
          </div>
          <div className="space-y-2">
            <Label>Begründung (optional)</Label>
            <Textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Grund für den Spielertausch"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Abbrechen
          </Button>
          <Button
            onClick={() => mutation.mutate()}
            disabled={!game1Id || !player1Id || !game2Id || !player2Id || mutation.isPending}
          >
            Spieler tauschen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function CreateManualPairingDialog({
  tournamentId,
  roundNumber,
  participants,
  existingPairings,
}: {
  tournamentId: string;
  roundNumber: number;
  participants: TournamentStatus['participants'];
  existingPairings: PairingGame[];
}) {
  const [open, setOpen] = useState(false);
  const [whiteParticipantId, setWhiteParticipantId] = useState('');
  const [blackParticipantId, setBlackParticipantId] = useState('');
  const [boardNumber, setBoardNumber] = useState('');
  const queryClient = useQueryClient();

  // Find participants already paired in this round
  const pairedParticipantIds = new Set<string>();
  existingPairings.forEach((p) => {
    if (p.whiteParticipantId) pairedParticipantIds.add(p.whiteParticipantId);
    if (p.blackParticipantId) pairedParticipantIds.add(p.blackParticipantId);
  });

  // Active participants not yet paired
  const availableParticipants = participants.filter(
    (p) => p.isActive && !pairedParticipantIds.has(p.id)
  );

  const mutation = useMutation({
    mutationFn: () =>
      api.post(`/arbiter/tournaments/${tournamentId}/create-pairing`, {
        roundNumber,
        whiteParticipantId,
        blackParticipantId: blackParticipantId === 'BYE' ? null : blackParticipantId,
        boardNumber: boardNumber ? parseInt(boardNumber) : undefined,
      }),
    onSuccess: (response: any) => {
      queryClient.invalidateQueries({ queryKey: ['pairings', tournamentId] });
      queryClient.invalidateQueries({ queryKey: ['arbiter-status', tournamentId] });

      // Show warnings if any
      if (response.data?.warnings?.length > 0) {
        response.data.warnings.forEach((w: { message: string }) => {
          toast.warning(w.message);
        });
      }

      toast.success('Paarung erstellt');
      setOpen(false);
      resetForm();
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler beim Erstellen');
    },
  });

  const resetForm = () => {
    setWhiteParticipantId('');
    setBlackParticipantId('');
    setBoardNumber('');
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <Plus className="mr-2 h-4 w-4" />
          Neue Paarung
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Manuelle Paarung erstellen</DialogTitle>
          <DialogDescription>
            Erstellen Sie eine neue Paarung für Runde {roundNumber}.
            {availableParticipants.length === 0 && (
              <span className="text-destructive block mt-1">
                Alle Teilnehmer sind bereits gepaart.
              </span>
            )}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label>Weiß</Label>
            <Select value={whiteParticipantId} onValueChange={setWhiteParticipantId}>
              <SelectTrigger>
                <SelectValue placeholder="Spieler mit Weiß" />
              </SelectTrigger>
              <SelectContent>
                {availableParticipants
                  .filter((p) => p.id !== blackParticipantId)
                  .map((p) => (
                    <SelectItem key={p.id} value={p.id}>
                      {p.startRank}. {p.playerId} ({p.startingRating})
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>Schwarz</Label>
            <Select value={blackParticipantId} onValueChange={setBlackParticipantId}>
              <SelectTrigger>
                <SelectValue placeholder="Spieler mit Schwarz oder Spielfrei" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="BYE">
                  <span className="italic">Spielfrei (BYE)</span>
                </SelectItem>
                {availableParticipants
                  .filter((p) => p.id !== whiteParticipantId)
                  .map((p) => (
                    <SelectItem key={p.id} value={p.id}>
                      {p.startRank}. {p.playerId} ({p.startingRating})
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>Brettnummer (optional)</Label>
            <Input
              type="number"
              min="1"
              value={boardNumber}
              onChange={(e) => setBoardNumber(e.target.value)}
              placeholder="Automatisch zuweisen"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Abbrechen
          </Button>
          <Button
            onClick={() => mutation.mutate()}
            disabled={!whiteParticipantId || !blackParticipantId || mutation.isPending}
          >
            Paarung erstellen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function AssignByeForm({
  tournamentId,
  roundNumber,
  participants,
}: {
  tournamentId: string;
  roundNumber: number;
  participants: TournamentStatus['participants'];
}) {
  const [participantId, setParticipantId] = useState('');
  const [byeType, setByeType] = useState('half');
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: () =>
      api.post(`/arbiter/tournaments/${tournamentId}/assign-bye`, {
        participantId,
        roundNumber,
        byeType,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['arbiter-status', tournamentId] });
      toast.success('Spielfrei vergeben');
      setParticipantId('');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler beim Vergeben');
    },
  });

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label>Spieler</Label>
        <Select value={participantId} onValueChange={setParticipantId}>
          <SelectTrigger>
            <SelectValue placeholder="Spieler auswählen" />
          </SelectTrigger>
          <SelectContent>
            {participants.map((p) => (
              <SelectItem key={p.id} value={p.id}>
                {p.startRank}. {p.playerId} ({p.startingRating})
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-2">
        <Label>Typ</Label>
        <Select value={byeType} onValueChange={setByeType}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="half">Halber Punkt (0.5)</SelectItem>
            <SelectItem value="full">Voller Punkt (1.0)</SelectItem>
            <SelectItem value="zero">Null Punkte (0)</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <Button
        onClick={() => mutation.mutate()}
        disabled={!participantId || mutation.isPending}
        className="w-full"
      >
        Spielfrei vergeben
      </Button>
    </div>
  );
}

function CorrectResultForm({ tournamentId }: { tournamentId: string }) {
  const [gameId, setGameId] = useState('');
  const [newResult, setNewResult] = useState('');
  const [reason, setReason] = useState('');
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: () =>
      api.post(`/arbiter/tournaments/${tournamentId}/correct-result`, {
        gameId,
        newResult,
        reason,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['arbiter-status', tournamentId] });
      toast.success('Ergebnis korrigiert');
      setGameId('');
      setNewResult('');
      setReason('');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler bei Korrektur');
    },
  });

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label>Partie-ID</Label>
        <Input
          value={gameId}
          onChange={(e) => setGameId(e.target.value)}
          placeholder="UUID der Partie"
        />
      </div>
      <div className="space-y-2">
        <Label>Neues Ergebnis</Label>
        <Select value={newResult} onValueChange={setNewResult}>
          <SelectTrigger>
            <SelectValue placeholder="Ergebnis wählen" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="1-0">1 - 0</SelectItem>
            <SelectItem value="0-1">0 - 1</SelectItem>
            <SelectItem value="1/2-1/2">½ - ½</SelectItem>
            <SelectItem value="1-0F">1 - 0 (kampflos)</SelectItem>
            <SelectItem value="0-1F">0 - 1 (kampflos)</SelectItem>
            <SelectItem value="0-0F">0 - 0</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-2">
        <Label>Begründung</Label>
        <Textarea
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="Grund für die Korrektur"
        />
      </div>
      <Button
        onClick={() => mutation.mutate()}
        disabled={!gameId || !newResult || !reason || mutation.isPending}
        className="w-full"
      >
        Ergebnis korrigieren
      </Button>
    </div>
  );
}

function CreateAnnouncementDialog({ tournamentId }: { tournamentId: string }) {
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState('');
  const [message, setMessage] = useState('');
  const [priority, setPriority] = useState('info');

  const mutation = useMutation({
    mutationFn: () =>
      api.post(`/arbiter/tournaments/${tournamentId}/announcements`, {
        title,
        message,
        priority,
      }),
    onSuccess: () => {
      toast.success('Ankündigung erstellt');
      setOpen(false);
      setTitle('');
      setMessage('');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler beim Erstellen');
    },
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <MessageSquare className="mr-2 h-4 w-4" />
          Ankündigung
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Neue Ankündigung</DialogTitle>
          <DialogDescription>
            Erstellen Sie eine Ankündigung für alle Teilnehmer
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label>Titel</Label>
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="z.B. Rundenbeginn"
            />
          </div>
          <div className="space-y-2">
            <Label>Nachricht</Label>
            <Textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="Nachricht eingeben..."
            />
          </div>
          <div className="space-y-2">
            <Label>Priorität</Label>
            <Select value={priority} onValueChange={setPriority}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="info">Information</SelectItem>
                <SelectItem value="warning">Warnung</SelectItem>
                <SelectItem value="urgent">Dringend</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Abbrechen
          </Button>
          <Button
            onClick={() => mutation.mutate()}
            disabled={!title || !message || mutation.isPending}
          >
            Veröffentlichen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ReactivatePlayerDialog({
  tournamentId,
  participantId,
  playerName,
}: {
  tournamentId: string;
  participantId: string;
  playerName: string;
}) {
  const [open, setOpen] = useState(false);
  const [missedRoundPoints, setMissedRoundPoints] = useState('0');
  const [reason, setReason] = useState('');
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: () =>
      api.post(`/arbiter/tournaments/${tournamentId}/reactivate-player`, {
        participantId,
        missedRoundPoints: parseFloat(missedRoundPoints) || 0,
        reason: reason || 'Reaktivierung durch Schiedsrichter',
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['arbiter-status', tournamentId] });
      toast.success('Spieler reaktiviert');
      setOpen(false);
      setMissedRoundPoints('0');
      setReason('');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler bei Reaktivierung');
    },
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="ghost" size="sm" title="Spieler reaktivieren">
          <UserPlus className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Spieler reaktivieren</DialogTitle>
          <DialogDescription>
            {playerName} wird wieder am Turnier teilnehmen.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label>Punkte für verpasste Runden</Label>
            <Input
              type="number"
              step="0.5"
              min="0"
              value={missedRoundPoints}
              onChange={(e) => setMissedRoundPoints(e.target.value)}
              placeholder="0"
            />
            <p className="text-xs text-muted-foreground">
              Optional: Punkte, die für verpasste Runden gutgeschrieben werden
            </p>
          </div>
          <div className="space-y-2">
            <Label>Begründung</Label>
            <Textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Grund für die Reaktivierung (optional)"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Abbrechen
          </Button>
          <Button
            onClick={() => mutation.mutate()}
            disabled={mutation.isPending}
          >
            Reaktivieren
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
