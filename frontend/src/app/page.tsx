import Link from 'next/link';

export default function HomePage() {
  return (
    <div className="min-h-screen bg-gradient-to-b from-background to-muted">
      {/* Header */}
      <header className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="container mx-auto flex h-16 items-center justify-between px-4">
          <div className="flex items-center space-x-2">
            <span className="text-2xl font-bold">OpenPairing</span>
          </div>
          <nav className="flex items-center space-x-6">
            <Link href="/tournaments" className="text-sm font-medium hover:text-primary">
              Turniere
            </Link>
            <Link href="/players" className="text-sm font-medium hover:text-primary">
              Spieler
            </Link>
            <Link
              href="/login"
              className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
            >
              Anmelden
            </Link>
          </nav>
        </div>
      </header>

      {/* Hero Section */}
      <main className="container mx-auto px-4 py-16">
        <div className="mx-auto max-w-3xl text-center">
          <h1 className="text-4xl font-bold tracking-tight sm:text-6xl">
            Modernes Schachturnier-Management
          </h1>
          <p className="mt-6 text-lg leading-8 text-muted-foreground">
            Die Open-Source Plattform für Schachturniere im DACH-Raum. Swiss-System Paarungen mit
            JaVaFo, nahtlose FIDE/DWZ/ÖSB Integration und Mannschaftsturniere.
          </p>
          <div className="mt-10 flex items-center justify-center gap-x-6">
            <Link
              href="/register"
              className="rounded-md bg-primary px-6 py-3 text-sm font-semibold text-primary-foreground shadow-sm hover:bg-primary/90"
            >
              Jetzt starten
            </Link>
            <Link href="/tournaments" className="text-sm font-semibold leading-6">
              Turniere ansehen <span aria-hidden="true">→</span>
            </Link>
          </div>
        </div>

        {/* Features */}
        <div className="mx-auto mt-24 max-w-5xl">
          <div className="grid gap-8 md:grid-cols-3">
            <div className="rounded-lg border bg-card p-6 shadow-sm">
              <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10">
                <svg
                  className="h-6 w-6 text-primary"
                  fill="none"
                  viewBox="0 0 24 24"
                  strokeWidth={1.5}
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5"
                  />
                </svg>
              </div>
              <h3 className="text-lg font-semibold">Swiss-System Paarungen</h3>
              <p className="mt-2 text-sm text-muted-foreground">
                FIDE-konforme Paarungen mit JaVaFo und bbpPairings. Über 90 Feinwertungs-Optionen.
              </p>
            </div>

            <div className="rounded-lg border bg-card p-6 shadow-sm">
              <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10">
                <svg
                  className="h-6 w-6 text-primary"
                  fill="none"
                  viewBox="0 0 24 24"
                  strokeWidth={1.5}
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M12 21a9.004 9.004 0 008.716-6.747M12 21a9.004 9.004 0 01-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 017.843 4.582M12 3a8.997 8.997 0 00-7.843 4.582m15.686 0A11.953 11.953 0 0112 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0121 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0112 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 013 12c0-1.605.42-3.113 1.157-4.418"
                  />
                </svg>
              </div>
              <h3 className="text-lg font-semibold">DACH Integration</h3>
              <p className="mt-2 text-sm text-muted-foreground">
                Native DSB/DWZ und ÖSB Anbindung. Automatische Spieler-Synchronisation und
                DWZ-Auswertung.
              </p>
            </div>

            <div className="rounded-lg border bg-card p-6 shadow-sm">
              <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10">
                <svg
                  className="h-6 w-6 text-primary"
                  fill="none"
                  viewBox="0 0 24 24"
                  strokeWidth={1.5}
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z"
                  />
                </svg>
              </div>
              <h3 className="text-lg font-semibold">Mannschaftsturniere</h3>
              <p className="mt-2 text-sm text-muted-foreground">
                Vollständige Team-Unterstützung mit Aufstellungsverwaltung und
                Bundesliga-kompatiblen Regeln.
              </p>
            </div>
          </div>
        </div>

        {/* Stats */}
        <div className="mx-auto mt-24 max-w-4xl">
          <div className="grid gap-8 text-center md:grid-cols-4">
            <div>
              <p className="text-4xl font-bold text-primary">AGPL-3.0</p>
              <p className="mt-1 text-sm text-muted-foreground">Open Source</p>
            </div>
            <div>
              <p className="text-4xl font-bold text-primary">90+</p>
              <p className="mt-1 text-sm text-muted-foreground">Feinwertungen</p>
            </div>
            <div>
              <p className="text-4xl font-bold text-primary">FIDE</p>
              <p className="mt-1 text-sm text-muted-foreground">Endorsed Engines</p>
            </div>
            <div>
              <p className="text-4xl font-bold text-primary">DE/AT/CH</p>
              <p className="mt-1 text-sm text-muted-foreground">DACH Support</p>
            </div>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="mt-24 border-t bg-muted/50">
        <div className="container mx-auto px-4 py-12">
          <div className="grid gap-8 md:grid-cols-4">
            <div>
              <h4 className="text-lg font-semibold">OpenPairing</h4>
              <p className="mt-2 text-sm text-muted-foreground">
                Open-Source Schachturnier-Verwaltung für den DACH-Raum.
              </p>
            </div>
            <div>
              <h4 className="text-sm font-semibold">Produkt</h4>
              <ul className="mt-2 space-y-2 text-sm text-muted-foreground">
                <li>
                  <Link href="/features">Features</Link>
                </li>
                <li>
                  <Link href="/pricing">Preise</Link>
                </li>
                <li>
                  <Link href="/docs">Dokumentation</Link>
                </li>
              </ul>
            </div>
            <div>
              <h4 className="text-sm font-semibold">Ressourcen</h4>
              <ul className="mt-2 space-y-2 text-sm text-muted-foreground">
                <li>
                  <a href="https://github.com/openpairing/openpairing">GitHub</a>
                </li>
                <li>
                  <Link href="/api">API</Link>
                </li>
                <li>
                  <Link href="/support">Support</Link>
                </li>
              </ul>
            </div>
            <div>
              <h4 className="text-sm font-semibold">Legal</h4>
              <ul className="mt-2 space-y-2 text-sm text-muted-foreground">
                <li>
                  <Link href="/privacy">Datenschutz</Link>
                </li>
                <li>
                  <Link href="/terms">AGB</Link>
                </li>
                <li>
                  <Link href="/imprint">Impressum</Link>
                </li>
              </ul>
            </div>
          </div>
          <div className="mt-8 border-t pt-8 text-center text-sm text-muted-foreground">
            <p>&copy; {new Date().getFullYear()} OpenPairing. Licensed under AGPL-3.0.</p>
          </div>
        </div>
      </footer>
    </div>
  );
}
