// Where a person manages their own pseuds: the pen names an account posts,
// comments and bookmarks under. Only the signed-in person ever sees this list;
// nothing public shows which pseuds belong together.
import { useCallback, useEffect, useId, useState } from 'react';
import { Link } from 'react-router-dom';
import { listMyPseuds, createPseud, updatePseud, deletePseud, type MyPseud } from '@/lib/api';
import { useAuth } from '@/utils/auth';
import { getPseudUrl } from '@/lib/urls';

const NAME_MAX = 40;
const DESC_MAX = 500;

interface PseudFormProps {
  initial?: { name: string; description: string };
  submitLabel: string;
  busy: boolean;
  onSubmit: (v: { name: string; description: string }) => void;
  onCancel?: () => void;
  idPrefix: string;
}

function PseudForm({ initial, submitLabel, busy, onSubmit, onCancel, idPrefix }: PseudFormProps) {
  const [name, setName] = useState(initial?.name ?? '');
  const [description, setDescription] = useState(initial?.description ?? '');
  return (
    <form
      className="space-y-3"
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit({ name: name.trim(), description });
      }}
    >
      <div>
        <label htmlFor={`${idPrefix}-name`} className="block text-sm font-medium text-slate-700 mb-1">
          Name
        </label>
        <input
          id={`${idPrefix}-name`}
          type="text"
          required
          maxLength={NAME_MAX}
          value={name}
          onChange={(e) => setName(e.target.value)}
          aria-describedby={`${idPrefix}-name-help`}
          className="w-full px-3 py-2 border border-slate-300 rounded-md"
        />
        <p id={`${idPrefix}-name-help`} className="text-xs text-slate-600 mt-1">
          Up to {NAME_MAX} characters: letters, numbers, spaces, underscores and dashes.
        </p>
      </div>
      <div>
        <label htmlFor={`${idPrefix}-desc`} className="block text-sm font-medium text-slate-700 mb-1">
          Description
        </label>
        <textarea
          id={`${idPrefix}-desc`}
          rows={3}
          maxLength={DESC_MAX}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          className="w-full px-3 py-2 border border-slate-300 rounded-md"
        />
        <p className="text-xs text-slate-600 mt-1">Shown on this pseud's public page. Up to {DESC_MAX} characters.</p>
      </div>
      <div className="flex gap-2">
        <button
          type="submit"
          disabled={busy || !name.trim()}
          className="px-4 py-2 bg-orange-600 text-white rounded-md disabled:opacity-50"
        >
          {submitLabel}
        </button>
        {onCancel && (
          <button type="button" onClick={onCancel} className="px-4 py-2 border border-slate-300 rounded-md">
            Cancel
          </button>
        )}
      </div>
    </form>
  );
}

export default function MyPseuds() {
  const { token } = useAuth();
  const [pseuds, setPseuds] = useState<MyPseud[] | null>(null);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [busy, setBusy] = useState(false);
  const [editing, setEditing] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [createKey, setCreateKey] = useState(0); // remounts the create form to clear it
  const createId = useId();

  const load = useCallback(async () => {
    try {
      setPseuds(await listMyPseuds(token || undefined));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load your pseuds');
      setPseuds([]);
    }
  }, [token]);

  useEffect(() => {
    load();
  }, [load]);

  // Runs a change, reports its result in the live regions, then reloads the list.
  const run = async (what: () => Promise<string>, done?: () => void) => {
    setBusy(true);
    setError('');
    setNotice('');
    try {
      setNotice(await what());
      done?.();
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Something went wrong');
    } finally {
      setBusy(false);
    }
  };

  const defaultPseud = pseuds?.find((p) => p.is_default);

  return (
    <div className="max-w-3xl mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold text-slate-900 mb-2">My pseuds</h1>
      <p className="text-slate-700 mb-6">
        A pseud is a name you post, comment and bookmark under. Only you can see which pseuds are yours.
        Your default pseud is used whenever you don't pick another.
      </p>

      <div aria-live="polite" className="mb-4">
        {notice && <p role="status" className="text-green-800 bg-green-50 border border-green-200 rounded p-3">{notice}</p>}
      </div>
      {error && <p role="alert" className="text-red-800 bg-red-50 border border-red-200 rounded p-3 mb-4">{error}</p>}

      {pseuds === null ? (
        <p role="status" className="text-slate-700">Loading…</p>
      ) : (
        <ul className="space-y-4 mb-10" aria-label="Your pseuds">
          {pseuds.map((p) => (
            <li key={p.id} className="bg-white border border-slate-200 rounded-lg p-4">
              {editing === p.id ? (
                <PseudForm
                  idPrefix={`edit-${p.id}`}
                  initial={{ name: p.name, description: p.description }}
                  submitLabel="Save changes"
                  busy={busy}
                  onCancel={() => setEditing(null)}
                  onSubmit={(v) =>
                    run(async () => {
                      await updatePseud(p.id, v, token || undefined);
                      return `Saved ${v.name}.`;
                    }, () => setEditing(null))
                  }
                />
              ) : (
                <>
                  <div className="flex items-center gap-2">
                    <h2 className="text-lg font-semibold text-slate-900">{p.name}</h2>
                    {p.is_default && (
                      <span className="text-xs px-2 py-0.5 rounded bg-orange-100 text-orange-800">Default</span>
                    )}
                  </div>
                  {p.description && <p className="text-sm text-slate-700 mt-1">{p.description}</p>}
                  <div className="flex flex-wrap gap-3 mt-3 text-sm">
                    <Link to={getPseudUrl(p.id)} className="text-blue-700 hover:underline" aria-label={`View the page of ${p.name}`}>
                      View page
                    </Link>
                    <button
                      type="button"
                      onClick={() => setEditing(p.id)}
                      className="text-blue-700 hover:underline"
                      aria-label={`Edit ${p.name}`}
                    >
                      Edit
                    </button>
                    {!p.is_default && (
                      <>
                        <button
                          type="button"
                          disabled={busy}
                          onClick={() =>
                            run(async () => {
                              await updatePseud(p.id, { is_default: true }, token || undefined);
                              return `${p.name} is now your default pseud.`;
                            })
                          }
                          className="text-blue-700 hover:underline"
                          aria-label={`Make ${p.name} my default`}
                        >
                          Make default
                        </button>
                        <button
                          type="button"
                          onClick={() => setDeleting(p.id)}
                          className="text-red-700 hover:underline"
                          aria-label={`Delete ${p.name}`}
                        >
                          Delete
                        </button>
                      </>
                    )}
                  </div>
                  {deleting === p.id && (
                    <div role="alertdialog" aria-labelledby={`del-${p.id}-title`} aria-describedby={`del-${p.id}-desc`} className="mt-3 p-3 bg-red-50 border border-red-200 rounded">
                      <p id={`del-${p.id}-title`} className="font-medium text-red-900">Delete {p.name}?</p>
                      <p id={`del-${p.id}-desc`} className="text-sm text-red-900 mt-1">
                        Its works, series, comments and gifts move to your default pseud
                        {defaultPseud ? ` (${defaultPseud.name})` : ''}. Anyone following {p.name} stops following it.
                        This cannot be undone.
                      </p>
                      <div className="flex gap-2 mt-3">
                        <button
                          type="button"
                          autoFocus
                          disabled={busy}
                          onClick={() =>
                            run(async () => {
                              await deletePseud(p.id, token || undefined);
                              return `Deleted ${p.name}. Its works and comments are now under your default pseud.`;
                            }, () => setDeleting(null))
                          }
                          className="px-3 py-1 bg-red-700 text-white rounded"
                        >
                          Yes, delete {p.name}
                        </button>
                        <button type="button" onClick={() => setDeleting(null)} className="px-3 py-1 border border-slate-300 rounded">
                          Keep it
                        </button>
                      </div>
                    </div>
                  )}
                </>
              )}
            </li>
          ))}
        </ul>
      )}

      <section aria-labelledby={`${createId}-heading`} className="bg-white border border-slate-200 rounded-lg p-4">
        <h2 id={`${createId}-heading`} className="text-lg font-semibold text-slate-900 mb-3">Add a pseud</h2>
        <PseudForm
          key={createKey}
          idPrefix="new-pseud"
          submitLabel="Create pseud"
          busy={busy}
          onSubmit={(v) =>
            run(async () => {
              await createPseud(v, token || undefined);
              return `Created ${v.name}.`;
            }, () => setCreateKey((k) => k + 1))
          }
        />
      </section>
    </div>
  );
}
