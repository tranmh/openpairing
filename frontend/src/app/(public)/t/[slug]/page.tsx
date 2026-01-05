'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import { useQuery, useMutation } from '@tanstack/react-query';
import { format } from 'date-fns';
import { de } from 'date-fns/locale';
import {
  CalendarIcon,
  MapPinIcon,
  UsersIcon,
  ClockIcon,
  CheckCircleIcon,
  TrophyIcon,
  ListIcon,
  GridIcon,
  UsersRoundIcon,
  CircleDot,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { api } from '@/lib/api';
import { useAuthStore } from '@/store/auth';
import { PublicStandingsTab } from '@/components/public/PublicStandingsTab';
import { PublicPairingsTab } from '@/components/public/PublicPairingsTab';
import { PublicCrosstableTab } from '@/components/public/PublicCrosstableTab';
import { PublicParticipantsTab } from '@/components/public/PublicParticipantsTab';
import { SeatingView } from '@/components/tournament/SeatingView';

interface TournamentPublicInfo {
  id: string;
  name: string;
  description?: string;
  slug: string;
  type: string;
  startDate: string;
  endDate?: string;
  venueName?: string;
  venueCity?: string;
  venueCountry?: string;
  maxPlayers?: number;
  participantCount: number;
  status: string;
  registrationOpens?: string;
  registrationCloses?: string;
  entryFeeCents: number;
  currency: string;
  currentRound?: number;
  totalRounds?: number;
}

interface RegistrationSettings {
  preRegistration: {
    enabled: boolean;
    requiresApproval: boolean;
    allowGuests: boolean;
  };
  checkIn: {
    enabled: boolean;
    methods: string[];
    windowOpenMinutes: number;
    windowCloseMinutes: number;
  };
  access: {
    mode: string;
  };
}

interface PreRegisterResponse {
  id: string;
  status: string;
  checkInCode?: string;
  requiresApproval: boolean;
  message: string;
}

interface TournamentDetails {
  id: string;
  currentRound: number;
  rounds: number;
  status: string;
}

const statusColors: Record<string, string> = {
  draft: 'bg-gray-500',
  published: 'bg-blue-500',
  registration: 'bg-green-500',
  ongoing: 'bg-yellow-500',
  completed: 'bg-gray-700',
  cancelled: 'bg-red-500',
};

const statusLabels: Record<string, string> = {
  draft: 'Entwurf',
  published: 'Veröffentlicht',
  registration: 'Anmeldung offen',
  ongoing: 'Laufend',
  completed: 'Beendet',
  cancelled: 'Abgesagt',
};

export default function PublicTournamentPage() {
  const params = useParams();
  const slug = params.slug as string;
  const { isAuthenticated, user } = useAuthStore();

  const [formData, setFormData] = useState({
    firstName: '',
    lastName: '',
    email: '',
    federationId: '',
    federationType: 'dsb',
    inviteToken: '',
  });
  const [registrationResult, setRegistrationResult] = useState<PreRegisterResponse | null>(null);

  // Fetch tournament data - use direct fetch for public endpoint (not under /api/v1)
  const { data, isLoading, error } = useQuery({
    queryKey: ['public-tournament', slug],
    queryFn: async () => {
      const response = await fetch(`/api/public/tournaments/${slug}`);
      if (!response.ok) {
        throw new Error('Tournament not found');
      }
      return response.json() as Promise<{
        tournament: TournamentPublicInfo;
        registrationSettings: RegistrationSettings;
        isCheckInOpen: boolean;
      }>;
    },
  });

  // Fetch tournament details (for rounds info) - only after we have tournament ID
  const tournamentId = data?.tournament?.id;
  const { data: tournamentDetails } = useQuery({
    queryKey: ['tournament-details', tournamentId],
    queryFn: async () => {
      const response = await api.get<TournamentDetails>(`/tournaments/${tournamentId}`);
      return response.data;
    },
    enabled: !!tournamentId,
  });

  // Pre-registration mutation - use direct fetch for public endpoint
  const preRegisterMutation = useMutation({
    mutationFn: async (formDataParam: typeof formData) => {
      const response = await fetch(`/api/public/tournaments/${slug}/pre-register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: formDataParam.email,
          firstName: formDataParam.firstName,
          lastName: formDataParam.lastName,
          federationId: formDataParam.federationId || undefined,
          federationType: formDataParam.federationType || undefined,
          inviteToken: formDataParam.inviteToken || undefined,
        }),
      });
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Registration failed');
      }
      return response.json() as Promise<PreRegisterResponse>;
    },
    onSuccess: (result) => {
      if (result) {
        setRegistrationResult(result);
      }
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    preRegisterMutation.mutate(formData);
  };

  if (isLoading) {
    return (
      <div className="container mx-auto flex min-h-[50vh] items-center justify-center px-4">
        <div className="animate-pulse text-muted-foreground">Laden...</div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="container mx-auto px-4 py-12">
        <Alert variant="destructive">
          <AlertTitle>Turnier nicht gefunden</AlertTitle>
          <AlertDescription>
            Das angeforderte Turnier existiert nicht oder ist nicht mehr verfügbar.
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  const { tournament, registrationSettings, isCheckInOpen } = data;
  const isRegistrationOpen =
    registrationSettings.preRegistration.enabled &&
    (tournament.status === 'published' || tournament.status === 'registration');
  const isFull = tournament.maxPlayers ? tournament.participantCount >= tournament.maxPlayers : false;
  const requiresInvite = registrationSettings.access.mode === 'invite_only';

  // Determine which tabs to show based on tournament status
  const hasStarted = tournament.status === 'ongoing' || tournament.status === 'completed';
  const currentRound = tournamentDetails?.currentRound || tournament.currentRound || 1;
  const totalRounds = tournamentDetails?.rounds || tournament.totalRounds || 7;

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mx-auto max-w-5xl">
        {/* Tournament Header */}
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-2">
            <Badge className={statusColors[tournament.status]}>
              {statusLabels[tournament.status] || tournament.status}
            </Badge>
            {isCheckInOpen && (
              <Badge variant="outline" className="border-green-500 text-green-600">
                Check-in offen
              </Badge>
            )}
            {hasStarted && (
              <Badge variant="outline">
                Runde {currentRound} / {totalRounds}
              </Badge>
            )}
          </div>
          <h1 className="text-3xl font-bold">{tournament.name}</h1>
          {tournament.description && (
            <p className="mt-2 text-muted-foreground">{tournament.description}</p>
          )}
        </div>

        {/* Tabs */}
        <Tabs defaultValue="info" className="space-y-6">
          <TabsList className={`grid w-full ${tournament.type === 'round_robin' ? 'grid-cols-6' : 'grid-cols-5'}`}>
            <TabsTrigger value="info" className="gap-2">
              <CalendarIcon className="h-4 w-4 hidden sm:inline" />
              Info
            </TabsTrigger>
            <TabsTrigger value="standings" className="gap-2" disabled={!hasStarted}>
              <TrophyIcon className="h-4 w-4 hidden sm:inline" />
              Rangliste
            </TabsTrigger>
            <TabsTrigger value="pairings" className="gap-2" disabled={!hasStarted}>
              <ListIcon className="h-4 w-4 hidden sm:inline" />
              Paarungen
            </TabsTrigger>
            {tournament.type === 'round_robin' && (
              <TabsTrigger value="seating" className="gap-2" disabled={!hasStarted}>
                <CircleDot className="h-4 w-4 hidden sm:inline" />
                Sitzplan
              </TabsTrigger>
            )}
            <TabsTrigger value="crosstable" className="gap-2" disabled={!hasStarted}>
              <GridIcon className="h-4 w-4 hidden sm:inline" />
              Kreuztabelle
            </TabsTrigger>
            <TabsTrigger value="participants" className="gap-2">
              <UsersRoundIcon className="h-4 w-4 hidden sm:inline" />
              Teilnehmer
            </TabsTrigger>
          </TabsList>

          {/* Info Tab */}
          <TabsContent value="info">
            <div className="grid gap-6 md:grid-cols-3">
              {/* Tournament Info */}
              <div className="md:col-span-2 space-y-6">
                {/* Details Card */}
                <Card>
                  <CardHeader>
                    <CardTitle>Turnierinformationen</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="flex items-center gap-3">
                      <CalendarIcon className="h-5 w-5 text-muted-foreground" />
                      <div>
                        <p className="font-medium">
                          {format(new Date(tournament.startDate), 'EEEE, d. MMMM yyyy', { locale: de })}
                        </p>
                        {tournament.endDate && tournament.endDate !== tournament.startDate && (
                          <p className="text-sm text-muted-foreground">
                            bis {format(new Date(tournament.endDate), 'd. MMMM yyyy', { locale: de })}
                          </p>
                        )}
                      </div>
                    </div>

                    {(tournament.venueName || tournament.venueCity) && (
                      <div className="flex items-center gap-3">
                        <MapPinIcon className="h-5 w-5 text-muted-foreground" />
                        <div>
                          {tournament.venueName && <p className="font-medium">{tournament.venueName}</p>}
                          {tournament.venueCity && (
                            <p className="text-sm text-muted-foreground">
                              {tournament.venueCity}
                              {tournament.venueCountry && `, ${tournament.venueCountry}`}
                            </p>
                          )}
                        </div>
                      </div>
                    )}

                    <div className="flex items-center gap-3">
                      <UsersIcon className="h-5 w-5 text-muted-foreground" />
                      <div>
                        <p className="font-medium">
                          {tournament.participantCount} Teilnehmer
                          {tournament.maxPlayers && ` / ${tournament.maxPlayers} max`}
                        </p>
                        {isFull && <p className="text-sm text-orange-600">Turnier ist voll</p>}
                      </div>
                    </div>

                    {tournament.entryFeeCents > 0 && (
                      <div className="flex items-center gap-3">
                        <ClockIcon className="h-5 w-5 text-muted-foreground" />
                        <p className="font-medium">
                          Startgeld: {(tournament.entryFeeCents / 100).toFixed(2)} {tournament.currency}
                        </p>
                      </div>
                    )}

                    {tournament.registrationCloses && (
                      <div className="rounded-lg bg-muted p-3 text-sm">
                        <p className="font-medium">Anmeldeschluss:</p>
                        <p>
                          {format(new Date(tournament.registrationCloses), "d. MMMM yyyy 'um' HH:mm 'Uhr'", {
                            locale: de,
                          })}
                        </p>
                      </div>
                    )}
                  </CardContent>
                </Card>

                {/* Check-in Info */}
                {registrationSettings.checkIn.enabled && (
                  <Card>
                    <CardHeader>
                      <CardTitle>Check-in Information</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <p className="text-sm text-muted-foreground">
                        Das Check-in Fenster öffnet {registrationSettings.checkIn.windowOpenMinutes} Minuten
                        vor Turnierbeginn und schließt {registrationSettings.checkIn.windowCloseMinutes}{' '}
                        Minuten nach Beginn.
                      </p>
                      {isCheckInOpen && (
                        <Alert className="mt-4" variant="default">
                          <CheckCircleIcon className="h-4 w-4" />
                          <AlertTitle>Check-in ist jetzt möglich!</AlertTitle>
                          <AlertDescription>
                            Klicke auf den Link in deiner Bestätigungs-E-Mail oder gehe zur Check-in Seite.
                          </AlertDescription>
                        </Alert>
                      )}
                    </CardContent>
                  </Card>
                )}
              </div>

              {/* Registration Form */}
              <div>
                <Card>
                  <CardHeader>
                    <CardTitle>Voranmeldung</CardTitle>
                    <CardDescription>
                      {registrationSettings.preRegistration.requiresApproval
                        ? 'Deine Anmeldung muss vom Veranstalter bestätigt werden.'
                        : 'Melde dich jetzt für das Turnier an.'}
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    {registrationResult ? (
                      <Alert variant="default" className="border-green-500">
                        <CheckCircleIcon className="h-4 w-4 text-green-600" />
                        <AlertTitle>Anmeldung erfolgreich!</AlertTitle>
                        <AlertDescription className="space-y-2">
                          <p>{registrationResult.message}</p>
                          {registrationResult.checkInCode && (
                            <div className="mt-3 rounded bg-muted p-3">
                              <p className="text-xs text-muted-foreground">Dein Check-in Code:</p>
                              <p className="font-mono text-lg font-bold">{registrationResult.checkInCode}</p>
                            </div>
                          )}
                        </AlertDescription>
                      </Alert>
                    ) : !isRegistrationOpen ? (
                      <Alert>
                        <AlertTitle>Anmeldung geschlossen</AlertTitle>
                        <AlertDescription>
                          Die Anmeldung für dieses Turnier ist derzeit nicht möglich.
                        </AlertDescription>
                      </Alert>
                    ) : isFull ? (
                      <Alert>
                        <AlertTitle>Turnier voll</AlertTitle>
                        <AlertDescription>
                          Das Turnier hat die maximale Teilnehmerzahl erreicht.
                        </AlertDescription>
                      </Alert>
                    ) : (
                      <form onSubmit={handleSubmit} className="space-y-4">
                        {requiresInvite && (
                          <div className="space-y-2">
                            <Label htmlFor="inviteToken">Einladungscode *</Label>
                            <Input
                              id="inviteToken"
                              value={formData.inviteToken}
                              onChange={(e) => setFormData({ ...formData, inviteToken: e.target.value })}
                              placeholder="Einladungscode eingeben"
                              required
                            />
                          </div>
                        )}

                        <div className="grid gap-4 sm:grid-cols-2">
                          <div className="space-y-2">
                            <Label htmlFor="firstName">Vorname *</Label>
                            <Input
                              id="firstName"
                              value={formData.firstName}
                              onChange={(e) => setFormData({ ...formData, firstName: e.target.value })}
                              placeholder="Max"
                              required
                            />
                          </div>
                          <div className="space-y-2">
                            <Label htmlFor="lastName">Nachname *</Label>
                            <Input
                              id="lastName"
                              value={formData.lastName}
                              onChange={(e) => setFormData({ ...formData, lastName: e.target.value })}
                              placeholder="Mustermann"
                              required
                            />
                          </div>
                        </div>

                        <div className="space-y-2">
                          <Label htmlFor="email">E-Mail *</Label>
                          <Input
                            id="email"
                            type="email"
                            value={formData.email}
                            onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                            placeholder="max@example.com"
                            required
                          />
                        </div>

                        <div className="space-y-2">
                          <Label htmlFor="federationId">
                            DSB/FIDE ID <span className="text-muted-foreground">(optional)</span>
                          </Label>
                          <Input
                            id="federationId"
                            value={formData.federationId}
                            onChange={(e) => setFormData({ ...formData, federationId: e.target.value })}
                            placeholder="z.B. 12345678"
                          />
                          <p className="text-xs text-muted-foreground">
                            Deine Spielerdaten werden automatisch aus der Datenbank geladen.
                          </p>
                        </div>

                        {preRegisterMutation.error && (
                          <Alert variant="destructive">
                            <AlertDescription>
                              {(preRegisterMutation.error as Error).message || 'Ein Fehler ist aufgetreten.'}
                            </AlertDescription>
                          </Alert>
                        )}

                        <Button type="submit" className="w-full" disabled={preRegisterMutation.isPending}>
                          {preRegisterMutation.isPending ? 'Wird angemeldet...' : 'Jetzt anmelden'}
                        </Button>

                        {isAuthenticated && (
                          <p className="text-center text-xs text-muted-foreground">
                            Angemeldet als {user?.email}
                          </p>
                        )}
                      </form>
                    )}
                  </CardContent>
                </Card>
              </div>
            </div>
          </TabsContent>

          {/* Standings Tab */}
          <TabsContent value="standings">
            {hasStarted && tournamentId ? (
              <Card>
                <CardHeader>
                  <CardTitle>Aktuelle Rangliste</CardTitle>
                </CardHeader>
                <CardContent>
                  <PublicStandingsTab tournamentId={tournamentId} />
                </CardContent>
              </Card>
            ) : (
              <Card>
                <CardContent className="py-8 text-center text-muted-foreground">
                  Die Rangliste wird angezeigt, sobald das Turnier gestartet ist.
                </CardContent>
              </Card>
            )}
          </TabsContent>

          {/* Pairings Tab */}
          <TabsContent value="pairings">
            {hasStarted && tournamentId ? (
              <Card>
                <CardHeader>
                  <CardTitle>Paarungen</CardTitle>
                </CardHeader>
                <CardContent>
                  <PublicPairingsTab
                    tournamentId={tournamentId}
                    currentRound={currentRound}
                    totalRounds={totalRounds}
                  />
                </CardContent>
              </Card>
            ) : (
              <Card>
                <CardContent className="py-8 text-center text-muted-foreground">
                  Die Paarungen werden angezeigt, sobald das Turnier gestartet ist.
                </CardContent>
              </Card>
            )}
          </TabsContent>

          {/* Crosstable Tab */}
          <TabsContent value="crosstable">
            {hasStarted && tournamentId ? (
              <Card>
                <CardHeader>
                  <CardTitle>Kreuztabelle</CardTitle>
                </CardHeader>
                <CardContent>
                  <PublicCrosstableTab tournamentId={tournamentId} />
                </CardContent>
              </Card>
            ) : (
              <Card>
                <CardContent className="py-8 text-center text-muted-foreground">
                  Die Kreuztabelle wird angezeigt, sobald das Turnier gestartet ist.
                </CardContent>
              </Card>
            )}
          </TabsContent>

          {/* Seating Tab (Round Robin only) */}
          {tournament.type === 'round_robin' && (
            <TabsContent value="seating">
              {hasStarted && tournamentId ? (
                <Card>
                  <CardHeader>
                    <CardTitle>Sitzplan (Rutsch-System)</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <SeatingView
                      tournamentId={tournamentId}
                      totalRounds={totalRounds}
                      initialRound={currentRound}
                      mode="public"
                    />
                  </CardContent>
                </Card>
              ) : (
                <Card>
                  <CardContent className="py-8 text-center text-muted-foreground">
                    Der Sitzplan wird angezeigt, sobald das Turnier gestartet ist.
                  </CardContent>
                </Card>
              )}
            </TabsContent>
          )}

          {/* Participants Tab */}
          <TabsContent value="participants">
            <Card>
              <CardHeader>
                <CardTitle>Teilnehmerliste</CardTitle>
              </CardHeader>
              <CardContent>
                {tournamentId ? (
                  <PublicParticipantsTab
                    tournamentId={tournamentId}
                    participantCount={tournament.participantCount}
                  />
                ) : (
                  <div className="text-center py-8 text-muted-foreground">
                    {tournament.participantCount} Teilnehmer angemeldet
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
