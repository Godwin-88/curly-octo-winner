'use client';

// Standard confirmation before a destructive action.
//
// Deleting curriculum, attendance or assessment data cannot be undone by the
// user, so every such action routes through this dialog: it names the item, says
// what will happen, and keeps the confirm button disabled while the request is
// in flight so a double click cannot fire two deletes.

import type { ReactNode } from 'react';

export function ConfirmDialog({
  title,
  message,
  confirmLabel = 'Delete',
  cancelLabel = 'Cancel',
  busy = false,
  onConfirm,
  onCancel,
  children,
}: {
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  busy?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  /** Optional extra detail, e.g. the server's explanation of what blocks a delete. */
  children?: ReactNode;
}) {
  return (
    <div
      className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="confirm-dialog-title"
      onClick={(e) => {
        if (e.target === e.currentTarget && !busy) onCancel();
      }}
    >
      <div className="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
        <h2 id="confirm-dialog-title" className="text-lg font-semibold mb-2">
          {title}
        </h2>
        <p className="text-sm text-gray-600 mb-4">{message}</p>
        {children}
        <div className="flex justify-end gap-3 pt-2">
          <button type="button" className="btn-secondary" onClick={onCancel} disabled={busy}>
            {cancelLabel}
          </button>
          <button
            type="button"
            className="btn-danger"
            onClick={onConfirm}
            disabled={busy}
          >
            {busy ? 'Working…' : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
