'use client';

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { JokerValue } from '@/types/wam';

interface JokerSelectProps {
  value: JokerValue;
  onChange: (_value: JokerValue) => void;
  disabled?: boolean;
  className?: string;
}

const JOKER_OPTIONS: { value: JokerValue; label: string }[] = [
  { value: 0, label: '0' },
  { value: 50, label: '+50' },
  { value: 100, label: '+100' },
  { value: 150, label: '+150' },
];

/**
 * Dropdown for selecting joker points (0, 50, 100, or 150 DWZ points)
 * Used in WAM tournaments to handicap stronger players
 */
export function JokerSelect({ value, onChange, disabled, className }: JokerSelectProps) {
  return (
    <Select
      value={String(value)}
      onValueChange={(v) => onChange(Number(v) as JokerValue)}
      disabled={disabled}
    >
      <SelectTrigger className={className ?? 'w-[80px]'}>
        <SelectValue placeholder="Joker" />
      </SelectTrigger>
      <SelectContent>
        {JOKER_OPTIONS.map((option) => (
          <SelectItem key={option.value} value={String(option.value)}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
