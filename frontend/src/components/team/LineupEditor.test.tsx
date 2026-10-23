import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, within, fireEvent, act } from '@testing-library/react';
import type { ReactNode } from 'react';
import { LineupEditor, type TeamLineup, type LineupEntry } from './LineupEditor';
import type { TeamMember } from './RosterManager';

// Radix Select relies on pointer capture and portals that jsdom does not support.
// Replace it with a minimal, accessible stand-in that keeps the value/onValueChange contract.
vi.mock('@/components/ui/select', async () => {
  const React = await import('react');
  const Ctx = React.createContext<{ value: string; onValueChange: (v: string) => void }>({
    value: '',
    onValueChange: () => {},
  });
  return {
    Select: ({
      value,
      onValueChange,
      children,
    }: {
      value: string;
      onValueChange: (v: string) => void;
      children: ReactNode;
    }) => (
      <Ctx.Provider value={{ value, onValueChange }}>
        <div data-testid="board-select" data-value={value}>
          {children}
        </div>
      </Ctx.Provider>
    ),
    SelectTrigger: ({ children }: { children: ReactNode }) => <div>{children}</div>,
    SelectValue: ({ placeholder, children }: { placeholder?: string; children?: ReactNode }) => (
      <span data-testid="select-value">{children || placeholder}</span>
    ),
    SelectContent: ({ children }: { children: ReactNode }) => (
      <div role="listbox">{children}</div>
    ),
    SelectItem: ({ value, children }: { value: string; children: ReactNode }) => {
      const ctx = React.useContext(Ctx);
      return (
        <button
          type="button"
          role="option"
          aria-selected={ctx.value === value}
          onClick={() => ctx.onValueChange(value)}
        >
          {children}
        </button>
      );
    },
  };
});

function member(
  playerId: string,
  playerName: string,
  playerRating: number,
  extra: Partial<TeamMember> = {}
): TeamMember {
  return {
    id: `m-${playerId}`,
    teamId: 'team-1',
    playerId,
    isCaptain: false,
    isReserve: false,
    createdAt: '2024-01-01T00:00:00Z',
    playerName,
    playerRating,
    ...extra,
  };
}

const roster: TeamMember[] = [
  member('p1', 'Anna Adler', 2200, { playerTitle: 'FM' }),
  member('p2', 'Bert Berg', 2100),
  member('p3', 'Carl Christ', 2000),
  member('p4', 'Dora Dorn', 1900),
  member('p5', 'Emil Ernst', 2300, { isReserve: true }),
];

function lineupOf(entries: LineupEntry[], extra: Partial<TeamLineup> = {}): TeamLineup {
  return {
    id: 'l1',
    tournamentId: 't1',
    roundId: 'r1',
    teamParticipantId: 'tp1',
    lineup: entries,
    isValid: true,
    ...extra,
  };
}

const e = (board: number, playerId: string, rating: number): LineupEntry => ({
  board,
  playerId,
  rating,
});

type Props = Partial<React.ComponentProps<typeof LineupEditor>>;

function renderEditor(props: Props = {}) {
  const onSubmit = props.onSubmit ?? vi.fn().mockResolvedValue(undefined);
  const utils = render(
    <LineupEditor
      roundId="r1"
      teamParticipantId="tp1"
      boardsPerMatch={4}
      roster={roster}
      {...props}
      onSubmit={onSubmit}
    />
  );
  return { ...utils, onSubmit };
}

function boards() {
  return screen.getAllByTestId('board-select');
}

function pick(boardIndex: number, name: string) {
  const board = boards()[boardIndex];
  fireEvent.click(within(board).getByRole('option', { name: new RegExp(name) }));
}

function submitButton() {
  return screen.getByRole('button', { name: /submit lineup|update lineup|submitting/i });
}

describe('LineupEditor', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('initial state', () => {
    it('renders one selector per board and requires all boards to be filled', () => {
      renderEditor();
      expect(boards()).toHaveLength(4);
      expect(screen.getByText('Select 4 players for the upcoming match')).toBeInTheDocument();
      expect(screen.getByText('Validation Errors')).toBeInTheDocument();
      expect(screen.getByText('All 4 boards must be filled')).toBeInTheDocument();
      expect(submitButton()).toBeDisabled();
    });

    it('offers only non-reserve players, sorted by rating descending', () => {
      renderEditor();
      const options = within(boards()[0]).getAllByRole('option');
      expect(options.map((o) => o.textContent)).toEqual([
        'FMAnna Adler(2200)',
        'Bert Berg(2100)',
        'Carl Christ(2000)',
        'Dora Dorn(1900)',
      ]);
      expect(within(boards()[0]).queryByRole('option', { name: /Emil Ernst/ })).toBeNull();
    });

    it('works with a single-board match', () => {
      renderEditor({ boardsPerMatch: 1 });
      expect(boards()).toHaveLength(1);
      expect(screen.getByText('All 1 boards must be filled')).toBeInTheDocument();
    });
  });

  describe('selecting players', () => {
    it('becomes valid when all boards are filled in rating order and submits the lineup', async () => {
      const { onSubmit } = renderEditor();
      pick(0, 'Anna Adler');
      pick(1, 'Bert Berg');
      pick(2, 'Carl Christ');
      pick(3, 'Dora Dorn');

      expect(screen.getByText('Lineup Valid')).toBeInTheDocument();
      expect(screen.getByText('Ready to submit')).toBeInTheDocument();
      expect(submitButton()).toBeEnabled();

      await act(async () => {
        fireEvent.click(submitButton());
      });

      expect(onSubmit).toHaveBeenCalledTimes(1);
      expect(onSubmit).toHaveBeenCalledWith([
        e(1, 'p1', 2200),
        e(2, 'p2', 2100),
        e(3, 'p3', 2000),
        e(4, 'p4', 1900),
      ]);
    });

    it('shows the selected player name and rating on the board', () => {
      renderEditor();
      pick(0, 'Bert Berg');
      expect(within(boards()[0]).getByTestId('select-value')).toHaveTextContent('Bert Berg (2100)');
      expect(within(boards()[1]).getByTestId('select-value')).toHaveTextContent('Select player...');
    });

    it('does not offer a player already placed on another board', () => {
      renderEditor();
      pick(0, 'Anna Adler');
      expect(within(boards()[1]).queryByRole('option', { name: /Anna Adler/ })).toBeNull();
      // still selectable on its own board
      expect(within(boards()[0]).getByRole('option', { name: /Anna Adler/ })).toBeInTheDocument();
    });

    it('offers "Clear selection" only on filled boards and clears the board', () => {
      renderEditor();
      expect(within(boards()[0]).queryByRole('option', { name: 'Clear selection' })).toBeNull();
      pick(0, 'Anna Adler');
      fireEvent.click(within(boards()[0]).getByRole('option', { name: 'Clear selection' }));
      expect(within(boards()[0]).getByTestId('select-value')).toHaveTextContent('Select player...');
      expect(within(boards()[1]).getByRole('option', { name: /Anna Adler/ })).toBeInTheDocument();
    });

    it('invalidates the lineup again after clearing a board', () => {
      renderEditor({ boardsPerMatch: 2 });
      pick(0, 'Anna Adler');
      pick(1, 'Bert Berg');
      expect(screen.getByText('Lineup Valid')).toBeInTheDocument();
      fireEvent.click(within(boards()[1]).getByRole('option', { name: 'Clear selection' }));
      expect(screen.getByText('All 2 boards must be filled')).toBeInTheDocument();
      expect(submitButton()).toBeDisabled();
    });
  });

  describe('rating order', () => {
    it('reports a stronger player on a lower board', () => {
      renderEditor({ boardsPerMatch: 2 });
      pick(0, 'Bert Berg');
      pick(1, 'Anna Adler');
      expect(
        screen.getByText('Board 2 rating (2200) exceeds board 1 rating (2100) by more than 0 points')
      ).toBeInTheDocument();
      expect(submitButton()).toBeDisabled();
    });

    it('accepts equal ratings on adjacent boards with zero tolerance', () => {
      renderEditor({
        boardsPerMatch: 2,
        roster: [member('a', 'Alpha', 2000), member('b', 'Beta', 2000)],
      });
      pick(0, 'Alpha');
      pick(1, 'Beta');
      expect(screen.getByText('Lineup Valid')).toBeInTheDocument();
    });

    it('allows a difference equal to the tolerance', () => {
      renderEditor({
        boardsPerMatch: 2,
        ratingOrderTolerance: 100,
        existingLineup: lineupOf([e(1, 'p3', 2000), e(2, 'p2', 2100)]),
      });
      expect(screen.getByText('Lineup Valid')).toBeInTheDocument();
    });

    it('rejects a difference one point above the tolerance', () => {
      renderEditor({
        boardsPerMatch: 2,
        ratingOrderTolerance: 100,
        existingLineup: lineupOf([e(1, 'p3', 2000), e(2, 'p2', 2101)]),
      });
      expect(
        screen.getByText(
          'Board 2 rating (2101) exceeds board 1 rating (2000) by more than 100 points'
        )
      ).toBeInTheDocument();
    });

    it('does not check the order when enforcement is disabled', () => {
      renderEditor({
        boardsPerMatch: 2,
        enforceRatingOrder: false,
        existingLineup: lineupOf([e(1, 'p4', 1900), e(2, 'p1', 2200)]),
      });
      expect(screen.getByText('Lineup Valid')).toBeInTheDocument();
    });

    it('reports every violating board pair', () => {
      renderEditor({
        boardsPerMatch: 3,
        existingLineup: lineupOf([e(1, 'p4', 1900), e(2, 'p3', 2000), e(3, 'p1', 2200)]),
      });
      expect(
        screen.getByText('Board 2 rating (2000) exceeds board 1 rating (1900) by more than 0 points')
      ).toBeInTheDocument();
      expect(
        screen.getByText('Board 3 rating (2200) exceeds board 2 rating (2000) by more than 0 points')
      ).toBeInTheDocument();
    });

    it.skip('aqaa: open bug, see report — reports a rating-order violation between later boards while an earlier board is still empty', () => {
      // aqaa: suspected bug - the rating-order loop runs i < filledBoards.length but indexes
      // the sparse `lineup` array, so pairs behind an empty board are never compared.
      // Board 3 (1900) followed by board 4 (2200) must be flagged regardless of board 1.
      renderEditor({
        boardsPerMatch: 4,
        existingLineup: lineupOf([e(2, 'p2', 2100), e(3, 'p4', 1900), e(4, 'p1', 2200)]),
      });
      expect(
        screen.getByText('Board 4 rating (2200) exceeds board 3 rating (1900) by more than 0 points')
      ).toBeInTheDocument();
    });
  });

  describe('existing lineup', () => {
    it('loads entries onto their boards and ignores out-of-range boards', () => {
      renderEditor({
        boardsPerMatch: 2,
        existingLineup: lineupOf([
          e(0, 'p4', 1900),
          e(1, 'p1', 2200),
          e(2, 'p2', 2100),
          e(3, 'p3', 2000),
        ]),
      });
      expect(within(boards()[0]).getByTestId('select-value')).toHaveTextContent('Anna Adler (2200)');
      expect(within(boards()[1]).getByTestId('select-value')).toHaveTextContent('Bert Berg (2100)');
      expect(screen.getByText('Lineup Valid')).toBeInTheDocument();
    });

    it('flags a player placed on two boards', () => {
      renderEditor({
        boardsPerMatch: 2,
        existingLineup: lineupOf([e(1, 'p1', 2200), e(2, 'p1', 2200)]),
      });
      expect(screen.getByText('Each player can only play on one board')).toBeInTheDocument();
      expect(submitButton()).toBeDisabled();
    });

    it('offers to update an already submitted lineup', () => {
      renderEditor({
        boardsPerMatch: 2,
        existingLineup: lineupOf([e(1, 'p1', 2200), e(2, 'p2', 2100)], {
          submittedAt: '2024-05-01T10:00:00Z',
        }),
      });
      expect(screen.getByRole('button', { name: /update lineup/i })).toBeEnabled();
      expect(screen.getByText(/^Submitted at /)).toBeInTheDocument();
    });
  });

  describe('submission', () => {
    it('shows a pending state while submitting and re-enables afterwards', async () => {
      let resolve!: () => void;
      const onSubmit = vi.fn(
        () =>
          new Promise<void>((r) => {
            resolve = r;
          })
      );
      renderEditor({
        boardsPerMatch: 2,
        onSubmit,
        existingLineup: lineupOf([e(1, 'p1', 2200), e(2, 'p2', 2100)]),
      });

      fireEvent.click(submitButton());
      expect(await screen.findByRole('button', { name: /submitting/i })).toBeDisabled();

      await act(async () => {
        resolve();
      });
      expect(screen.getByRole('button', { name: /submit lineup/i })).toBeEnabled();
      expect(onSubmit).toHaveBeenCalledTimes(1);
    });
  });

  describe('read-only mode', () => {
    it('shows assignments as text without selectors or submit button', () => {
      renderEditor({
        readOnly: true,
        existingLineup: lineupOf([e(1, 'p1', 2200), e(2, 'ghost', 1500)]),
      });
      expect(screen.queryAllByTestId('board-select')).toHaveLength(0);
      expect(screen.queryByRole('button')).toBeNull();
      expect(screen.getByText('Anna Adler (2200)')).toBeInTheDocument();
      expect(screen.getByText('Unknown (1500)')).toBeInTheDocument();
      expect(screen.getAllByText('Not assigned')).toHaveLength(2);
    });
  });

  describe('deadline', () => {
    const NOW = new Date('2024-06-01T12:00:00Z');

    beforeEach(() => {
      vi.useFakeTimers();
      vi.setSystemTime(NOW);
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it.each([
      [2 * 3600_000 + 5 * 60_000, '2h 5m'],
      [3 * 60_000 + 20_000, '3m 20s'],
      [45_000, '45s'],
    ])('formats %i ms remaining as %s', (ms, text) => {
      renderEditor({ deadline: new Date(NOW.getTime() + ms) });
      expect(screen.getByText(text)).toBeInTheDocument();
    });

    it('counts down every second', () => {
      renderEditor({ deadline: new Date(NOW.getTime() + 10_000) });
      expect(screen.getByText('10s')).toBeInTheDocument();
      act(() => {
        vi.advanceTimersByTime(3000);
      });
      expect(screen.getByText('7s')).toBeInTheDocument();
    });

    it('marks the lineup overdue and blocks submission once the deadline passed', () => {
      renderEditor({
        boardsPerMatch: 2,
        deadline: new Date(NOW.getTime() - 1),
        existingLineup: lineupOf([e(1, 'p1', 2200), e(2, 'p2', 2100)]),
      });
      expect(screen.getByText('Overdue')).toBeInTheDocument();
      expect(screen.getByText('Lineup Valid')).toBeInTheDocument();
      expect(submitButton()).toBeDisabled();
    });

    it('treats the exact deadline moment as overdue', () => {
      renderEditor({ deadline: new Date(NOW.getTime()) });
      expect(screen.getByText('Overdue')).toBeInTheDocument();
    });

    it('switches to overdue when the countdown runs out', () => {
      renderEditor({
        boardsPerMatch: 2,
        deadline: new Date(NOW.getTime() + 2000),
        existingLineup: lineupOf([e(1, 'p1', 2200), e(2, 'p2', 2100)]),
      });
      expect(submitButton()).toBeEnabled();
      act(() => {
        vi.advanceTimersByTime(2000);
      });
      expect(screen.getByText('Overdue')).toBeInTheDocument();
      expect(submitButton()).toBeDisabled();
    });
  });
});
