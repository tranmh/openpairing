'use client';

import { useState, useEffect } from 'react';
import { useParams, useSearchParams } from 'next/navigation';
import { useQuery, useMutation } from '@tanstack/react-query';
import Link from 'next/link';
import { CheckCircleIcon, XCircleIcon, ClockIcon } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { api } from '@/lib/api';

interface CheckInResponse {
  success: boolean;
  id: string;
  status: string;
  checkedIn: string;
  message: string;
}

interface TournamentInfo {
  name: string;
  startDate: string;
}

export default function CheckInPage() {
  const params = useParams();
  const searchParams = useSearchParams();
  const slug = params.slug as string;
  const codeFromUrl = searchParams.get('code');

  const [code, setCode] = useState(codeFromUrl || '');
  const [checkInResult, setCheckInResult] = useState<CheckInResponse | null>(null);

  // Fetch tournament info
  const { data: tournamentData, isLoading } = useQuery({
    queryKey: ['public-tournament', slug],
    queryFn: async () => {
      const response = await api.get<{
        tournament: TournamentInfo;
        isCheckInOpen: boolean;
      }>(`/public/tournaments/${slug}`);
      return response.data;
    },
  });

  // Check-in mutation
  const checkInMutation = useMutation({
    mutationFn: async (checkInCode: string) => {
      const response = await api.post<CheckInResponse>(`/public/tournaments/${slug}/check-in`, {
        code: checkInCode,
        method: 'self_service',
      });
      return response.data;
    },
    onSuccess: (result) => {
      if (result) {
        setCheckInResult(result);
      }
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (code.trim()) {
      checkInMutation.mutate(code.trim());
    }
  };

  // Auto check-in if code is in URL
  useEffect(() => {
    if (codeFromUrl && !checkInResult && !checkInMutation.isPending) {
      checkInMutation.mutate(codeFromUrl);
    }
  }, [codeFromUrl]); // Only run when codeFromUrl changes

  if (isLoading) {
    return (
      <div className="container mx-auto flex min-h-[50vh] items-center justify-center px-4">
        <div className="animate-pulse text-muted-foreground">Laden...</div>
      </div>
    );
  }

  const isCheckInOpen = tournamentData?.isCheckInOpen ?? false;

  return (
    <div className="container mx-auto px-4 py-12">
      <div className="mx-auto max-w-md">
        <Card>
          <CardHeader className="text-center">
            <CardTitle className="text-2xl">Check-in</CardTitle>
            <CardDescription>
              {tournamentData?.tournament?.name || 'Turnier'}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {checkInResult ? (
              <div className="space-y-4">
                {checkInResult.success ? (
                  <Alert variant="default" className="border-green-500 bg-green-50">
                    <CheckCircleIcon className="h-5 w-5 text-green-600" />
                    <AlertTitle className="text-green-800">Erfolgreich eingecheckt!</AlertTitle>
                    <AlertDescription className="text-green-700">
                      {checkInResult.message}
                    </AlertDescription>
                  </Alert>
                ) : (
                  <Alert variant="destructive">
                    <XCircleIcon className="h-5 w-5" />
                    <AlertTitle>Check-in fehlgeschlagen</AlertTitle>
                    <AlertDescription>{checkInResult.message}</AlertDescription>
                  </Alert>
                )}

                <div className="text-center">
                  <Link href={`/t/${slug}`}>
                    <Button variant="outline">Zurück zum Turnier</Button>
                  </Link>
                </div>
              </div>
            ) : !isCheckInOpen ? (
              <div className="space-y-4">
                <Alert>
                  <ClockIcon className="h-5 w-5" />
                  <AlertTitle>Check-in noch nicht geöffnet</AlertTitle>
                  <AlertDescription>
                    Das Check-in Fenster öffnet kurz vor Turnierbeginn. Bitte komm später zurück.
                  </AlertDescription>
                </Alert>

                <div className="text-center">
                  <Link href={`/t/${slug}`}>
                    <Button variant="outline">Zurück zum Turnier</Button>
                  </Link>
                </div>
              </div>
            ) : (
              <form onSubmit={handleSubmit} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="code">Check-in Code</Label>
                  <Input
                    id="code"
                    value={code}
                    onChange={(e) => setCode(e.target.value.toUpperCase())}
                    placeholder="z.B. ABCD1234"
                    className="text-center font-mono text-lg tracking-widest"
                    maxLength={20}
                    autoFocus
                  />
                  <p className="text-xs text-center text-muted-foreground">
                    Den Code findest du in deiner Bestätigungs-E-Mail.
                  </p>
                </div>

                {checkInMutation.error && (
                  <Alert variant="destructive">
                    <AlertDescription>
                      {(checkInMutation.error as Error).message || 'Ein Fehler ist aufgetreten.'}
                    </AlertDescription>
                  </Alert>
                )}

                <Button
                  type="submit"
                  className="w-full"
                  size="lg"
                  disabled={checkInMutation.isPending || !code.trim()}
                >
                  {checkInMutation.isPending ? 'Wird verarbeitet...' : 'Jetzt einchecken'}
                </Button>

                <div className="text-center">
                  <Link href={`/t/${slug}`} className="text-sm text-muted-foreground hover:underline">
                    Zurück zur Turnierseite
                  </Link>
                </div>
              </form>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
