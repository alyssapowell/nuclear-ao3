// Hide, unhide and spam buttons for one piece of content. Hiding and marking
// spam ask for a reason first (the server requires one and records it);
// undoing them does not need asking. Used by the Content screen and by each
// report in the Reports queue.
import { useState } from 'react';
import ConfirmDialog, { FIELD, NEUTRAL, DANGER } from '@/components/admin/ConfirmDialog';
import { actionsFor, contentAction, type ContentAction, type ContentKind } from '@/lib/admin';

const LABEL: Record<ContentAction, string> = { hide: 'Hide', unhide: 'Unhide', spam: 'Mark as spam', unspam: 'Not spam' };
const DONE: Record<ContentAction, string> = { hide: 'hidden', unhide: 'unhidden', spam: 'marked as spam', unspam: 'unmarked as spam' };
const ASKS: ContentAction[] = ['hide', 'spam'];

interface Props {
  kind: ContentKind;
  id: string;
  /** Names the content in button labels, e.g. "work abc". */
  name: string;
  onDone?: (message: string) => void;
}

export default function ContentActions({ kind, id, name, onDone }: Props) {
  const [asking, setAsking] = useState<ContentAction | null>(null);
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  const run = async (action: ContentAction, why?: string) => {
    setBusy(true);
    setError('');
    setNotice('');
    const r = await contentAction(kind, id, action, why);
    setBusy(false);
    if (!r.ok) {
      setError(r.error || 'Could not do that.');
      return false;
    }
    const msg = `Done: ${name} ${DONE[action]}.`;
    setNotice(msg);
    onDone?.(msg);
    return true;
  };

  const confirm = async () => {
    if (!asking) return;
    if (await run(asking, reason.trim())) setAsking(null);
  };

  return (
    <div>
      <div className="flex flex-wrap gap-2">
        {actionsFor(kind).map((a) => (
          <button key={a} type="button" className={ASKS.includes(a) ? DANGER : NEUTRAL} disabled={busy}
            aria-label={`${LABEL[a]}: ${name}`}
            onClick={() => {
              setError('');
              if (ASKS.includes(a)) {
                setReason('');
                setAsking(a);
              } else {
                run(a);
              }
            }}>
            {LABEL[a]}{ASKS.includes(a) ? '…' : ''}
          </button>
        ))}
      </div>
      {!asking && error && <p role="alert" className="mt-2 text-sm text-red-600">{error}</p>}
      {!asking && notice && <p role="status" className="mt-2 text-sm text-green-700">{notice}</p>}
      {asking && (
        <ConfirmDialog title={asking === 'hide' ? `Hide this ${kind}?` : `Mark this ${kind} as spam?`}
          confirmLabel={LABEL[asking]} danger canConfirm={reason.trim() !== ''} busy={busy} error={error}
          onConfirm={confirm} onCancel={() => setAsking(null)}>
          <p>
            {asking === 'hide'
              ? 'It disappears for everyone except admins, moderators and its creators.'
              : 'It is hidden for readers until the mark is cleared.'}
          </p>
          <div>
            <label htmlFor={`reason-${kind}-${id}`} className="block font-medium text-gray-700">Reason (required)</label>
            <textarea id={`reason-${kind}-${id}`} rows={3} value={reason} onChange={(e) => setReason(e.target.value)} className={FIELD} />
          </div>
        </ConfirmDialog>
      )}
    </div>
  );
}
