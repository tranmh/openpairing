'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import * as z from 'zod';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
} from '@/components/ui/command';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { cn } from '@/lib/utils';
import { Check, ChevronsUpDown, Search, UserPlus } from 'lucide-react';
import { useDebounce } from 'use-debounce';
import { api, federationApi, FederationPlayer } from '@/lib/api';
import { toast } from 'sonner';

interface Player {
  id: string;
  firstName: string;
  lastName: string;
  fideId?: string;
  dsbId?: string;
  fideStandard?: number;
  dwz?: number;
}

interface RegistrationFormProps {
  tournamentId: string;
  onSuccess?: () => void;
}

const formSchema = z
  .object({
    playerId: z.string().optional(),
    dsbId: z.string().optional(),
    startingRating: z.coerce.number().min(0).optional(),
  })
  .refine((data) => data.playerId || data.dsbId, {
    message: 'Spieler auswählen oder aus Portal64 importieren',
    path: ['playerId'],
  });

type FormValues = z.infer<typeof formSchema>;

export function RegistrationForm({
  tournamentId,
  onSuccess,
}: RegistrationFormProps) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [debouncedSearch] = useDebounce(search, 300);
  const [searchMode, setSearchMode] = useState<'local' | 'federation'>('local');
  const [selectedFederationPlayer, setSelectedFederationPlayer] =
    useState<FederationPlayer | null>(null);

  // Local player search
  const { data: playersData, isLoading: localSearchLoading } = useQuery({
    queryKey: ['players', 'search', debouncedSearch],
    queryFn: async () => {
      if (!debouncedSearch || debouncedSearch.length < 2) return { players: [] };
      const response = await api.get<{ players: Player[] }>(
        `/players?q=${encodeURIComponent(debouncedSearch)}&limit=10`
      );
      return response.data;
    },
    enabled: searchMode === 'local' && debouncedSearch.length >= 2,
  });

  // Federation (Portal64) search
  const { data: federationData, isLoading: federationSearchLoading, error: federationError } = useQuery({
    queryKey: ['federation', 'search', debouncedSearch],
    queryFn: async () => {
      const response = await federationApi.search(debouncedSearch, 'dsb', 10);
      if (response.error) {
        throw new Error(response.error.message || 'Portal64 Suche fehlgeschlagen');
      }
      return response.data;
    },
    enabled: searchMode === 'federation' && debouncedSearch.length >= 2,
    retry: 1,
  });

  const searchLoading =
    searchMode === 'local' ? localSearchLoading : federationSearchLoading;

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      playerId: '',
    },
  });

  const registerMutation = useMutation({
    mutationFn: async (data: FormValues) => {
      // Clean up empty strings - backend UUID parsing fails on empty string
      const payload: Record<string, unknown> = {};
      if (data.playerId) payload.playerId = data.playerId;
      if (data.dsbId) payload.dsbId = data.dsbId;
      if (data.startingRating) payload.startingRating = data.startingRating;

      const response = await api.post(`/tournaments/${tournamentId}/participants`, payload);
      if (response.error) {
        throw new Error(response.error.message || 'Fehler bei der Anmeldung');
      }
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['participants', tournamentId] });
      toast.success('Spieler erfolgreich angemeldet');
      form.reset();
      setSearch('');
      setSelectedFederationPlayer(null);
      onSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Fehler bei der Anmeldung');
    },
  });

  const players: Player[] = playersData?.players || [];
  const federationPlayers: FederationPlayer[] = federationData?.results || [];
  const selectedPlayer = players.find((p) => p.id === form.watch('playerId'));

  // Get display name for selected player
  const getSelectedDisplayName = () => {
    if (searchMode === 'local' && selectedPlayer) {
      return (
        <span>
          {selectedPlayer.lastName}, {selectedPlayer.firstName}
          {selectedPlayer.dwz && ` (${selectedPlayer.dwz})`}
        </span>
      );
    }
    if (searchMode === 'federation' && selectedFederationPlayer) {
      const displayName =
        selectedFederationPlayer.lastName && selectedFederationPlayer.firstName
          ? `${selectedFederationPlayer.lastName}, ${selectedFederationPlayer.firstName}`
          : selectedFederationPlayer.name;
      return (
        <span>
          {displayName}
          {selectedFederationPlayer.rating && ` (${selectedFederationPlayer.rating})`}
        </span>
      );
    }
    return null;
  };

  const onSubmit = (data: FormValues) => {
    if (searchMode === 'local') {
      // Use player's rating if not manually specified
      if (!data.startingRating && selectedPlayer) {
        const rating = selectedPlayer.dwz || selectedPlayer.fideStandard;
        if (rating) {
          data.startingRating = rating;
        }
      }
    } else {
      // Federation mode - use rating from selected federation player
      if (!data.startingRating && selectedFederationPlayer) {
        const rating = selectedFederationPlayer.rating || selectedFederationPlayer.fideRating;
        if (rating) {
          data.startingRating = rating;
        }
      }
    }
    registerMutation.mutate(data);
  };

  // Handle mode switch - reset form
  const handleModeSwitch = (mode: 'local' | 'federation') => {
    setSearchMode(mode);
    setSearch('');
    setSelectedFederationPlayer(null);
    form.setValue('playerId', '');
    form.setValue('dsbId', '');
    form.setValue('startingRating', undefined);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <UserPlus className="h-5 w-5" />
          Spieler anmelden
        </CardTitle>
        <CardDescription>
          Suchen Sie einen Spieler und melden Sie ihn für das Turnier an
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
            <FormField
              control={form.control}
              name="playerId"
              render={({ field }) => (
                <FormItem className="flex flex-col">
                  <FormLabel>Spieler *</FormLabel>
                  <div className="flex gap-2 mb-2">
                    <Button
                      type="button"
                      variant={searchMode === 'local' ? 'default' : 'outline'}
                      size="sm"
                      onClick={() => handleModeSwitch('local')}
                    >
                      Lokal
                    </Button>
                    <Button
                      type="button"
                      variant={searchMode === 'federation' ? 'default' : 'outline'}
                      size="sm"
                      onClick={() => handleModeSwitch('federation')}
                    >
                      Portal64
                    </Button>
                  </div>
                  <Popover open={open} onOpenChange={setOpen}>
                    <PopoverTrigger asChild>
                      <FormControl>
                        <Button
                          variant="outline"
                          role="combobox"
                          className={cn(
                            'justify-between',
                            !field.value &&
                              !form.watch('dsbId') &&
                              'text-muted-foreground'
                          )}
                        >
                          {getSelectedDisplayName() || 'Spieler suchen...'}
                          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                        </Button>
                      </FormControl>
                    </PopoverTrigger>
                    <PopoverContent className="w-[400px] p-0">
                      <Command>
                        <CommandInput
                          placeholder={
                            searchMode === 'local'
                              ? 'Name oder ID eingeben...'
                              : 'Name in Portal64 suchen...'
                          }
                          value={search}
                          onValueChange={setSearch}
                        />
                        <CommandEmpty>
                          {searchLoading ? (
                            'Suche...'
                          ) : search.length < 2 ? (
                            'Mindestens 2 Zeichen eingeben'
                          ) : federationError && searchMode === 'federation' ? (
                            <span className="text-destructive">
                              Portal64 nicht erreichbar
                            </span>
                          ) : (
                            'Kein Spieler gefunden'
                          )}
                        </CommandEmpty>
                        <CommandGroup>
                          {searchMode === 'local'
                            ? players.map((player) => (
                                <CommandItem
                                  key={player.id}
                                  value={player.id}
                                  onSelect={() => {
                                    form.setValue('playerId', player.id);
                                    form.setValue('dsbId', '');
                                    setSelectedFederationPlayer(null);
                                    if (player.dwz || player.fideStandard) {
                                      form.setValue(
                                        'startingRating',
                                        player.dwz || player.fideStandard
                                      );
                                    }
                                    setOpen(false);
                                  }}
                                >
                                  <Check
                                    className={cn(
                                      'mr-2 h-4 w-4',
                                      player.id === field.value
                                        ? 'opacity-100'
                                        : 'opacity-0'
                                    )}
                                  />
                                  <div className="flex flex-col">
                                    <span>
                                      {player.lastName}, {player.firstName}
                                    </span>
                                    <span className="text-xs text-muted-foreground">
                                      {[
                                        player.dwz && `DWZ: ${player.dwz}`,
                                        player.fideStandard &&
                                          `ELO: ${player.fideStandard}`,
                                        player.fideId && `FIDE: ${player.fideId}`,
                                        player.dsbId && `PKZ: ${player.dsbId}`,
                                      ]
                                        .filter(Boolean)
                                        .join(' | ')}
                                    </span>
                                  </div>
                                </CommandItem>
                              ))
                            : federationPlayers.map((player) => {
                                const displayName =
                                  player.lastName && player.firstName
                                    ? `${player.lastName}, ${player.firstName}`
                                    : player.name;
                                return (
                                  <CommandItem
                                    key={player.id}
                                    value={player.id}
                                    onSelect={() => {
                                      form.setValue('dsbId', player.id);
                                      form.setValue('playerId', '');
                                      setSelectedFederationPlayer(player);
                                      if (player.rating || player.fideRating) {
                                        form.setValue(
                                          'startingRating',
                                          player.rating || player.fideRating
                                        );
                                      }
                                      setOpen(false);
                                    }}
                                  >
                                    <Check
                                      className={cn(
                                        'mr-2 h-4 w-4',
                                        player.id === form.watch('dsbId')
                                          ? 'opacity-100'
                                          : 'opacity-0'
                                      )}
                                    />
                                    <div className="flex flex-col">
                                      <span>{displayName}</span>
                                      <span className="text-xs text-muted-foreground">
                                        {[
                                          player.rating && `DWZ: ${player.rating}`,
                                          player.fideRating &&
                                            `ELO: ${player.fideRating}`,
                                          player.title,
                                          player.club,
                                        ]
                                          .filter(Boolean)
                                          .join(' | ')}
                                      </span>
                                    </div>
                                  </CommandItem>
                                );
                              })}
                        </CommandGroup>
                      </Command>
                    </PopoverContent>
                  </Popover>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="startingRating"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Startwertung</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      placeholder="Automatisch aus Profil"
                      {...field}
                      value={field.value || ''}
                    />
                  </FormControl>
                  <FormDescription>
                    Wird automatisch aus dem Spielerprofil übernommen
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className="flex gap-4">
              <Button
                type="submit"
                disabled={registerMutation.isPending}
                className="flex-1"
              >
                <UserPlus className="mr-2 h-4 w-4" />
                {registerMutation.isPending ? 'Wird angemeldet...' : 'Anmelden'}
              </Button>
            </div>
          </form>
        </Form>
      </CardContent>
    </Card>
  );
}
