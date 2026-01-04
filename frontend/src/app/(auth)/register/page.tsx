'use client';

import Link from 'next/link';
import { useState } from 'react';

export default function RegisterPage() {
  const [formData, setFormData] = useState({
    email: '',
    password: '',
    confirmPassword: '',
    firstName: '',
    lastName: '',
    acceptTerms: false,
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value, type, checked } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value,
    }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError('');

    // Validation
    if (formData.password !== formData.confirmPassword) {
      setError('Die Passwörter stimmen nicht überein');
      setIsLoading(false);
      return;
    }

    if (formData.password.length < 8) {
      setError('Das Passwort muss mindestens 8 Zeichen lang sein');
      setIsLoading(false);
      return;
    }

    if (!formData.acceptTerms) {
      setError('Bitte akzeptieren Sie die AGB');
      setIsLoading(false);
      return;
    }

    try {
      // TODO: Implement actual registration API call
      console.log('Registration attempt:', formData);
      // Simulate API call
      await new Promise((resolve) => setTimeout(resolve, 1000));
      // Redirect to login on success
      window.location.href = '/login?registered=true';
    } catch (err) {
      setError('Registrierung fehlgeschlagen. Bitte versuchen Sie es erneut.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/50 px-4 py-12">
      <div className="w-full max-w-md">
        <div className="rounded-lg border bg-card p-8 shadow-sm">
          <div className="mb-8 text-center">
            <Link href="/" className="text-2xl font-bold">
              OpenPairing
            </Link>
            <h1 className="mt-4 text-xl font-semibold">Konto erstellen</h1>
            <p className="mt-2 text-sm text-muted-foreground">
              Registrieren Sie sich für OpenPairing
            </p>
          </div>

          {error && (
            <div className="mb-4 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label htmlFor="firstName" className="block text-sm font-medium">
                  Vorname
                </label>
                <input
                  id="firstName"
                  name="firstName"
                  type="text"
                  value={formData.firstName}
                  onChange={handleChange}
                  className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  placeholder="Max"
                  required
                />
              </div>
              <div>
                <label htmlFor="lastName" className="block text-sm font-medium">
                  Nachname
                </label>
                <input
                  id="lastName"
                  name="lastName"
                  type="text"
                  value={formData.lastName}
                  onChange={handleChange}
                  className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  placeholder="Mustermann"
                  required
                />
              </div>
            </div>

            <div>
              <label htmlFor="email" className="block text-sm font-medium">
                E-Mail
              </label>
              <input
                id="email"
                name="email"
                type="email"
                value={formData.email}
                onChange={handleChange}
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
                name="password"
                type="password"
                value={formData.password}
                onChange={handleChange}
                className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                placeholder="Mindestens 8 Zeichen"
                required
                minLength={8}
              />
            </div>

            <div>
              <label htmlFor="confirmPassword" className="block text-sm font-medium">
                Passwort bestätigen
              </label>
              <input
                id="confirmPassword"
                name="confirmPassword"
                type="password"
                value={formData.confirmPassword}
                onChange={handleChange}
                className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                placeholder="Passwort wiederholen"
                required
              />
            </div>

            <div className="flex items-start">
              <input
                id="acceptTerms"
                name="acceptTerms"
                type="checkbox"
                checked={formData.acceptTerms}
                onChange={handleChange}
                className="mt-1 h-4 w-4 rounded border-input text-primary focus:ring-primary"
                required
              />
              <label htmlFor="acceptTerms" className="ml-2 text-sm text-muted-foreground">
                Ich akzeptiere die{' '}
                <Link href="/terms" className="text-primary hover:text-primary/80">
                  AGB
                </Link>{' '}
                und{' '}
                <Link href="/privacy" className="text-primary hover:text-primary/80">
                  Datenschutzerklärung
                </Link>
              </label>
            </div>

            <button
              type="submit"
              disabled={isLoading}
              className="w-full rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {isLoading ? 'Wird registriert...' : 'Registrieren'}
            </button>
          </form>

          <div className="mt-6 text-center text-sm text-muted-foreground">
            Bereits ein Konto?{' '}
            <Link href="/login" className="text-primary hover:text-primary/80">
              Jetzt anmelden
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
