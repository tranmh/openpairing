'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useDebounce } from 'use-debounce';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import { ArrowLeft, Search, UserPlus, Loader2, Check } from 'lucide-react';
import { api, federationApi, type FederationPlayer } from '@/lib/api';
import { useAuthStore } from '@/store/auth';
import { toast } from 'sonner';

interface Tournament {
  id: string;
  name: string;
  status: string;
}

interface Participant {
  id: string;
  playerId: string;
  playerName?: string;
  startingRating: number;
  isActive: boolean;
}

export default function RegisterPlayerPage() {
  const params = useParams();
  const router = useRouter();
  const queryClient = useQueryClient();
  const tournamentId = params.id as string;
  const { isHydrated, isAuthenticated } = useAuthStore();

  const [search, setSearch] = useState('');
  const [debouncedSearch] = useDebounce(search, 300);
  const [registering, setRegistering] = useState<string | null>(null);

  // Wait for auth hydration before making authenticated API calls
  const canFetch = isHydrated && isAuthenticated;

  // Fetch tournament info (public endpoint, doesn't require auth)
  const { data: tournamentData, isLoading: tournamentLoading } = useQuery<{ tournament: Tournament }>({
    queryKey: ['tournament', tournamentId],
    queryFn: async () => {
      const response = await api.get(`/tournaments/${tournamentId}`);
      return response.data as { tournament: Tournament };
    },
  });

  // Fetch existing participants (requires auth)
  const { data: participantsData } = useQuery<{ participants: Participant[] }>({
    queryKey: ['participants', tournamentId],
    queryFn: async () => {
      const response = await api.get(`/tournaments/${tournamentId}/participants`);
      return response.data as { participants: Participant[] };
    },
    enabled: canFetch,
  });

  // Search federation players
  const { data: searchResults, isLoading: searchLoading } = useQuery({
    queryKey: ['federation', 'search', debouncedSearch],
    queryFn: async () => {
      const response = await federationApi.search(debouncedSearch, 'dsb', 20);
      if (response.error) throw new Error(response.error.message);
      return response.data;
    },
    enabled: debouncedSearch.length >= 2,
  });

  // Register player mutation
  const registerMutation = useMutation({
    mutationFn: async (player: FederationPlayer) => {
      // Use the participant registration endpoint with dsbId to auto-import
      const response = await api.post(`/tournaments/${tournamentId}/participants`, {
        dsbId: player.id,
        startingRating: player.rating || player.fideRating,
      });
      if (response.status >= 400) {
        throw new Error((response as any).error?.message || 'Fehler beim Registrieren');
      }
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['participants', tournamentId] });
      toast.success('Spieler erfolgreich angemeldet');
      setRegistering(null);
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Fehler beim Anmelden');
      setRegistering(null);
    },
  });

  const tournament = tournamentData?.tournament;
  const participants = participantsData?.participants || [];
  const registeredPlayerIds = new Set(participants.map(p => p.playerId));

  const handleRegister = (player: FederationPlayer) => {
    setRegistering(player.id);
    registerMutation.mutate(player);
  };

  if (tournamentLoading) {
    return (
      <div className="container mx-auto py-6 space-y-6">
        <Skeleton className="h-10 w-1/3" />
        <Skeleton className="h-64" />
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

  if (tournament.status !== 'registration' && tournament.status !== 'draft' && tournament.status !== 'published') {
    return (
      <div className="container mx-auto py-6">
        <Card>
          <CardContent className="py-10 text-center">
            <p className="text-muted-foreground">
              Anmeldung ist nicht mehr moeglich. Das Turnier hat bereits begonnen.
            </p>
            <Link href={`/tournaments/${tournamentId}`}>
              <Button className="mt-4">
                <ArrowLeft className="mr-2 h-4 w-4" />
                Zurueck zum Turnier
              </Button>
            </Link>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-6 space-y-6">
      <div className="flex items-center gap-4">
        <Link href={`/tournaments/${tournamentId}`}>
          <Button variant="ghost" size="sm">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Zurueck
          </Button>
        </Link>
        <div>
          <h1 className="text-2xl font-bold">Spieler anmelden</h1>
          <p className="text-muted-foreground">{tournament.name}</p>
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        {/* Search and Register */}
        <Card>
          <CardHeader>
            <CardTitle>Spieler suchen</CardTitle>
            <CardDescription>
              Suchen Sie nach Spielern im Portal64 (DSB)
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="search">Name eingeben</Label>
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  id="search"
                  className="pl-10"
                  placeholder="z.B. Mueller, Max"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                />
              </div>
            </div>

            {searchLoading && (
              <div className="py-4 text-center text-muted-foreground">
                <Loader2 className="h-6 w-6 animate-spin mx-auto" />
                <p className="mt-2 text-sm">Suche...</p>
              </div>
            )}

            {searchResults?.results && searchResults.results.length > 0 && (
              <div className="border rounded-md max-h-96 overflow-y-auto divide-y">
                {searchResults.results.map((player) => {
                  const isRegistered = registeredPlayerIds.has(player.id);
                  const isRegistering = registering === player.id;

                  return (
                    <div
                      key={player.id}
                      className="p-3 hover:bg-muted/50 flex items-center justify-between"
                    >
                      <div>
                        <div className="font-medium">
                          {player.firstName} {player.name}
                        </div>
                        <div className="text-sm text-muted-foreground">
                          {player.club && `${player.club} - `}
                          DWZ: {player.rating || '-'}
                          {player.fideRating && ` | FIDE: ${player.fideRating}`}
                        </div>
                      </div>
                      {isRegistered ? (
                        <Button variant="ghost" size="sm" disabled>
                          <Check className="mr-2 h-4 w-4 text-green-600" />
                          Angemeldet
                        </Button>
                      ) : (
                        <Button
                          size="sm"
                          onClick={() => handleRegister(player)}
                          disabled={isRegistering || !canFetch}
                        >
                          {isRegistering ? (
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          ) : (
                            <UserPlus className="mr-2 h-4 w-4" />
                          )}
                          Anmelden
                        </Button>
                      )}
                    </div>
                  );
                })}
              </div>
            )}

            {debouncedSearch.length >= 2 && !searchLoading && (!searchResults?.results || searchResults.results.length === 0) && (
              <p className="text-center text-muted-foreground py-4">
                Keine Spieler gefunden
              </p>
            )}
          </CardContent>
        </Card>

        {/* Registered Players */}
        <Card>
          <CardHeader>
            <CardTitle>Angemeldete Spieler ({participants.filter(p => p.isActive).length})</CardTitle>
            <CardDescription>
              Liste aller bereits angemeldeten Teilnehmer
            </CardDescription>
          </CardHeader>
          <CardContent>
            {participants.filter(p => p.isActive).length === 0 ? (
              <p className="text-center text-muted-foreground py-8">
                Noch keine Spieler angemeldet
              </p>
            ) : (
              <div className="border rounded-md max-h-96 overflow-y-auto divide-y">
                {participants
                  .filter(p => p.isActive)
                  .map((p) => (
                    <div key={p.id} className="p-3 flex justify-between items-center">
                      <span className="font-medium">{p.playerName || p.playerId}</span>
                      <span className="text-muted-foreground">{p.startingRating}</span>
                    </div>
                  ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
