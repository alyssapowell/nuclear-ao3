import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/Button';

// A block is shown as the pseud that was picked, and nothing else about the
// account behind it.
interface BlockedPseud {
  pseud: { id: string; name: string };
  created_at: string;
}

type Tab = 'settings' | 'blocked-users' | 'moderation';

// Blocks are real (GET /blocks, DELETE /blocks/:pseud_id). Account preferences
// and the list of comments held for review have no server side yet, so those
// tabs say so rather than failing.
export default function PrivacySettingsPage() {
  const [activeTab, setActiveTab] = useState<Tab>('blocked-users');
  const [blocked, setBlocked] = useState<BlockedPseud[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (activeTab !== 'blocked-users') return;
    const token = localStorage.getItem('auth_token');
    if (!token) return;
    setLoading(true);
    setError(null);
    fetch('/api/v1/blocks', { headers: { Authorization: `Bearer ${token}` } })
      .then(async (response) => {
        if (!response.ok) throw new Error('Could not load your blocks.');
        const data = await response.json();
        setBlocked(data.blocks || []);
      })
      .catch((e) => setError(e instanceof Error ? e.message : 'Could not load your blocks.'))
      .finally(() => setLoading(false));
  }, [activeTab]);

  const unblock = async (pseudId: string) => {
    const token = localStorage.getItem('auth_token');
    if (!token) return;
    setError(null);
    try {
      const response = await fetch(`/api/v1/blocks/${pseudId}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!response.ok) throw new Error('Could not remove the block.');
      setBlocked((prev) => prev.filter((b) => b.pseud.id !== pseudId));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not remove the block.');
    }
  };

  const tabButton = (id: Tab, label: string) => (
    <button
      onClick={() => setActiveTab(id)}
      className={`flex items-center px-4 py-3 text-sm font-medium rounded-lg transition-colors ${
        activeTab === id
          ? 'bg-orange-100 text-orange-700 border border-orange-200'
          : 'text-gray-700 hover:text-gray-900 hover:bg-gray-50'
      }`}
    >
      {label}
    </button>
  );

  const notYet = (title: string, body: string) => (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold text-gray-900">{title}</h2>
      <div className="bg-gray-50 border border-gray-200 rounded-lg p-6" role="status">
        <p className="font-medium text-gray-900 mb-1">Not available yet</p>
        <p className="text-sm text-gray-700">{body}</p>
      </div>
    </div>
  );

  const renderBlocked = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-900">Blocked pseuds</h2>
        <span className="text-sm text-gray-700">{blocked.length} blocked</span>
      </div>

      {error && <p role="alert" className="text-sm text-red-700">{error}</p>}

      {loading ? (
        <div className="flex justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-orange-600"></div>
        </div>
      ) : blocked.length === 0 ? (
        <div className="text-center py-8">
          <h3 className="text-lg font-medium text-gray-900 mb-2">No blocked pseuds</h3>
          <p className="text-gray-700">You haven't blocked anyone yet. Use the Block button on a pseud's page.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {blocked.map((block) => (
            <div key={block.pseud.id} className="bg-white border border-gray-200 rounded-lg p-6">
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <h3 className="font-medium text-gray-900">{block.pseud.name}</h3>
                  <p className="text-sm text-gray-700 mt-1">
                    Blocked {new Date(block.created_at).toLocaleDateString()}
                  </p>
                </div>
                <Button onClick={() => unblock(block.pseud.id)} variant="outline" size="sm" className="ml-4">
                  Unblock
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6">
        <h3 className="font-semibold text-yellow-900 mb-2">About blocking</h3>
        <div className="text-sm text-yellow-800 space-y-2">
          <p>Blocking a pseud blocks the person behind it, under every pseud they have, and works both ways.</p>
          <p>You can still read each other's works, but comments are closed between you: neither of you can comment, reply or leave kudos on the other's works, and you won't see comments from the person you blocked.</p>
          <p>You won't be notified about anything they do.</p>
        </div>
      </div>
    </div>
  );

  return (
    <div className="max-w-6xl mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900 mb-2">Privacy & Safety</h1>
        <p className="text-gray-700">Blocked pseuds, preferences and comment moderation.</p>
      </div>

      <div className="flex flex-col lg:flex-row gap-8">
        <div className="lg:w-64 space-y-2">
          {tabButton('blocked-users', 'Blocked pseuds')}
          {tabButton('settings', 'Preferences')}
          {tabButton('moderation', 'Comment moderation')}
        </div>

        <div className="flex-1">
          {activeTab === 'blocked-users' && renderBlocked()}
          {activeTab === 'settings' &&
            notYet('Preferences', 'Account preferences, such as whether your email address is shown, are not available yet.')}
          {activeTab === 'moderation' &&
            notYet('Comment moderation', 'Reviewing comments held on your works is not available yet.')}
        </div>
      </div>
    </div>
  );
}
