import './globals.css';
import type { Metadata } from 'next';
import Link from 'next/link';

export const metadata: Metadata = {
  title: 'Flowrunner GUI',
  description: 'Cyberpunk control surface for Flowrunner',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <div className="min-h-screen grid grid-rows-[auto,1fr]">
          <header className="border-b border-white/5 bg-bg-subtle/60 backdrop-blur">
            <div className="mx-auto max-w-7xl px-6 py-4 flex items-center justify-between">
              <Link href="/" className="flex items-center gap-3">
                <div className="h-8 w-8 rounded bg-gradient-to-tr from-neon-cyan to-neon-pink shadow-glow" />
                <span className="font-semibold tracking-wide neon-text">Flowrunner</span>
              </Link>
              <nav className="flex items-center gap-6 text-sm text-white/80">
                <Link href="/flows">Flows</Link>
                <Link href="/executions">Executions</Link>
              </nav>
            </div>
          </header>
          <main className="mx-auto max-w-7xl w-full px-6 py-8">
            {children}
          </main>
        </div>
      </body>
    </html>
  );
}
