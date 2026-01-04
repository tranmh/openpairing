import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import { Providers } from '@/components/providers';
import './globals.css';

const inter = Inter({
  subsets: ['latin'],
  variable: '--font-inter',
});

export const metadata: Metadata = {
  title: {
    default: 'OpenPairing - Schachturnier-Management',
    template: '%s | OpenPairing',
  },
  description:
    'Moderne Schachturnier-Verwaltung für den DACH-Raum. Swiss-System Paarungen mit JaVaFo, FIDE/DWZ/ÖSB Integration.',
  keywords: [
    'Schach',
    'Turnier',
    'Swiss-System',
    'FIDE',
    'DWZ',
    'DSB',
    'Mannschaftsturnier',
    'JaVaFo',
    'Paarung',
  ],
  authors: [{ name: 'OpenPairing Team' }],
  openGraph: {
    title: 'OpenPairing',
    description: 'Moderne Schachturnier-Verwaltung für den DACH-Raum',
    type: 'website',
    locale: 'de_DE',
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="de" suppressHydrationWarning>
      <body className={`${inter.variable} font-sans antialiased`}>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
