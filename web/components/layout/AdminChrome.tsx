'use client';

// The frame around every staff screen: sidebar (a drawer below md), the
// context bar, and the workspace. Used by the (admin) layout for the screens
// that have not moved to the list → view → edit shell yet, and by the shell
// itself (shell/Shell.tsx), so both look and behave the same.

import { useState, type ReactNode } from 'react';
import { Menu, X } from 'lucide-react';
import Sidebar from '@/components/layout/Sidebar';
import { ContextBar } from '@/shell/ContextBar';
import type { Ctx } from '@/framework/types';

export function AdminChrome({ ctx, onSwitch, children, flush }: {
  ctx: Ctx;
  onSwitch: (group: string, school: string) => void;
  children: ReactNode;
  /** The workspace manages its own padding and scrolling (the shell). */
  flush?: boolean;
}) {
  const [mobileOpen, setMobileOpen] = useState(false);
  // With no school chosen there is nothing for the school menu to open.
  const hasSidebar = ctx.scope === 'school';

  return (
    <div className="flex min-h-screen">
      <a href="#main-content" className="skip-link">
        Skip to content
      </a>

      {hasSidebar && <Sidebar mobileOpen={mobileOpen} onClose={() => setMobileOpen(false)} />}

      {hasSidebar && mobileOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-30 md:hidden"
          aria-hidden="true"
          onClick={() => setMobileOpen(false)}
        />
      )}

      <div className={`flex-1 min-w-0 flex flex-col ${hasSidebar ? 'md:ml-64' : ''} ${flush ? 'h-screen' : ''}`}>
        <div className="sticky top-0 z-20">
          <ContextBar
            ctx={ctx}
            onSwitch={onSwitch}
            leading={
              hasSidebar ? (
                <button
                  type="button"
                  className="rounded-lg border border-gray-300 p-1.5 md:hidden"
                  onClick={() => setMobileOpen((open) => !open)}
                  aria-expanded={mobileOpen}
                  aria-controls="sidebar"
                  aria-label={mobileOpen ? 'Close menu' : 'Open menu'}
                >
                  {mobileOpen ? <X size={18} aria-hidden="true" /> : <Menu size={18} aria-hidden="true" />}
                </button>
              ) : (
                <span className="text-lg font-bold">Shule360</span>
              )
            }
          />
        </div>

        <main
          id="main-content"
          tabIndex={-1}
          className={`flex-1 focus:outline-none ${flush ? 'min-h-0 flex flex-col' : 'p-4 sm:p-6 lg:p-8'}`}
        >
          {children}
        </main>
      </div>
    </div>
  );
}
