'use client';

import { useState } from 'react';
import { giveKudos } from '@/lib/api';

interface KudosButtonProps {
  workId: string;
  initialKudos: number;
  hasGivenKudos?: boolean;
  authToken?: string;
}

export default function KudosButton({ workId, initialKudos, hasGivenKudos = false, authToken }: KudosButtonProps) {
  const [kudosCount, setKudosCount] = useState(initialKudos);
  const [hasKudos, setHasKudos] = useState(hasGivenKudos);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleGiveKudos = async () => {
    if (hasKudos || loading) return;

    try {
      setLoading(true);
      setError(null);
      
      await giveKudos(workId, authToken);
      
      setKudosCount(prev => prev + 1);
      setHasKudos(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to give kudos');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex items-center gap-3">
      <button
        onClick={handleGiveKudos}
        disabled={hasKudos || loading}
        className={`flex items-center gap-2 px-4 py-2 rounded-lg transition-colors ${
          hasKudos 
            ? 'bg-red-100 text-red-700 cursor-not-allowed' 
            : 'bg-red-50 hover:bg-red-100 text-red-600 hover:text-red-700'
        }`}
        title={hasKudos ? 'You have already left kudos' : 'Leave kudos'}
      >
        <svg 
          className={`w-5 h-5 ${loading ? 'animate-pulse' : ''}`} 
          fill={hasKudos ? 'currentColor' : 'none'} 
          stroke="currentColor" 
          viewBox="0 0 24 24"
        >
          <path 
            strokeLinecap="round" 
            strokeLinejoin="round" 
            strokeWidth={hasKudos ? 0 : 2} 
            d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" 
          />
        </svg>
        <span className="font-medium">
          {loading ? 'Giving...' : hasKudos ? 'Kudos Given' : 'Give Kudos'}
        </span>
        <span className="text-sm">({kudosCount})</span>
      </button>

      {error && (
        <div className="text-sm text-red-600">
          {error}
        </div>
      )}
    </div>
  );
}