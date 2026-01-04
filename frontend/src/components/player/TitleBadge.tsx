'use client';

interface TitleBadgeProps {
  title?: string;
  size?: 'sm' | 'md' | 'lg';
}

const titleConfig: Record<
  string,
  { label: string; bgColor: string; textColor: string; description: string }
> = {
  GM: {
    label: 'GM',
    bgColor: 'bg-yellow-500',
    textColor: 'text-black',
    description: 'Großmeister',
  },
  IM: {
    label: 'IM',
    bgColor: 'bg-gray-400',
    textColor: 'text-black',
    description: 'Internationaler Meister',
  },
  FM: {
    label: 'FM',
    bgColor: 'bg-amber-600',
    textColor: 'text-white',
    description: 'FIDE-Meister',
  },
  CM: {
    label: 'CM',
    bgColor: 'bg-amber-800',
    textColor: 'text-white',
    description: 'Kandidatenmeister',
  },
  WGM: {
    label: 'WGM',
    bgColor: 'bg-yellow-500',
    textColor: 'text-black',
    description: 'Großmeisterin',
  },
  WIM: {
    label: 'WIM',
    bgColor: 'bg-gray-400',
    textColor: 'text-black',
    description: 'Internationale Meisterin',
  },
  WFM: {
    label: 'WFM',
    bgColor: 'bg-amber-600',
    textColor: 'text-white',
    description: 'FIDE-Meisterin',
  },
  WCM: {
    label: 'WCM',
    bgColor: 'bg-amber-800',
    textColor: 'text-white',
    description: 'Kandidatenmeisterin',
  },
};

const sizeClasses = {
  sm: 'text-xs px-1 py-0.5',
  md: 'text-sm px-1.5 py-0.5',
  lg: 'text-base px-2 py-1',
};

export function TitleBadge({ title, size = 'md' }: TitleBadgeProps) {
  if (!title) return null;

  const config = titleConfig[title];
  if (!config) return null;

  return (
    <span
      className={`inline-flex items-center font-bold rounded ${config.bgColor} ${config.textColor} ${sizeClasses[size]}`}
      title={config.description}
    >
      {config.label}
    </span>
  );
}

interface TitleLegendProps {
  showWomen?: boolean;
}

export function TitleLegend({ showWomen = true }: TitleLegendProps) {
  const titles = showWomen
    ? Object.entries(titleConfig)
    : Object.entries(titleConfig).filter(([key]) => !key.startsWith('W'));

  return (
    <div className="flex flex-wrap gap-2">
      {titles.map(([key, config]) => (
        <div key={key} className="flex items-center gap-1.5 text-sm">
          <span
            className={`inline-flex items-center font-bold rounded px-1.5 py-0.5 text-xs ${config.bgColor} ${config.textColor}`}
          >
            {config.label}
          </span>
          <span className="text-gray-600 dark:text-gray-400">
            {config.description}
          </span>
        </div>
      ))}
    </div>
  );
}
