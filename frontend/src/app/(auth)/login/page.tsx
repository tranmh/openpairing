'use client';

import Link from 'next/link';
import { useState } from 'react';

export default function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError('');

    try {
      // TODO: Implement actual login API call
      console.log('Login attempt:', { email, password });
      // Simulate API call
      await new Promise((resolve) => setTimeout(resolve, 1000));
      // Redirect to dashboard on success
      window.location.href = '/dashboard';
    } catch (err) {
      setError('Ungültige E-Mail oder Passwort');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/50 px-4">
      <div className="w-full max-w-md">
        <div className="rounded-lg border bg-card p-8 shadow-sm">
          <div className="mb-8 text-center">
            <Link href="/" className="text-2xl font-bold">
              OpenPairing
            </Link>
            <h1 className="mt-4 text-xl font-semibold">Anmelden</h1>
            <p className="mt-2 text-sm text-muted-foreground">
              Melden Sie sich mit Ihrem Konto an
            </p>
          </div>

          {error && (
            <div className="mb-4 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label htmlFor="email" className="block text-sm font-medium">
                E-Mail
              </label>
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                placeholder="name@beispiel.de"
                required
              />
            </div>

            <div>
              <label htmlFor="password" className="block text-sm font-medium">
                Passwort
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                placeholder="••••••••"
                required
              />
            </div>

            <div className="flex items-center justify-between">
              <label className="flex items-center">
                <input
                  type="checkbox"
                  className="h-4 w-4 rounded border-input text-primary focus:ring-primary"
                />
                <span className="ml-2 text-sm text-muted-foreground">Angemeldet bleiben</span>
              </label>
              <Link
                href="/forgot-password"
                className="text-sm text-primary hover:text-primary/80"
              >
                Passwort vergessen?
              </Link>
            </div>

            <button
              type="submit"
              disabled={isLoading}
              className="w-full rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {isLoading ? 'Wird angemeldet...' : 'Anmelden'}
            </button>
          </form>

          <div className="mt-6 text-center text-sm text-muted-foreground">
            Noch kein Konto?{' '}
            <Link href="/register" className="text-primary hover:text-primary/80">
              Jetzt registrieren
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
