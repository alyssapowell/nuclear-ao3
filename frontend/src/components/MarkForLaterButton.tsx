import { useEffect, useState } from 'react';
import Button from '@/components/ui/Button';
import { getHistoryEntry, setMarkedForLater } from '@/lib/api';

// "Mark for Later" on the work page, and "Mark as Read" once marked. As in
// AO3 it is only offered to a logged-in person who has reading history on.
export function MarkForLaterButton({ workId, authToken }: { workId: string; authToken?: string }) {
  const [state, setState] = useState<{ enabled: boolean; marked: boolean } | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!authToken) {
      setState(null);
      return;
    }
    let live = true;
    getHistoryEntry(workId, authToken)
      .then((r) => live && setState({ enabled: r.history_enabled, marked: r.marked_for_later }))
      .catch(() => live && setState(null));
    return () => { live = false; };
  }, [workId, authToken]);

  if (!authToken || !state || !state.enabled) return null;

  const toggle = async () => {
    setBusy(true);
    setError('');
    try {
      const r = await setMarkedForLater(workId, !state.marked, authToken);
      setState({ enabled: true, marked: r.marked_for_later });
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not update Marked for Later');
    } finally {
      setBusy(false);
    }
  };

  return (
    <span>
      <Button type="button" variant="outline" size="sm" onClick={toggle} disabled={busy}>
        {state.marked ? 'Mark as Read' : 'Mark for Later'}
      </Button>
      {error && <span role="alert" className="text-sm text-red-700 ml-2">{error}</span>}
    </span>
  );
}

export default MarkForLaterButton;
