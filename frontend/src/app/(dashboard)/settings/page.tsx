'use client';

import { useState } from 'react';
import { useAuthStore } from '@/store/auth';

export default function SettingsPage() {
  const { user } = useAuthStore();
  const [activeTab, setActiveTab] = useState<'account' | 'notifications' | 'security'>('account');
  const [isLoading, setIsLoading] = useState(false);
  const [success, setSuccess] = useState('');
  const [error, setError] = useState('');

  // Account settings
  const [email, setEmail] = useState(user?.email || '');
  const [language, setLanguage] = useState(user?.preferredLanguage || 'de');

  // Notification settings
  const [notifications, setNotifications] = useState({
    emailPairings: true,
    emailResults: true,
    emailTournaments: true,
    pushEnabled: false,
  });

  // Security settings
  const [passwordForm, setPasswordForm] = useState({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  });

  const handleAccountSave = async () => {
    setIsLoading(true);
    setError('');
    setSuccess('');

    try {
      // TODO: Implement API call
      await new Promise((resolve) => setTimeout(resolve, 500));
      setSuccess('Einstellungen wurden gespeichert');
    } catch (err) {
      setError('Einstellungen konnten nicht gespeichert werden');
    } finally {
      setIsLoading(false);
    }
  };

  const handlePasswordChange = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      setError('Die Passwörter stimmen nicht überein');
      return;
    }

    if (passwordForm.newPassword.length < 8) {
      setError('Das neue Passwort muss mindestens 8 Zeichen lang sein');
      return;
    }

    setIsLoading(true);
    try {
      // TODO: Implement API call
      await new Promise((resolve) => setTimeout(resolve, 500));
      setSuccess('Passwort wurde geändert');
      setPasswordForm({ currentPassword: '', newPassword: '', confirmPassword: '' });
    } catch (err) {
      setError('Passwort konnte nicht geändert werden');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold">Einstellungen</h1>
        <p className="mt-2 text-muted-foreground">
          Verwalten Sie Ihre Kontoeinstellungen und Präferenzen
        </p>
      </div>

      {/* Tabs */}
      <div className="border-b">
        <nav className="-mb-px flex gap-6">
          <TabButton
            active={activeTab === 'account'}
            onClick={() => setActiveTab('account')}
          >
            Konto
          </TabButton>
          <TabButton
            active={activeTab === 'notifications'}
            onClick={() => setActiveTab('notifications')}
          >
            Benachrichtigungen
          </TabButton>
          <TabButton
            active={activeTab === 'security'}
            onClick={() => setActiveTab('security')}
          >
            Sicherheit
          </TabButton>
        </nav>
      </div>

      {/* Messages */}
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

      {/* Account Tab */}
      {activeTab === 'account' && (
        <div className="rounded-lg border bg-card p-6">
          <h2 className="text-lg font-semibold">Kontoeinstellungen</h2>
          <div className="mt-6 space-y-6">
            <div className="max-w-md">
              <label className="block text-sm font-medium">E-Mail-Adresse</label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="mt-1 block w-full rounded-md border bg-background px-3 py-2"
              />
              <p className="mt-1 text-xs text-muted-foreground">
                Ihre E-Mail-Adresse für die Anmeldung und Benachrichtigungen
              </p>
            </div>

            <div className="max-w-md">
              <label className="block text-sm font-medium">Sprache</label>
              <select
                value={language}
                onChange={(e) => setLanguage(e.target.value)}
                className="mt-1 block w-full rounded-md border bg-background px-3 py-2"
              >
                <option value="de">Deutsch</option>
                <option value="en">English</option>
              </select>
              <p className="mt-1 text-xs text-muted-foreground">
                Sprache für die Benutzeroberfläche
              </p>
            </div>

            <div className="pt-4">
              <button
                onClick={handleAccountSave}
                disabled={isLoading}
                className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                {isLoading ? 'Wird gespeichert...' : 'Speichern'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Notifications Tab */}
      {activeTab === 'notifications' && (
        <div className="rounded-lg border bg-card p-6">
          <h2 className="text-lg font-semibold">Benachrichtigungseinstellungen</h2>
          <div className="mt-6 space-y-4">
            <NotificationToggle
              label="Paarungen per E-Mail"
              description="Erhalten Sie eine E-Mail, wenn neue Paarungen veröffentlicht werden"
              checked={notifications.emailPairings}
              onChange={(checked) => setNotifications({ ...notifications, emailPairings: checked })}
            />
            <NotificationToggle
              label="Ergebnisse per E-Mail"
              description="Erhalten Sie eine E-Mail, wenn Ihre Ergebnisse eingetragen werden"
              checked={notifications.emailResults}
              onChange={(checked) => setNotifications({ ...notifications, emailResults: checked })}
            />
            <NotificationToggle
              label="Turnier-Updates"
              description="Erhalten Sie Updates zu Turnieren, für die Sie sich angemeldet haben"
              checked={notifications.emailTournaments}
              onChange={(checked) => setNotifications({ ...notifications, emailTournaments: checked })}
            />
            <hr />
            <NotificationToggle
              label="Push-Benachrichtigungen"
              description="Browser-Benachrichtigungen für wichtige Updates (kommt bald)"
              checked={notifications.pushEnabled}
              onChange={(checked) => setNotifications({ ...notifications, pushEnabled: checked })}
              disabled
            />

            <div className="pt-4">
              <button
                onClick={handleAccountSave}
                disabled={isLoading}
                className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                {isLoading ? 'Wird gespeichert...' : 'Speichern'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Security Tab */}
      {activeTab === 'security' && (
        <div className="space-y-6">
          {/* Change Password */}
          <div className="rounded-lg border bg-card p-6">
            <h2 className="text-lg font-semibold">Passwort ändern</h2>
            <form onSubmit={handlePasswordChange} className="mt-6 max-w-md space-y-4">
              <div>
                <label className="block text-sm font-medium">Aktuelles Passwort</label>
                <input
                  type="password"
                  value={passwordForm.currentPassword}
                  onChange={(e) => setPasswordForm({ ...passwordForm, currentPassword: e.target.value })}
                  className="mt-1 block w-full rounded-md border bg-background px-3 py-2"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium">Neues Passwort</label>
                <input
                  type="password"
                  value={passwordForm.newPassword}
                  onChange={(e) => setPasswordForm({ ...passwordForm, newPassword: e.target.value })}
                  className="mt-1 block w-full rounded-md border bg-background px-3 py-2"
                  minLength={8}
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium">Neues Passwort bestätigen</label>
                <input
                  type="password"
                  value={passwordForm.confirmPassword}
                  onChange={(e) => setPasswordForm({ ...passwordForm, confirmPassword: e.target.value })}
                  className="mt-1 block w-full rounded-md border bg-background px-3 py-2"
                  required
                />
              </div>
              <button
                type="submit"
                disabled={isLoading}
                className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                {isLoading ? 'Wird geändert...' : 'Passwort ändern'}
              </button>
            </form>
          </div>

          {/* Sessions */}
          <div className="rounded-lg border bg-card p-6">
            <h2 className="text-lg font-semibold">Aktive Sitzungen</h2>
            <p className="mt-2 text-sm text-muted-foreground">
              Verwalten Sie Ihre angemeldeten Geräte
            </p>
            <div className="mt-4 rounded-lg border p-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <div className="rounded-lg bg-muted p-2">
                    <MonitorIcon className="h-5 w-5" />
                  </div>
                  <div>
                    <p className="font-medium">Aktuelles Gerät</p>
                    <p className="text-sm text-muted-foreground">
                      Diese Sitzung • Zuletzt aktiv: Jetzt
                    </p>
                  </div>
                </div>
                <span className="rounded-full bg-green-500/10 px-2 py-1 text-xs font-medium text-green-600">
                  Aktiv
                </span>
              </div>
            </div>
            <button className="mt-4 text-sm font-medium text-destructive hover:underline">
              Alle anderen Sitzungen abmelden
            </button>
          </div>

          {/* Danger Zone */}
          <div className="rounded-lg border border-destructive/50 bg-card p-6">
            <h2 className="text-lg font-semibold text-destructive">Gefahrenzone</h2>
            <p className="mt-2 text-sm text-muted-foreground">
              Diese Aktionen können nicht rückgängig gemacht werden
            </p>
            <button className="mt-4 rounded-md border border-destructive px-4 py-2 text-sm font-medium text-destructive hover:bg-destructive/10">
              Konto löschen
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      className={`border-b-2 py-3 text-sm font-medium transition-colors ${
        active
          ? 'border-primary text-primary'
          : 'border-transparent text-muted-foreground hover:text-foreground'
      }`}
    >
      {children}
    </button>
  );
}

function NotificationToggle({
  label,
  description,
  checked,
  onChange,
  disabled,
}: {
  label: string;
  description: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <div className="flex items-start justify-between">
      <div>
        <p className="font-medium">{label}</p>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 ${
          checked ? 'bg-primary' : 'bg-muted'
        }`}
      >
        <span
          className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition-transform ${
            checked ? 'translate-x-5' : 'translate-x-0'
          }`}
        />
      </button>
    </div>
  );
}

function MonitorIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" d="M9 17.25v1.007a3 3 0 0 1-.879 2.122L7.5 21h9l-.621-.621A3 3 0 0 1 15 18.257V17.25m6-12V15a2.25 2.25 0 0 1-2.25 2.25H5.25A2.25 2.25 0 0 1 3 15V5.25m18 0A2.25 2.25 0 0 0 18.75 3H5.25A2.25 2.25 0 0 0 3 5.25m18 0V12a2.25 2.25 0 0 1-2.25 2.25H5.25A2.25 2.25 0 0 1 3 12V5.25" />
    </svg>
  );
}
