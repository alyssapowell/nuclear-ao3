// A confirmation box for actions that are hard to undo. It takes focus when it
// opens (on Cancel, so Enter does not confirm by accident, or on the first
// field if it has one), keeps Tab inside, closes on Escape, and gives focus
// back to the button that opened it.
import { useEffect, useId, useRef, type ReactNode } from 'react';

export const BTN =
  'py-1 px-3 border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-orange-500 disabled:opacity-50';
export const DANGER = `${BTN} border-transparent text-white bg-red-700 hover:bg-red-800`;
export const GOOD = `${BTN} border-transparent text-white bg-green-700 hover:bg-green-800`;
export const NEUTRAL = `${BTN} border-gray-300 text-gray-700 bg-white hover:bg-gray-50`;
export const FIELD =
  'mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm text-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500';

const FOCUSABLE = 'button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), a[href]';

interface Props {
  title: string;
  confirmLabel: string;
  /** Red for actions that hurt (ban, hide); green for the rest. */
  danger?: boolean;
  /** The confirm button stays off until this is true (e.g. a reason is typed). */
  canConfirm?: boolean;
  busy?: boolean;
  /** Shown inside the box, e.g. an error from the server. */
  error?: string;
  onConfirm: () => void;
  onCancel: () => void;
  children?: ReactNode;
}

export default function ConfirmDialog({ title, confirmLabel, danger, canConfirm = true, busy, error, onConfirm, onCancel, children }: Props) {
  const box = useRef<HTMLDivElement>(null);
  const opener = useRef<Element | null>(document.activeElement);
  const titleId = useId();

  useEffect(() => {
    const el = box.current;
    if (!el) return;
    const first = el.querySelector<HTMLElement>('input, select, textarea') ?? el.querySelector<HTMLElement>('[data-cancel]');
    first?.focus();
    const back = opener.current;
    return () => {
      if (back instanceof HTMLElement && back.isConnected) back.focus();
    };
  }, []);

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      e.stopPropagation();
      onCancel();
      return;
    }
    if (e.key !== 'Tab' || !box.current) return;
    const items = Array.from(box.current.querySelectorAll<HTMLElement>(FOCUSABLE));
    if (items.length === 0) return;
    const firstItem = items[0];
    const last = items[items.length - 1];
    if (e.shiftKey && document.activeElement === firstItem) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      firstItem.focus();
    }
  };

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 p-4" onKeyDown={onKeyDown}>
      <div ref={box} role="alertdialog" aria-modal="true" aria-labelledby={titleId}
        className="w-full max-w-md bg-white rounded-lg shadow-xl p-6">
        <h2 id={titleId} className="text-lg font-bold text-gray-900 mb-3">{title}</h2>
        <div className="text-sm text-gray-700 space-y-3">{children}</div>
        {error && <p role="alert" className="mt-3 text-sm text-red-600">{error}</p>}
        <div className="mt-5 flex justify-end gap-2">
          <button type="button" data-cancel className={NEUTRAL} onClick={onCancel} disabled={busy}>Cancel</button>
          <button type="button" className={danger ? DANGER : GOOD} onClick={onConfirm} disabled={busy || !canConfirm}>
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
