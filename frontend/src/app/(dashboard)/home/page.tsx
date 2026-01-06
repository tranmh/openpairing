'use client';

import Link from 'next/link';
import { useAuthStore } from '@/store/auth';

export default function DashboardPage() {
  const { user } = useAuthStore();

  return (
    <div className="space-y-8">
      {/* Welcome Section */}
      <div>
        <h1 className="text-3xl font-bold">Willkommen zurück!</h1>
        <p className="mt-2 text-muted-foreground">
          Verwalten Sie Ihre Turniere und Ergebnisse von hier aus.
        </p>
      </div>

      {/* Quick Stats */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Aktive Turniere"
          value="0"
          description="Turniere in denen Sie teilnehmen"
          href="/tournaments"
        />
        <StatCard
          title="Gespielte Partien"
          value="0"
          description="Gesamtzahl Ihrer Partien"
          href="/tournaments"
        />
        <StatCard
          title="Aktuelle Wertung"
          value="-"
          description="Ihre höchste Wertungszahl"
          href="/profile"
        />
        <StatCard
          title="Nächste Runde"
          value="-"
          description="Keine anstehenden Partien"
          href="/tournaments"
        />
      </div>

      {/* Quick Actions */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <QuickActionCard
          title="Turnier suchen"
          description="Finden und melden Sie sich für Turniere an"
          href="/tournaments"
          icon={SearchIcon}
        />
        <QuickActionCard
          title="Profil bearbeiten"
          description="Aktualisieren Sie Ihre persönlichen Daten"
          href="/profile"
          icon={UserIcon}
        />
        {(user?.role === 'organizer' || user?.role === 'admin') && (
          <QuickActionCard
            title="Turnier erstellen"
            description="Neues Turnier erstellen und verwalten"
            href="/tournaments/new"
            icon={PlusIcon}
          />
        )}
        {(user?.role === 'arbiter' || user?.role === 'organizer' || user?.role === 'admin') && (
          <QuickActionCard
            title="Schiedsrichter-Konsole"
            description="Turniere als Schiedsrichter verwalten"
            href="/tournaments"
            icon={ClipboardIcon}
          />
        )}
      </div>

      {/* Recent Activity */}
      <div className="rounded-lg border bg-card p-6">
        <h2 className="text-lg font-semibold">Letzte Aktivitäten</h2>
        <div className="mt-4 flex flex-col items-center justify-center py-8 text-center">
          <div className="rounded-full bg-muted p-4">
            <ClockIcon className="h-8 w-8 text-muted-foreground" />
          </div>
          <p className="mt-4 text-muted-foreground">Noch keine Aktivitäten vorhanden</p>
          <Link
            href="/tournaments"
            className="mt-4 text-sm font-medium text-primary hover:underline"
          >
            Erstes Turnier finden →
          </Link>
        </div>
      </div>
    </div>
  );
}

function StatCard({
  title,
  value,
  description,
  href,
}: {
  title: string;
  value: string;
  description: string;
  href: string;
}) {
  return (
    <Link
      href={href}
      className="block rounded-lg border bg-card p-6 transition-colors hover:bg-muted/50"
    >
      <p className="text-sm font-medium text-muted-foreground">{title}</p>
      <p className="mt-2 text-3xl font-bold">{value}</p>
      <p className="mt-1 text-xs text-muted-foreground">{description}</p>
    </Link>
  );
}

function QuickActionCard({
  title,
  description,
  href,
  icon: Icon,
}: {
  title: string;
  description: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
}) {
  return (
    <Link
      href={href}
      className="flex items-start gap-4 rounded-lg border bg-card p-6 transition-colors hover:bg-muted/50"
    >
      <div className="rounded-lg bg-primary/10 p-2">
        <Icon className="h-6 w-6 text-primary" />
      </div>
      <div>
        <h3 className="font-semibold">{title}</h3>
        <p className="mt-1 text-sm text-muted-foreground">{description}</p>
      </div>
    </Link>
  );
}

// Icons
function SearchIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z" />
    </svg>
  );
}

function UserIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 6a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0ZM4.501 20.118a7.5 7.5 0 0 1 14.998 0A17.933 17.933 0 0 1 12 21.75c-2.676 0-5.216-.584-7.499-1.632Z" />
    </svg>
  );
}

function PlusIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
    </svg>
  );
}

function ClipboardIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 0 0 2.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 0 0-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 0 0 .75-.75 2.25 2.25 0 0 0-.1-.664m-5.8 0A2.251 2.251 0 0 1 13.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V19.5a2.25 2.25 0 0 0 2.25 2.25h.75m0-3H12m-.75 3h.008v.008h-.008v-.008Zm.375 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Z" />
    </svg>
  );
}

function ClockIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
    </svg>
  );
}
