// The form behind the Contact and Feedback pages. It follows AO3's two forms:
// Support & Feedback, and Policy & Abuse (which also asks which page it is
// about). A guest gives an email address for the reply; a logged-in person
// does not, and the account's own address is used.
import { useState, type FormEvent } from 'react';
import { isAuthenticated } from '@/utils/auth';
import { LIMITS, sendTicket, type TicketKind } from '@/lib/support';

const FIELD =
  'mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500';

export default function SupportForm({ defaultKind = 'support', allowAbuse = true }: { defaultKind?: TicketKind; allowAbuse?: boolean }) {
  const loggedIn = isAuthenticated();
  const [kind, setKind] = useState<TicketKind>(defaultKind);
  const [subject, setSubject] = useState('');
  const [body, setBody] = useState('');
  const [email, setEmail] = useState('');
  const [url, setUrl] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [sent, setSent] = useState(false);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError('');
    const result = await sendTicket({
      kind,
      subject: subject.trim(),
      body: body.trim(),
      ...(loggedIn ? {} : { email: email.trim() }),
      ...(kind === 'abuse' && url.trim() ? { url: url.trim() } : {}),
    });
    setBusy(false);
    if (result.ok) {
      setSent(true);
      setSubject('');
      setBody('');
      setUrl('');
    } else {
      setError(result.error || 'Something went wrong. Please try again.');
    }
  };

  if (sent) {
    return (
      <div role="status" className="p-4 border border-green-300 bg-green-50 rounded-md">
        <h2 className="text-lg font-semibold text-green-900">Message sent</h2>
        <p className="text-sm text-green-900 mt-1">
          Thank you. We have your message{loggedIn ? ' and will write to the email address on your account' : ' and will write to the address you gave'}.
          A confirmation is on its way.
        </p>
        <button type="button" onClick={() => setSent(false)} className="mt-3 text-sm underline text-green-900">
          Send another message
        </button>
      </div>
    );
  }

  return (
    <form onSubmit={submit} className="space-y-4" aria-label="Contact form">
      {allowAbuse && (
        <fieldset>
          <legend className="block text-sm font-medium text-gray-700">What is this about?</legend>
          <div className="mt-1 space-y-1">
            <label className="flex items-center gap-2 text-sm text-gray-800">
              <input type="radio" name="kind" value="support" checked={kind === 'support'} onChange={() => setKind('support')} />
              Support and feedback: a question, a problem with the site, or an idea
            </label>
            <label className="flex items-center gap-2 text-sm text-gray-800">
              <input type="radio" name="kind" value="abuse" checked={kind === 'abuse'} onChange={() => setKind('abuse')} />
              Policy and abuse: report something that breaks the rules
            </label>
          </div>
        </fieldset>
      )}

      {!loggedIn && (
        <div>
          <label htmlFor="support-email" className="block text-sm font-medium text-gray-700">Your email address (required)</label>
          <input id="support-email" type="email" required autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)}
            aria-describedby="support-email-help" className={FIELD} />
          <p id="support-email-help" className="text-xs text-gray-600 mt-1">Only the people who answer messages see it. It is how we write back.</p>
        </div>
      )}
      {loggedIn && (
        <p className="text-sm text-gray-700">We will write back to the email address on your account.</p>
      )}

      {kind === 'abuse' && (
        <div>
          <label htmlFor="support-url" className="block text-sm font-medium text-gray-700">Address of the page (optional)</label>
          <input id="support-url" type="url" maxLength={LIMITS.url} value={url} onChange={(e) => setUrl(e.target.value)}
            placeholder="https://" className={FIELD} />
        </div>
      )}

      <div>
        <label htmlFor="support-subject" className="block text-sm font-medium text-gray-700">Subject</label>
        <input id="support-subject" type="text" required maxLength={LIMITS.subject} value={subject} onChange={(e) => setSubject(e.target.value)} className={FIELD} />
      </div>

      <div>
        <label htmlFor="support-body" className="block text-sm font-medium text-gray-700">Message</label>
        <textarea id="support-body" required rows={8} maxLength={LIMITS.body} value={body} onChange={(e) => setBody(e.target.value)}
          aria-describedby="support-body-help" className={FIELD} />
        <p id="support-body-help" className="text-xs text-gray-600 mt-1">Plain text only, up to {LIMITS.body.toLocaleString('en-US')} characters.</p>
      </div>

      {error && <p role="alert" className="text-sm text-red-700">{error}</p>}
      <button type="submit" disabled={busy}
        className="py-2 px-4 border border-transparent rounded-md text-sm font-medium text-white bg-orange-700 hover:bg-orange-800 focus:outline-none focus:ring-2 focus:ring-orange-500 disabled:opacity-50">
        {busy ? 'Sending…' : 'Send'}
      </button>
    </form>
  );
}
