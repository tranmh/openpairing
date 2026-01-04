'use client';

import { useState, useEffect } from 'react';
import { useAuthStore } from '@/store/auth';
import { api } from '@/lib/api';
import { formatDate, getTitleDisplay, getTitleClass, formatRating } from '@/lib/utils';

interface PlayerProfile {
  id: string;
  firstName: string;
  lastName: string;
  birthDate?: string;
  gender?: string;
  fideId?: string;
  dsbId?: string;
  oesbId?: string;
  fideStandard?: number;
  fideRapid?: number;
  fideBlitz?: number;
  dwz?: number;
  dwzIndex?: number;
  oesbElo?: number;
  fideTitle?: string;
  nationalTitle?: string;
  clubName?: string;
  federation?: string;
}

export default function ProfilePage() {
  const { user } = useAuthStore();
  const [profile, setProfile] = useState<PlayerProfile | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isEditing, setIsEditing] = useState(false);
  const [editForm, setEditForm] = useState({
    firstName: '',
    lastName: '',
    fideId: '',
    dsbId: '',
  });
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  useEffect(() => {
    loadProfile();
  }, []);

  const loadProfile = async () => {
    setIsLoading(true);
    try {
      // TODO: Implement actual API call when backend endpoint is ready
      // const response = await api.get<PlayerProfile>('/players/me');
      // if (response.data) {
      //   setProfile(response.data);
      // }

      // Mock data for now
      setProfile({
        id: '1',
        firstName: 'Max',
        lastName: 'Mustermann',
        fideId: undefined,
        dsbId: undefined,
        fideStandard: undefined,
        dwz: undefined,
      });
    } catch (err) {
      setError('Profil konnte nicht geladen werden');
    } finally {
      setIsLoading(false);
    }
  };

  const handleEditSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    try {
      // TODO: Implement actual API call
      // await api.put('/players/me', editForm);
      setSuccess('Profil wurde aktualisiert');
      setIsEditing(false);
      loadProfile();
    } catch (err) {
      setError('Profil konnte nicht aktualisiert werden');
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Mein Profil</h1>
          <p className="mt-2 text-muted-foreground">
            Verwalten Sie Ihre Spielerdaten und Verbandsmitgliedschaften
          </p>
        </div>
        {!isEditing && (
          <button
            onClick={() => {
              setEditForm({
                firstName: profile?.firstName || '',
                lastName: profile?.lastName || '',
                fideId: profile?.fideId || '',
                dsbId: profile?.dsbId || '',
              });
              setIsEditing(true);
            }}
            className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
          >
            Bearbeiten
          </button>
        )}
      </div>

      {error && (
        <div className="rounded-md bg-destructive/10 p-4 text-sm text-destructive">
          {error}
        </div>
      )}

      {success && (
        <div className="rounded-md bg-green-500/10 p-4 text-sm text-green-600">
          {success}
        </div>
      )}

      {isEditing ? (
        <form onSubmit={handleEditSubmit} className="rounded-lg border bg-card p-6">
          <h2 className="text-lg font-semibold">Profil bearbeiten</h2>
          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <div>
              <label className="block text-sm font-medium">Vorname</label>
              <input
                type="text"
                value={editForm.firstName}
                onChange={(e) => setEditForm({ ...editForm, firstName: e.target.value })}
                className="mt-1 block w-full rounded-md border bg-background px-3 py-2"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium">Nachname</label>
              <input
                type="text"
                value={editForm.lastName}
                onChange={(e) => setEditForm({ ...editForm, lastName: e.target.value })}
                className="mt-1 block w-full rounded-md border bg-background px-3 py-2"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium">FIDE-ID</label>
              <input
                type="text"
                value={editForm.fideId}
                onChange={(e) => setEditForm({ ...editForm, fideId: e.target.value })}
                className="mt-1 block w-full rounded-md border bg-background px-3 py-2"
                placeholder="z.B. 12345678"
              />
              <p className="mt-1 text-xs text-muted-foreground">
                Verknüpfen Sie Ihr Profil mit Ihrer FIDE-ID
              </p>
            </div>
            <div>
              <label className="block text-sm font-medium">DSB-ID (PKZ)</label>
              <input
                type="text"
                value={editForm.dsbId}
                onChange={(e) => setEditForm({ ...editForm, dsbId: e.target.value })}
                className="mt-1 block w-full rounded-md border bg-background px-3 py-2"
                placeholder="z.B. 10012345"
              />
              <p className="mt-1 text-xs text-muted-foreground">
                Ihre Personenkennziffer des Deutschen Schachbunds
              </p>
            </div>
          </div>
          <div className="mt-6 flex justify-end gap-4">
            <button
              type="button"
              onClick={() => setIsEditing(false)}
              className="rounded-md border px-4 py-2 text-sm font-medium hover:bg-muted"
            >
              Abbrechen
            </button>
            <button
              type="submit"
              className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
            >
              Speichern
            </button>
          </div>
        </form>
      ) : (
        <div className="grid gap-6 lg:grid-cols-3">
          {/* Main Profile Card */}
          <div className="lg:col-span-2 rounded-lg border bg-card p-6">
            <div className="flex items-start gap-6">
              <div className="flex h-20 w-20 items-center justify-center rounded-full bg-primary/10">
                <span className="text-3xl font-bold text-primary">
                  {profile?.firstName?.charAt(0)}{profile?.lastName?.charAt(0)}
                </span>
              </div>
              <div>
                <div className="flex items-center gap-2">
                  {profile?.fideTitle && (
                    <span className={`title-badge ${getTitleClass(profile.fideTitle)}`}>
                      {getTitleDisplay(profile.fideTitle)}
                    </span>
                  )}
                  <h2 className="text-2xl font-bold">
                    {profile?.firstName} {profile?.lastName}
                  </h2>
                </div>
                <p className="mt-1 text-muted-foreground">{user?.email}</p>
                {profile?.clubName && (
                  <p className="mt-2 text-sm">{profile.clubName}</p>
                )}
              </div>
            </div>

            {/* Federation IDs */}
            <div className="mt-8">
              <h3 className="text-sm font-semibold text-muted-foreground">Verbands-IDs</h3>
              <div className="mt-4 grid gap-4 sm:grid-cols-3">
                <div className="rounded-lg bg-muted/50 p-4">
                  <p className="text-xs font-medium text-muted-foreground">FIDE-ID</p>
                  <p className="mt-1 font-mono text-lg">
                    {profile?.fideId || <span className="text-muted-foreground">-</span>}
                  </p>
                </div>
                <div className="rounded-lg bg-muted/50 p-4">
                  <p className="text-xs font-medium text-muted-foreground">DSB-ID (PKZ)</p>
                  <p className="mt-1 font-mono text-lg">
                    {profile?.dsbId || <span className="text-muted-foreground">-</span>}
                  </p>
                </div>
                <div className="rounded-lg bg-muted/50 p-4">
                  <p className="text-xs font-medium text-muted-foreground">ÖSB-ID</p>
                  <p className="mt-1 font-mono text-lg">
                    {profile?.oesbId || <span className="text-muted-foreground">-</span>}
                  </p>
                </div>
              </div>
            </div>
          </div>

          {/* Ratings Card */}
          <div className="rounded-lg border bg-card p-6">
            <h3 className="text-lg font-semibold">Wertungszahlen</h3>
            <div className="mt-4 space-y-4">
              <RatingItem
                label="FIDE Standard"
                value={profile?.fideStandard}
                className="rating-fide"
              />
              <RatingItem
                label="FIDE Rapid"
                value={profile?.fideRapid}
                className="rating-fide"
              />
              <RatingItem
                label="FIDE Blitz"
                value={profile?.fideBlitz}
                className="rating-fide"
              />
              <hr />
              <RatingItem
                label="DWZ"
                value={profile?.dwz}
                index={profile?.dwzIndex}
                className="rating-dwz"
              />
              <hr />
              <RatingItem
                label="ÖSB Elo"
                value={profile?.oesbElo}
                className="rating-oesb"
              />
            </div>
          </div>
        </div>
      )}

      {/* Statistics Section */}
      <div className="rounded-lg border bg-card p-6">
        <h3 className="text-lg font-semibold">Statistiken</h3>
        <div className="mt-4 grid gap-4 sm:grid-cols-4">
          <StatItem label="Turniere gespielt" value="0" />
          <StatItem label="Partien gespielt" value="0" />
          <StatItem label="Siege" value="0" />
          <StatItem label="Remis" value="0" />
        </div>
      </div>
    </div>
  );
}

function RatingItem({
  label,
  value,
  index,
  className,
}: {
  label: string;
  value?: number;
  index?: number;
  className?: string;
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span className={`rating-display font-semibold ${className}`}>
        {formatRating(value)}
        {index !== undefined && value && (
          <span className="ml-1 text-xs text-muted-foreground">({index})</span>
        )}
      </span>
    </div>
  );
}

function StatItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="text-center">
      <p className="text-3xl font-bold">{value}</p>
      <p className="mt-1 text-sm text-muted-foreground">{label}</p>
    </div>
  );
}
