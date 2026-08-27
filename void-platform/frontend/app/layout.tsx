import type { Metadata } from 'next';
import React from 'react';
import { I18nProvider } from '../lib/i18n/I18nProvider';
import { ThemeProvider } from '../components/theme/ThemeProvider';
import './globals.css';

export const metadata: Metadata = {
  title: 'VOID — Synthetic Reality Generator',
  description: 'Synthetic Data, Environment & Behavior Simulation Platform',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" dir="ltr" suppressHydrationWarning>
      <body>
        <ThemeProvider>
          <I18nProvider>{children}</I18nProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
