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
import { api } from '@/lib/api';
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

const formSchema = z.object({
  playerId: z.string().min(1, 'Bitte wählen Sie einen Spieler'),
  startingRating: z.coerce.number().min(0).optional(),
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

  const { data: playersData, isLoading: searchLoading } = useQuery({
    queryKey: ['players', 'search', debouncedSearch],
    queryFn: async () => {
      if (!debouncedSearch || debouncedSearch.length < 2) return { players: [] };
      const response = await api.get(`/players?q=${encodeURIComponent(debouncedSearch)}&limit=10`);
      return response.data;
    },
    enabled: debouncedSearch.length >= 2,
  });

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      playerId: '',
    },
  });

  const registerMutation = useMutation({
    mutationFn: (data: FormValues) =>
      api.post(`/tournaments/${tournamentId}/participants`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['participants', tournamentId] });
      toast.success('Spieler erfolgreich angemeldet');
      form.reset();
      onSuccess?.();
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler bei der Anmeldung');
    },
  });

  const players: Player[] = playersData?.players || [];
  const selectedPlayer = players.find((p) => p.id === form.watch('playerId'));

  const onSubmit = (data: FormValues) => {
    // Use player's rating if not manually specified
    if (!data.startingRating && selectedPlayer) {
      data.startingRating = selectedPlayer.dwz || selectedPlayer.fideStandard || 0;
    }
    registerMutation.mutate(data);
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
                  <Popover open={open} onOpenChange={setOpen}>
                    <PopoverTrigger asChild>
                      <FormControl>
                        <Button
                          variant="outline"
                          role="combobox"
                          className={cn(
                            'justify-between',
                            !field.value && 'text-muted-foreground'
                          )}
                        >
                          {selectedPlayer ? (
                            <span>
                              {selectedPlayer.lastName}, {selectedPlayer.firstName}
                              {selectedPlayer.dwz && ` (${selectedPlayer.dwz})`}
                            </span>
                          ) : (
                            'Spieler suchen...'
                          )}
                          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                        </Button>
                      </FormControl>
                    </PopoverTrigger>
                    <PopoverContent className="w-[400px] p-0">
                      <Command>
                        <CommandInput
                          placeholder="Name oder ID eingeben..."
                          value={search}
                          onValueChange={setSearch}
                        />
                        <CommandEmpty>
                          {searchLoading ? (
                            'Suche...'
                          ) : search.length < 2 ? (
                            'Mindestens 2 Zeichen eingeben'
                          ) : (
                            'Kein Spieler gefunden'
                          )}
                        </CommandEmpty>
                        <CommandGroup>
                          {players.map((player) => (
                            <CommandItem
                              key={player.id}
                              value={player.id}
                              onSelect={() => {
                                form.setValue('playerId', player.id);
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
                                    player.fideStandard && `ELO: ${player.fideStandard}`,
                                    player.fideId && `FIDE: ${player.fideId}`,
                                    player.dsbId && `PKZ: ${player.dsbId}`,
                                  ]
                                    .filter(Boolean)
                                    .join(' | ')}
                                </span>
                              </div>
                            </CommandItem>
                          ))}
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
