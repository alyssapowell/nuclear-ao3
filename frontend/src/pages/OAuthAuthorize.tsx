// "Sign in with this archive": the page an app sends a person to. The app
// can't send the person's login on a redirect, so this page does it: it logs
// them in if needed, asks the server about the request, shows the consent
// screen, and sends the person on to the app with the answer.
import { useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { getAuthToken } from '@/utils/auth';
import { answerConsent, authorize, nav, type ConsentScreen } from '@/lib/oauth';

const BUTTON =
  'py-2 px-4 border rounded-md shadow-sm text-sm font-medium focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500 disabled:opacity-50 disabled:cursor-not-allowed';

function ConsentCard({ screen }: { screen: ConsentScreen }) {
  const [answerError, setAnswerError] = useState('');
  const asksProfile = screen.scopes.some((s) => s.name === 'profile');
  const asksEmail = screen.scopes.some((s) => s.name === 'email');
  const pseuds = screen.pseuds ?? [];
  const [pseudId, setPseudId] = useState(screen.default_pseud_id ?? pseuds[0]?.id ?? '');
  const [shareEmail, setShareEmail] = useState(false);
  const [busy, setBusy] = useState(false);

  const answer = async (approved: boolean) => {
    setBusy(true);
    setAnswerError('');
    const body: { approved: boolean; pseud_id?: string; share_email?: boolean } = { approved };
    if (approved) {
      if (asksProfile && pseudId) body.pseud_id = pseudId;
      if (asksEmail) body.share_email = shareEmail;
    }
    const result = await answerConsent(screen.consent_id, body);
    if (result.ok && result.data?.redirect) {
      // The server decided where the app wants the person to land.
      nav.to(result.data.redirect);
      return;
    }
    setAnswerError(result.error || 'Could not send your answer. Please go back to the app and try again.');
    setBusy(false);
  };

  const website = screen.app.website && screen.app.website.startsWith('https://') ? screen.app.website : '';
  const logo = screen.app.logo_url && screen.app.logo_url.startsWith('https://') ? screen.app.logo_url : '';

  return (
    <div>
      <div className="flex items-center gap-4 mb-4">
        {logo && <img src={logo} alt={`${screen.app.name} logo`} className="h-14 w-14 rounded object-contain" />}
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Allow {screen.app.name} to use your account?</h1>
          {website && (
            <p className="text-sm text-gray-700">
              Website:{' '}
              <a href={website} target="_blank" rel="noopener noreferrer" className="text-orange-700 underline">
                {website}
              </a>
            </p>
          )}
        </div>
      </div>

      <p className="mb-4 text-sm text-green-800 bg-green-50 border border-green-200 rounded-md p-3">
        This app was reviewed by the archive&apos;s admins.
      </p>

      <h2 className="text-lg font-semibold text-gray-900 mb-2">If you allow it, this app will be able to:</h2>
      <ul className="list-disc pl-6 mb-4 text-sm text-gray-800 space-y-1">
        {screen.scopes.map((s) => (
          <li key={s.name}>{s.name === 'openid' ? 'Know that it is you signing in, using an id that is different for every app' : s.description || s.name}</li>
        ))}
      </ul>
      <p className="mb-4 text-sm text-gray-700">
        It will never see your password. It will not be told your email address or account, and it gets a different
        sign-in id from every other app. You can disconnect it at any time in your profile settings under
        &quot;Connected apps&quot;.
      </p>

      {asksProfile && (
        <div className="mb-4">
          <label htmlFor="pseud" className="block text-sm font-medium text-gray-700">
            Which pseud should this app see?
          </label>
          <select
            id="pseud"
            value={pseudId}
            onChange={(e) => setPseudId(e.target.value)}
            disabled={busy}
            className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500"
          >
            {pseuds.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
          <p className="mt-1 text-xs text-gray-600">The app sees only this pseud, not your other pseuds.</p>
        </div>
      )}

      {asksEmail && (
        <div className="mb-4 flex items-start gap-2">
          <input
            id="share-email"
            type="checkbox"
            checked={shareEmail}
            onChange={(e) => setShareEmail(e.target.checked)}
            disabled={busy}
            className="mt-1 h-4 w-4 rounded border-gray-300 text-orange-600 focus:ring-orange-500"
          />
          <label htmlFor="share-email" className="text-sm text-gray-800">
            Also share the email address of my account with this app
            <span className="block text-xs text-gray-600">
              Leave this unticked and the app can still sign you in, but it will not get your email address.
            </span>
          </label>
        </div>
      )}

      {answerError && (
        <p role="alert" className="mb-3 text-red-700 text-sm">
          {answerError}
        </p>
      )}

      <div className="flex gap-3">
        <button
          type="button"
          onClick={() => answer(true)}
          disabled={busy}
          className={`${BUTTON} border-transparent text-white bg-orange-600 hover:bg-orange-700`}
        >
          Allow
        </button>
        <button
          type="button"
          onClick={() => answer(false)}
          disabled={busy}
          className={`${BUTTON} border-gray-300 text-gray-700 bg-white hover:bg-gray-50`}
        >
          Deny
        </button>
      </div>
    </div>
  );
}

export default function OAuthAuthorizePage() {
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const query = params.toString();
  const [screen, setScreen] = useState<ConsentScreen | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    // Not logged in: log in first, then come back to this same request.
    if (!getAuthToken()) {
      navigate(`/auth/login?redirect=${encodeURIComponent(`/oauth/authorize?${query}`)}`, { replace: true });
      return;
    }
    (async () => {
      const result = await authorize(query);
      if (cancelled) return;
      if (result.status === 401) {
        navigate(`/auth/login?redirect=${encodeURIComponent(`/oauth/authorize?${query}`)}`, { replace: true });
        return;
      }
      if (result.ok && result.data?.redirect) {
        nav.to(result.data.redirect);
        return;
      }
      if (result.ok && result.data?.consent) {
        setScreen(result.data.consent);
      } else {
        // Problems that must not be sent to the app (an unknown app, a wrong
        // return address) are shown here and nowhere else.
        setError(result.error || 'This sign-in request is not valid.');
      }
      setLoading(false);
    })();
    return () => {
      cancelled = true;
    };
  }, [navigate, query]);

  return (
    <div className="max-w-xl mx-auto mt-8 p-6 bg-white rounded-lg shadow-md">
      {loading && !error && <p className="text-sm text-gray-700">Checking this sign-in request…</p>}
      {error && (
        <>
          <h1 className="text-2xl font-bold text-gray-900 mb-4">We can&apos;t continue</h1>
          <p role="alert" className="text-red-700 text-sm">
            {error}
          </p>
          <p className="mt-4 text-sm text-gray-700">
            Nothing was shared. Go back to the app you came from and try again. If this keeps happening, tell the
            app&apos;s owner.
          </p>
        </>
      )}
      {screen && <ConsentCard screen={screen} />}
    </div>
  );
}
