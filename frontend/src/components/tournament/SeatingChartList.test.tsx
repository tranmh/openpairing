import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { SeatingChartList } from './SeatingChartList';
import type { SeatingChartData } from '@/types/roundrobin';

const mockChartData: SeatingChartData = {
  tournamentId: '123e4567-e89b-12d3-a456-426614174000',
  tournamentName: 'Test Tournament',
  numPlayers: 6,
  numSeats: 6,
  fixedSeat: 1,
  byeSeat: 0,
  roundNumber: 1,
  seats: [
    { seatNumber: 1, startNumber: 1, playerName: 'Player A', rating: 2000, isFixed: true, isBye: false },
    { seatNumber: 2, startNumber: 2, playerName: 'Player B', rating: 1900, isFixed: false, isBye: false },
    { seatNumber: 3, startNumber: 3, playerName: 'Player C', rating: 1800, isFixed: false, isBye: false },
    { seatNumber: 4, startNumber: 4, playerName: 'Player D', rating: 1700, isFixed: false, isBye: false },
    { seatNumber: 5, startNumber: 5, playerName: 'Player E', rating: 1600, isFixed: false, isBye: false },
    { seatNumber: 6, startNumber: 6, playerName: 'Player F', rating: 1500, isFixed: false, isBye: false },
  ],
  pairings: [
    { board: 1, whiteSeat: 1, blackSeat: 6, whiteStartNo: 1, blackStartNo: 6, whiteName: 'Player A', blackName: 'Player F', isBye: false },
    { board: 2, whiteSeat: 2, blackSeat: 5, whiteStartNo: 2, blackStartNo: 5, whiteName: 'Player B', blackName: 'Player E', isBye: false },
    { board: 3, whiteSeat: 3, blackSeat: 4, whiteStartNo: 3, blackStartNo: 4, whiteName: 'Player C', blackName: 'Player D', isBye: false },
  ],
};

const mockChartDataWithBye: SeatingChartData = {
  tournamentId: '123e4567-e89b-12d3-a456-426614174001',
  tournamentName: 'Test Tournament Odd',
  numPlayers: 5,
  numSeats: 6,
  fixedSeat: 1,
  byeSeat: 4,
  roundNumber: 1,
  seats: [
    { seatNumber: 1, startNumber: 1, playerName: 'Player A', rating: 2000, isFixed: true, isBye: false },
    { seatNumber: 2, startNumber: 2, playerName: 'Player B', rating: 1900, isFixed: false, isBye: false },
    { seatNumber: 3, startNumber: 3, playerName: 'Player C', rating: 1800, isFixed: false, isBye: false },
    { seatNumber: 4, startNumber: 6, playerName: '', rating: 0, isFixed: false, isBye: true },
    { seatNumber: 5, startNumber: 5, playerName: 'Player E', rating: 1600, isFixed: false, isBye: false },
    { seatNumber: 6, startNumber: 4, playerName: 'Player D', rating: 1700, isFixed: false, isBye: false },
  ],
  pairings: [
    { board: 1, whiteSeat: 1, blackSeat: 6, whiteStartNo: 1, blackStartNo: 4, whiteName: 'Player A', blackName: 'Player D', isBye: false },
    { board: 2, whiteSeat: 2, blackSeat: 5, whiteStartNo: 2, blackStartNo: 5, whiteName: 'Player B', blackName: 'Player E', isBye: false },
    { board: 3, whiteSeat: 3, blackSeat: 4, whiteStartNo: 3, blackStartNo: 6, whiteName: 'Player C', blackName: '', isBye: true },
  ],
};

describe('SeatingChartList', () => {
  it('renders all seats', () => {
    render(<SeatingChartList data={mockChartData} />);

    // Check that all players are rendered
    expect(screen.getByText('Player A')).toBeInTheDocument();
    expect(screen.getByText('Player B')).toBeInTheDocument();
    expect(screen.getByText('Player C')).toBeInTheDocument();
    expect(screen.getByText('Player D')).toBeInTheDocument();
    expect(screen.getByText('Player E')).toBeInTheDocument();
    expect(screen.getByText('Player F')).toBeInTheDocument();
  });

  it('renders seat numbers', () => {
    render(<SeatingChartList data={mockChartData} />);

    // Check that seat numbers are shown
    expect(screen.getAllByText('1').length).toBeGreaterThan(0);
    expect(screen.getAllByText('2').length).toBeGreaterThan(0);
    expect(screen.getAllByText('6').length).toBeGreaterThan(0);
  });

  it('renders ratings', () => {
    render(<SeatingChartList data={mockChartData} />);

    expect(screen.getByText('2000')).toBeInTheDocument();
    expect(screen.getByText('1900')).toBeInTheDocument();
    expect(screen.getByText('1500')).toBeInTheDocument();
  });

  it('shows fixed player indicator', () => {
    render(<SeatingChartList data={mockChartData} />);

    // Fixed player should have a star indicator
    const starIndicators = screen.getAllByText((content, element) => {
      return content.includes('fixiert') || content.includes('Fixiert') || element?.textContent?.includes('★') || false;
    });
    expect(starIndicators.length).toBeGreaterThan(0);
  });

  it('handles bye seats', () => {
    render(<SeatingChartList data={mockChartDataWithBye} />);

    // Should show "Spielfrei" for the bye seat
    const byeIndicators = screen.getAllByText(/spielfrei/i);
    expect(byeIndicators.length).toBeGreaterThan(0);
  });

  it('renders correct number of rows', () => {
    render(<SeatingChartList data={mockChartData} />);

    // Should have header row plus 6 data rows
    const table = document.querySelector('table');
    expect(table).toBeInTheDocument();

    const rows = table?.querySelectorAll('tbody tr');
    expect(rows?.length).toBe(6);
  });

  it('applies custom className', () => {
    const { container } = render(<SeatingChartList data={mockChartData} className="custom-class" />);

    expect(container.firstChild).toHaveClass('custom-class');
  });
});

describe('SeatingChartList sorting', () => {
  it('can sort by seat number', () => {
    render(<SeatingChartList data={mockChartData} />);

    // Check that seats appear in order initially
    const rows = document.querySelectorAll('tbody tr');
    const firstRow = rows[0];
    expect(firstRow.textContent).toContain('1');
  });

  it('displays start numbers', () => {
    render(<SeatingChartList data={mockChartData} />);

    // All start numbers 1-6 should be visible
    for (let i = 1; i <= 6; i++) {
      const startNoElements = screen.getAllByText(String(i));
      expect(startNoElements.length).toBeGreaterThan(0);
    }
  });
});
