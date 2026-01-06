'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
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
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Checkbox } from '@/components/ui/checkbox';
import { Calendar } from '@/components/ui/calendar';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { cn } from '@/lib/utils';
import { format } from 'date-fns';
import { de } from 'date-fns/locale';
import { CalendarIcon, ArrowLeft, ArrowRight, Trophy } from 'lucide-react';
import { api } from '@/lib/api';
import { toast } from 'sonner';

const formSchema = z.object({
  name: z.string().min(3, 'Name muss mindestens 3 Zeichen haben'),
  description: z.string().optional(),
  type: z.enum(['swiss', 'round_robin', 'knockout', 'team_swiss', 'wam']),
  mode: z.enum(['otb', 'online', 'hybrid']),
  rounds: z.coerce.number().min(1).max(20),
  timeControl: z.string().min(1, 'Bedenkzeit ist erforderlich'),
  timeControlType: z.enum(['standard', 'rapid', 'blitz']),
  startDate: z.date({ required_error: 'Startdatum ist erforderlich' }),
  endDate: z.date().optional(),
  venueCity: z.string().optional(),
  venueName: z.string().optional(),
  venueAddress: z.string().optional(),
  maxPlayers: z.coerce.number().optional(),
  minPlayers: z.coerce.number().min(2).default(4),
  fideRated: z.boolean().default(false),
  dwzRated: z.boolean().default(false),
  pairingSystem: z.enum(['dutch', 'burstein', 'dubov']).default('dutch'),
  pairingEngine: z.enum(['javafo', 'bbp', 'wam']).optional(),
  tiebreaks: z.array(z.string()).default(['buchholz_cut1', 'buchholz', 'sonneborn_berger']),
  // WAM-specific settings
  wamGroupSize: z.coerce.number().min(3).max(10).optional(),
  wamAssignmentMethod: z.enum(['sequential', 'serpentine', 'random']).optional(),
  wamEnablePlayoffs: z.boolean().optional(),
  wamPlayoffsTopN: z.coerce.number().min(1).max(4).optional(),
});

type FormValues = z.infer<typeof formSchema>;

const steps = [
  { id: 'basic', title: 'Grunddaten' },
  { id: 'details', title: 'Details' },
  { id: 'settings', title: 'Einstellungen' },
];

export default function NewTournamentPage() {
  const router = useRouter();
  const [step, setStep] = useState(0);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      type: 'swiss',
      mode: 'otb',
      rounds: 7,
      timeControl: '90+30',
      timeControlType: 'standard',
      minPlayers: 4,
      fideRated: false,
      dwzRated: true,
      pairingSystem: 'dutch',
      tiebreaks: ['buchholz_cut1', 'buchholz', 'sonneborn_berger'],
    },
  });

  const createMutation = useMutation({
    mutationFn: (data: FormValues) => api.post<{ tournament: { id: string } }>('/tournaments', data),
    onSuccess: (response) => {
      toast.success('Turnier erstellt');
      if (response.data) {
        router.push(`/tournaments/${response.data.tournament.id}`);
      }
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Fehler beim Erstellen');
    },
  });

  const onSubmit = (data: FormValues) => {
    createMutation.mutate(data);
  };

  const nextStep = async () => {
    const fieldsToValidate = getFieldsForStep(step);
    const isValid = await form.trigger(fieldsToValidate as any);
    if (isValid) {
      setStep(step + 1);
    }
  };

  const prevStep = () => {
    setStep(step - 1);
  };

  return (
    <div className="container max-w-3xl mx-auto py-6 space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Neues Turnier</h1>
        <p className="text-muted-foreground">
          Erstellen Sie ein neues Schachturnier
        </p>
      </div>

      <div className="flex justify-between mb-8">
        {steps.map((s, i) => (
          <div
            key={s.id}
            className={cn(
              'flex items-center',
              i < steps.length - 1 && 'flex-1'
            )}
          >
            <div
              className={cn(
                'w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium',
                i <= step
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-muted text-muted-foreground'
              )}
            >
              {i + 1}
            </div>
            <span
              className={cn(
                'ml-2 text-sm font-medium',
                i <= step ? 'text-foreground' : 'text-muted-foreground'
              )}
            >
              {s.title}
            </span>
            {i < steps.length - 1 && (
              <div
                className={cn(
                  'flex-1 h-px mx-4',
                  i < step ? 'bg-primary' : 'bg-muted'
                )}
              />
            )}
          </div>
        ))}
      </div>

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)}>
          <Card>
            <CardHeader>
              <CardTitle>{steps[step].title}</CardTitle>
              <CardDescription>
                {step === 0 && 'Geben Sie die grundlegenden Turnierinformationen ein'}
                {step === 1 && 'Legen Sie Datum, Ort und Teilnehmerzahl fest'}
                {step === 2 && 'Konfigurieren Sie Paarungssystem und Feinwertungen'}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              {step === 0 && (
                <>
                  <FormField
                    control={form.control}
                    name="name"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Turniername *</FormLabel>
                        <FormControl>
                          <Input placeholder="z.B. Vereinsmeisterschaft 2024" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="description"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Beschreibung</FormLabel>
                        <FormControl>
                          <Textarea
                            placeholder="Optionale Beschreibung des Turniers"
                            {...field}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <div className="grid grid-cols-2 gap-4">
                    <FormField
                      control={form.control}
                      name="type"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Turniertyp *</FormLabel>
                          <Select onValueChange={field.onChange} defaultValue={field.value}>
                            <FormControl>
                              <SelectTrigger>
                                <SelectValue placeholder="Typ wählen" />
                              </SelectTrigger>
                            </FormControl>
                            <SelectContent>
                              <SelectItem value="swiss">Schweizer System</SelectItem>
                              <SelectItem value="round_robin">Rundenturnier</SelectItem>
                              <SelectItem value="knockout">K.O.-System</SelectItem>
                              <SelectItem value="team_swiss">Mannschaftsturnier</SelectItem>
                              <SelectItem value="wam">WAM (Gruppen + Playoffs)</SelectItem>
                            </SelectContent>
                          </Select>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="mode"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Modus *</FormLabel>
                          <Select onValueChange={field.onChange} defaultValue={field.value}>
                            <FormControl>
                              <SelectTrigger>
                                <SelectValue placeholder="Modus wählen" />
                              </SelectTrigger>
                            </FormControl>
                            <SelectContent>
                              <SelectItem value="otb">Präsenz (OTB)</SelectItem>
                              <SelectItem value="online">Online</SelectItem>
                              <SelectItem value="hybrid">Hybrid</SelectItem>
                            </SelectContent>
                          </Select>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <FormField
                      control={form.control}
                      name="rounds"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Anzahl Runden *</FormLabel>
                          <FormControl>
                            <Input type="number" min={1} max={20} {...field} />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="timeControl"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Bedenkzeit *</FormLabel>
                          <FormControl>
                            <Input placeholder="z.B. 90+30" {...field} />
                          </FormControl>
                          <FormDescription>
                            Format: Minuten+Inkrement (z.B. 90+30)
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>

                  <FormField
                    control={form.control}
                    name="timeControlType"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Kategorie *</FormLabel>
                        <Select onValueChange={field.onChange} defaultValue={field.value}>
                          <FormControl>
                            <SelectTrigger>
                              <SelectValue placeholder="Kategorie wählen" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectItem value="standard">Standard (≥60 Min)</SelectItem>
                            <SelectItem value="rapid">Schnellschach (15-60 Min)</SelectItem>
                            <SelectItem value="blitz">Blitz (&lt;15 Min)</SelectItem>
                          </SelectContent>
                        </Select>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </>
              )}

              {step === 1 && (
                <>
                  <div className="grid grid-cols-2 gap-4">
                    <FormField
                      control={form.control}
                      name="startDate"
                      render={({ field }) => (
                        <FormItem className="flex flex-col">
                          <FormLabel>Startdatum *</FormLabel>
                          <Popover>
                            <PopoverTrigger asChild>
                              <FormControl>
                                <Button
                                  variant="outline"
                                  className={cn(
                                    'pl-3 text-left font-normal',
                                    !field.value && 'text-muted-foreground'
                                  )}
                                >
                                  {field.value ? (
                                    format(field.value, 'PPP', { locale: de })
                                  ) : (
                                    <span>Datum wählen</span>
                                  )}
                                  <CalendarIcon className="ml-auto h-4 w-4 opacity-50" />
                                </Button>
                              </FormControl>
                            </PopoverTrigger>
                            <PopoverContent className="w-auto p-0" align="start">
                              <Calendar
                                mode="single"
                                selected={field.value}
                                onSelect={field.onChange}
                                locale={de}
                                initialFocus
                              />
                            </PopoverContent>
                          </Popover>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="endDate"
                      render={({ field }) => (
                        <FormItem className="flex flex-col">
                          <FormLabel>Enddatum</FormLabel>
                          <Popover>
                            <PopoverTrigger asChild>
                              <FormControl>
                                <Button
                                  variant="outline"
                                  className={cn(
                                    'pl-3 text-left font-normal',
                                    !field.value && 'text-muted-foreground'
                                  )}
                                >
                                  {field.value ? (
                                    format(field.value, 'PPP', { locale: de })
                                  ) : (
                                    <span>Datum wählen</span>
                                  )}
                                  <CalendarIcon className="ml-auto h-4 w-4 opacity-50" />
                                </Button>
                              </FormControl>
                            </PopoverTrigger>
                            <PopoverContent className="w-auto p-0" align="start">
                              <Calendar
                                mode="single"
                                selected={field.value}
                                onSelect={field.onChange}
                                locale={de}
                                initialFocus
                              />
                            </PopoverContent>
                          </Popover>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>

                  <FormField
                    control={form.control}
                    name="venueName"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Spielort</FormLabel>
                        <FormControl>
                          <Input placeholder="z.B. Schachverein Berlin e.V." {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="venueAddress"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Adresse</FormLabel>
                        <FormControl>
                          <Input placeholder="Straße und Hausnummer" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="venueCity"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Stadt</FormLabel>
                        <FormControl>
                          <Input placeholder="z.B. Berlin" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <div className="grid grid-cols-2 gap-4">
                    <FormField
                      control={form.control}
                      name="minPlayers"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Mindesteilnehmer</FormLabel>
                          <FormControl>
                            <Input type="number" min={2} {...field} />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="maxPlayers"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Maximale Teilnehmer</FormLabel>
                          <FormControl>
                            <Input
                              type="number"
                              min={2}
                              placeholder="Unbegrenzt"
                              {...field}
                              value={field.value || ''}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>
                </>
              )}

              {step === 2 && (
                <>
                  {/* Show pairing system only for Swiss tournaments */}
                  {form.watch('type') === 'swiss' && (
                    <>
                      <FormField
                        control={form.control}
                        name="pairingSystem"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>Paarungssystem</FormLabel>
                            <Select onValueChange={field.onChange} defaultValue={field.value}>
                              <FormControl>
                                <SelectTrigger>
                                  <SelectValue placeholder="System wählen" />
                                </SelectTrigger>
                              </FormControl>
                              <SelectContent>
                                <SelectItem value="dutch">Dutch (FIDE Standard)</SelectItem>
                                <SelectItem value="burstein">Burstein</SelectItem>
                                <SelectItem value="dubov">Dubov</SelectItem>
                              </SelectContent>
                            </Select>
                            <FormDescription>
                              Dutch ist das offizielle FIDE-Paarungssystem
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name="pairingEngine"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>Paarungs-Engine</FormLabel>
                            <Select onValueChange={field.onChange} value={field.value || 'javafo'}>
                              <FormControl>
                                <SelectTrigger>
                                  <SelectValue placeholder="Engine wählen" />
                                </SelectTrigger>
                              </FormControl>
                              <SelectContent>
                                <SelectItem value="javafo">JaVaFo (FIDE-endorsed)</SelectItem>
                                <SelectItem value="bbp">bbpPairings</SelectItem>
                              </SelectContent>
                            </Select>
                            <FormDescription>
                              JaVaFo ist die offizielle FIDE-empfohlene Engine
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </>
                  )}

                  {/* WAM-specific settings */}
                  {form.watch('type') === 'wam' && (
                    <div className="space-y-4 rounded-lg border p-4 bg-muted/50">
                      <h4 className="font-medium">WAM-Einstellungen</h4>

                      <div className="grid grid-cols-2 gap-4">
                        <FormField
                          control={form.control}
                          name="wamGroupSize"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>Gruppengröße</FormLabel>
                              <Select onValueChange={(v) => field.onChange(parseInt(v))} value={field.value?.toString() || '6'}>
                                <FormControl>
                                  <SelectTrigger>
                                    <SelectValue placeholder="Größe wählen" />
                                  </SelectTrigger>
                                </FormControl>
                                <SelectContent>
                                  <SelectItem value="4">4 Spieler</SelectItem>
                                  <SelectItem value="5">5 Spieler</SelectItem>
                                  <SelectItem value="6">6 Spieler</SelectItem>
                                </SelectContent>
                              </Select>
                              <FormMessage />
                            </FormItem>
                          )}
                        />

                        <FormField
                          control={form.control}
                          name="wamAssignmentMethod"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>Gruppeneinteilung</FormLabel>
                              <Select onValueChange={field.onChange} value={field.value || 'serpentine'}>
                                <FormControl>
                                  <SelectTrigger>
                                    <SelectValue placeholder="Methode wählen" />
                                  </SelectTrigger>
                                </FormControl>
                                <SelectContent>
                                  <SelectItem value="sequential">Sequentiell</SelectItem>
                                  <SelectItem value="serpentine">Serpentine (empfohlen)</SelectItem>
                                  <SelectItem value="random">Zufällig</SelectItem>
                                </SelectContent>
                              </Select>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                      </div>

                      <FormField
                        control={form.control}
                        name="wamEnablePlayoffs"
                        render={({ field }) => (
                          <FormItem className="flex items-center space-x-2">
                            <FormControl>
                              <Checkbox
                                checked={field.value}
                                onCheckedChange={field.onChange}
                              />
                            </FormControl>
                            <FormLabel className="!mt-0 font-normal">
                              Playoffs aktivieren (Gruppensieger spielen K.O.-Runde)
                            </FormLabel>
                          </FormItem>
                        )}
                      />

                      {form.watch('wamEnablePlayoffs') && (
                        <FormField
                          control={form.control}
                          name="wamPlayoffsTopN"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>Playoff-Teilnehmer pro Gruppe</FormLabel>
                              <Select onValueChange={(v) => field.onChange(parseInt(v))} value={field.value?.toString() || '1'}>
                                <FormControl>
                                  <SelectTrigger>
                                    <SelectValue />
                                  </SelectTrigger>
                                </FormControl>
                                <SelectContent>
                                  <SelectItem value="1">Top 1</SelectItem>
                                  <SelectItem value="2">Top 2</SelectItem>
                                </SelectContent>
                              </Select>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                      )}
                    </div>
                  )}

                  <div className="space-y-4">
                    <FormLabel>Wertung</FormLabel>
                    <div className="flex gap-6">
                      <FormField
                        control={form.control}
                        name="fideRated"
                        render={({ field }) => (
                          <FormItem className="flex items-center space-x-2">
                            <FormControl>
                              <Checkbox
                                checked={field.value}
                                onCheckedChange={field.onChange}
                              />
                            </FormControl>
                            <FormLabel className="!mt-0 font-normal">
                              FIDE-gewertet
                            </FormLabel>
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name="dwzRated"
                        render={({ field }) => (
                          <FormItem className="flex items-center space-x-2">
                            <FormControl>
                              <Checkbox
                                checked={field.value}
                                onCheckedChange={field.onChange}
                              />
                            </FormControl>
                            <FormLabel className="!mt-0 font-normal">
                              DWZ-gewertet
                            </FormLabel>
                          </FormItem>
                        )}
                      />
                    </div>
                  </div>

                  <FormField
                    control={form.control}
                    name="tiebreaks"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Feinwertungen</FormLabel>
                        <FormDescription>
                          Wählen Sie die Feinwertungen in der Reihenfolge ihrer Priorität
                        </FormDescription>
                        <div className="grid grid-cols-2 gap-2 mt-2">
                          {tiebreakOptions.map((tb) => (
                            <div key={tb.value} className="flex items-center space-x-2">
                              <Checkbox
                                checked={field.value?.includes(tb.value)}
                                onCheckedChange={(checked) => {
                                  const newValue = checked
                                    ? [...(field.value || []), tb.value]
                                    : field.value?.filter((v) => v !== tb.value);
                                  field.onChange(newValue);
                                }}
                              />
                              <label className="text-sm">{tb.label}</label>
                            </div>
                          ))}
                        </div>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </>
              )}
            </CardContent>
          </Card>

          <div className="flex justify-between mt-6">
            <Button
              type="button"
              variant="outline"
              onClick={prevStep}
              disabled={step === 0}
            >
              <ArrowLeft className="mr-2 h-4 w-4" />
              Zurück
            </Button>

            {step < steps.length - 1 ? (
              <Button type="button" onClick={nextStep}>
                Weiter
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            ) : (
              <Button type="submit" disabled={createMutation.isPending}>
                <Trophy className="mr-2 h-4 w-4" />
                {createMutation.isPending ? 'Wird erstellt...' : 'Turnier erstellen'}
              </Button>
            )}
          </div>
        </form>
      </Form>
    </div>
  );
}

function getFieldsForStep(step: number): (keyof FormValues)[] {
  switch (step) {
    case 0:
      return ['name', 'type', 'mode', 'rounds', 'timeControl', 'timeControlType'];
    case 1:
      return ['startDate', 'minPlayers'];
    case 2:
      return ['pairingSystem', 'tiebreaks'];
    default:
      return [];
  }
}

const tiebreakOptions = [
  { value: 'buchholz', label: 'Buchholz' },
  { value: 'buchholz_cut1', label: 'Buchholz Cut 1' },
  { value: 'buchholz_cut2', label: 'Buchholz Cut 2' },
  { value: 'buchholz_median', label: 'Buchholz Median' },
  { value: 'sonneborn_berger', label: 'Sonneborn-Berger' },
  { value: 'direct_encounter', label: 'Direkter Vergleich' },
  { value: 'wins', label: 'Anzahl Siege' },
  { value: 'games_black', label: 'Partien mit Schwarz' },
  { value: 'progress', label: 'Progressive Wertung' },
  { value: 'koya', label: 'Koya-System' },
];
