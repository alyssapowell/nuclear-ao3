import { useEffect, useId, useState } from 'react';
import { listMyPseuds, type MyPseud } from '@/lib/api';

// "Post as": lets a signed-in person pick which of their own pseuds a work,
// comment or bookmark goes out under. Only the person's own list is ever
// loaded, and nothing is shown unless they have more than one pseud, so a
// reader of the page learns nothing. An empty value means "my default pseud",
// which is what the server uses when no pseud is sent.

interface PseudPickerProps {
  token?: string | null;
  /** The chosen pseud id, or '' for the default. */
  value: string;
  onChange: (pseudId: string) => void;
  label?: string;
  className?: string;
}

export default function PseudPicker({ token, value, onChange, label = 'Post as', className }: PseudPickerProps) {
  const id = useId();
  const [pseuds, setPseuds] = useState<MyPseud[]>([]);

  useEffect(() => {
    if (!token) {
      setPseuds([]);
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        const list = await listMyPseuds(token);
        if (!cancelled) setPseuds(list);
      } catch {
        // No picker is better than a broken form: the server falls back to the default pseud.
        if (!cancelled) setPseuds([]);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [token]);

  if (!token || pseuds.length < 2) return null;

  return (
    <div className={className}>
      <label htmlFor={id} className="block text-sm font-medium text-slate-700 mb-1">
        {label}
      </label>
      <select
        id={id}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500"
      >
        {pseuds.map((p) => (
          <option key={p.id} value={p.is_default ? '' : p.id}>
            {p.name}
            {p.is_default ? ' (default)' : ''}
          </option>
        ))}
      </select>
    </div>
  );
}
