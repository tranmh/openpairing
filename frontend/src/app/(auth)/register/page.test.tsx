import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';

const mockPush = vi.fn();

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockPush, replace: vi.fn(), prefetch: vi.fn() }),
  usePathname: () => '/register',
}));

const mockStore = {
  register: vi.fn(),
  clearError: vi.fn(),
  isLoading: false,
  error: null as string | null,
};

vi.mock('@/store/auth', () => ({
  useAuthStore: () => mockStore,
}));

const redirectState = { isAuthenticated: false, isLoading: false, isHydrated: true };

vi.mock('@/hooks/useAuth', () => ({
  useRedirectIfAuthenticated: () => redirectState,
}));

import RegisterPage from './page';

const VALID_PASSWORD = 'Sicher123';

function fill(values: Partial<Record<string, string | boolean>> = {}) {
  const v = {
    firstName: 'Max',
    lastName: 'Mustermann',
    email: 'max@beispiel.de',
    password: VALID_PASSWORD,
    confirmPassword: VALID_PASSWORD,
    acceptTerms: true,
    ...values,
  };
  fireEvent.change(screen.getByLabelText('Vorname'), { target: { value: v.firstName } });
  fireEvent.change(screen.getByLabelText('Nachname'), { target: { value: v.lastName } });
  fireEvent.change(screen.getByLabelText('E-Mail'), { target: { value: v.email } });
  fireEvent.change(screen.getByLabelText('Passwort'), { target: { value: v.password } });
  fireEvent.change(screen.getByLabelText('Passwort bestätigen'), {
    target: { value: v.confirmPassword },
  });
  const terms = screen.getByRole('checkbox') as HTMLInputElement;
  if (terms.checked !== v.acceptTerms) {
    fireEvent.click(terms);
  }
}

function submit() {
  // Submit the form directly so that the component's own validation is tested,
  // not the browser's constraint validation.
  const form = screen.getByRole('button', { name: /registrieren/i }).closest('form')!;
  fireEvent.submit(form);
}

describe('RegisterPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockStore.isLoading = false;
    mockStore.error = null;
    mockStore.register.mockResolvedValue(true);
    redirectState.isAuthenticated = false;
  });

  it('renders the registration form', () => {
    render(<RegisterPage />);
    expect(screen.getByRole('heading', { name: 'Konto erstellen' })).toBeInTheDocument();
    expect(screen.getByLabelText('Vorname')).toBeInTheDocument();
    expect(screen.getByLabelText('Nachname')).toBeInTheDocument();
    expect(screen.getByLabelText('E-Mail')).toBeInTheDocument();
    expect(screen.getByLabelText('Passwort')).toBeInTheDocument();
    expect(screen.getByLabelText('Passwort bestätigen')).toBeInTheDocument();
    expect(screen.getByRole('checkbox')).not.toBeChecked();
    expect(screen.getByRole('button', { name: 'Registrieren' })).toBeEnabled();
    expect(screen.getByRole('link', { name: 'Jetzt anmelden' })).toHaveAttribute('href', '/login');
  });

  it('clears a stale store error on mount', () => {
    render(<RegisterPage />);
    expect(mockStore.clearError).toHaveBeenCalled();
  });

  it('registers with the entered data and redirects to the login page', async () => {
    render(<RegisterPage />);
    fill();
    submit();

    await waitFor(() => expect(mockPush).toHaveBeenCalledWith('/login?registered=true'));
    expect(mockStore.register).toHaveBeenCalledTimes(1);
    expect(mockStore.register).toHaveBeenCalledWith({
      email: 'max@beispiel.de',
      password: VALID_PASSWORD,
      firstName: 'Max',
      lastName: 'Mustermann',
    });
    // BUG-18 regression (B17): never redirect to the non-existent dashboard
    expect(mockPush).not.toHaveBeenCalledWith(expect.stringContaining('/dashboard'));
  });

  it('does not redirect when the registration fails', async () => {
    mockStore.register.mockResolvedValue(false);
    render(<RegisterPage />);
    fill();
    submit();

    await waitFor(() => expect(mockStore.register).toHaveBeenCalled());
    expect(mockPush).not.toHaveBeenCalled();
  });

  it('shows the server error from the store', () => {
    mockStore.error = 'E-Mail bereits registriert';
    render(<RegisterPage />);
    expect(screen.getByText('E-Mail bereits registriert')).toBeInTheDocument();
  });

  it('shows the validation error instead of the server error', () => {
    mockStore.error = 'E-Mail bereits registriert';
    render(<RegisterPage />);
    fill({ confirmPassword: 'Anders123' });
    submit();
    expect(screen.getByText('Die Passwörter stimmen nicht überein')).toBeInTheDocument();
    expect(screen.queryByText('E-Mail bereits registriert')).not.toBeInTheDocument();
  });

  it.each([
    ['mismatching passwords', { confirmPassword: 'Sicher124' }, 'Die Passwörter stimmen nicht überein'],
    [
      'a 7 character password',
      { password: 'Abcde12', confirmPassword: 'Abcde12' },
      'Das Passwort muss mindestens 8 Zeichen lang sein',
    ],
    [
      'an empty password',
      { password: '', confirmPassword: '' },
      'Das Passwort muss mindestens 8 Zeichen lang sein',
    ],
    [
      'a password without uppercase letter',
      { password: 'sicher123', confirmPassword: 'sicher123' },
      'Das Passwort muss mindestens einen Großbuchstaben enthalten',
    ],
    [
      'a password without lowercase letter',
      { password: 'SICHER123', confirmPassword: 'SICHER123' },
      'Das Passwort muss mindestens einen Kleinbuchstaben enthalten',
    ],
    [
      'a password without digit',
      { password: 'SicherAbc', confirmPassword: 'SicherAbc' },
      'Das Passwort muss mindestens eine Zahl enthalten',
    ],
    ['unaccepted terms', { acceptTerms: false }, 'Bitte akzeptieren Sie die AGB'],
  ] as const)('rejects %s', (_label, values, message) => {
    render(<RegisterPage />);
    fill(values);
    submit();
    expect(screen.getByText(message)).toBeInTheDocument();
    expect(mockStore.register).not.toHaveBeenCalled();
    expect(mockPush).not.toHaveBeenCalled();
  });

  it('accepts a password of exactly 8 characters', async () => {
    render(<RegisterPage />);
    fill({ password: 'Abcdef12', confirmPassword: 'Abcdef12' });
    submit();
    await waitFor(() => expect(mockStore.register).toHaveBeenCalled());
    expect(mockStore.register.mock.calls[0][0]).toMatchObject({ password: 'Abcdef12' });
  });

  it('clears the validation error once the user edits a field', () => {
    render(<RegisterPage />);
    fill({ confirmPassword: 'Anders123' });
    submit();
    expect(screen.getByText('Die Passwörter stimmen nicht überein')).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText('Passwort bestätigen'), {
      target: { value: VALID_PASSWORD },
    });
    expect(screen.queryByText('Die Passwörter stimmen nicht überein')).not.toBeInTheDocument();
  });

  it('disables the inputs and shows progress while loading', () => {
    mockStore.isLoading = true;
    render(<RegisterPage />);
    expect(screen.getByText('Wird registriert...')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /wird registriert/i })).toBeDisabled();
    expect(screen.getByLabelText('E-Mail')).toBeDisabled();
    expect(screen.getByLabelText('Passwort')).toBeDisabled();
  });

  it('renders nothing for an already authenticated user', () => {
    redirectState.isAuthenticated = true;
    const { container } = render(<RegisterPage />);
    expect(container).toBeEmptyDOMElement();
  });
});
