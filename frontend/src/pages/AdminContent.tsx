// Admin "Content": hide or unhide a work, chapter, series, bookmark or
// comment, and mark works and comments as spam. Paste the address of the
// page or the id. The screen can't see what state the content is in, so both
// "Hide" and "Unhide" are offered; the server says if nothing changed.
import { useState } from 'react';
import AdminPage from '@/components/admin/AdminPage';
import ContentActions from '@/components/admin/ContentActions';
import { BTN, FIELD } from '@/components/admin/ConfirmDialog';
import { CONTENT_KINDS, parseContentRef, type ContentKind } from '@/lib/admin';

export default function AdminContentPage() {
  const [input, setInput] = useState('');
  const [kind, setKind] = useState<ContentKind>('work');
  const [target, setTarget] = useState<{ kind: ContentKind; id: string } | null>(null);
  const [error, setError] = useState('');

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    setTarget(null);
    const ref = parseContentRef(input);
    if (!ref) {
      setError('That is not an id or the address of a work, chapter, series, bookmark or comment.');
      return;
    }
    setError('');
    if (ref.kind) setKind(ref.kind);
    setTarget({ kind: ref.kind ?? kind, id: ref.id });
  };

  return (
    <AdminPage title="Content" need="moderator" audience="the archive's admins and moderators"
      intro="Hide or unhide content, or mark it as spam. Paste the address of the page, or its id and pick the type. Every action asks for a reason where one is needed and is written to the audit log.">
      <form onSubmit={submit} className="space-y-3">
        <div>
          <label htmlFor="content-ref" className="block text-sm font-medium text-gray-700">Address or id</label>
          <input id="content-ref" type="text" value={input} onChange={(e) => setInput(e.target.value)} className={FIELD} />
        </div>
        <div>
          <label htmlFor="content-kind" className="block text-sm font-medium text-gray-700">Type (used when you paste an id; an address decides it itself)</label>
          <select id="content-kind" value={kind} onChange={(e) => setKind(e.target.value as ContentKind)}
            className="mt-1 block px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500">
            {CONTENT_KINDS.map((k) => <option key={k.value} value={k.value}>{k.label}</option>)}
          </select>
        </div>
        <button type="submit" disabled={!input.trim()} className={`${BTN} border-transparent text-white bg-orange-700 hover:bg-orange-800`}>Continue</button>
      </form>
      {error && <p role="alert" className="mt-4 text-sm text-red-600">{error}</p>}
      {target && (
        <section aria-labelledby="content-target" className="mt-6 border-t border-gray-200 pt-4">
          <h2 id="content-target" className="text-lg font-medium text-gray-900">
            {CONTENT_KINDS.find((k) => k.value === target.kind)?.label} <span className="text-sm font-normal break-all">{target.id}</span>
          </h2>
          <div className="mt-3">
            <ContentActions key={`${target.kind}-${target.id}`} kind={target.kind} id={target.id} name={`${target.kind} ${target.id}`} />
          </div>
        </section>
      )}
    </AdminPage>
  );
}
