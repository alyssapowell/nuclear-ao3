'use client';

import { useEffect, useState } from 'react';

export default function DebugAuthPage() {
  const [tokenInfo, setTokenInfo] = useState<any>(null);
  const [userInfo, setUserInfo] = useState<any>(null);

  useEffect(() => {
    if (typeof window !== 'undefined') {
      const token = localStorage.getItem('auth_token');
      const userData = localStorage.getItem('user');
      
      setUserInfo(userData ? JSON.parse(userData) : null);
      
      if (token) {
        try {
          const payload = JSON.parse(atob(token.split('.')[1]));
          setTokenInfo(payload);
        } catch (error) {
          setTokenInfo({ error: 'Invalid token format' });
        }
      }
    }
  }, []);

  return (
    <div className="max-w-4xl mx-auto p-6">
      <h1 className="text-2xl font-bold mb-6">Auth Debug Info</h1>
      
      <div className="space-y-6">
        <div className="bg-white p-4 rounded-lg shadow">
          <h2 className="text-lg font-semibold mb-2">JWT Token Payload:</h2>
          <pre className="bg-gray-100 p-2 rounded text-xs overflow-auto">
            {JSON.stringify(tokenInfo, null, 2)}
          </pre>
        </div>
        
        <div className="bg-white p-4 rounded-lg shadow">
          <h2 className="text-lg font-semibold mb-2">Stored User Data:</h2>
          <pre className="bg-gray-100 p-2 rounded text-xs overflow-auto">
            {JSON.stringify(userInfo, null, 2)}
          </pre>
        </div>
        
        <div className="bg-white p-4 rounded-lg shadow">
          <h2 className="text-lg font-semibold mb-2">Raw Token:</h2>
          <div className="bg-gray-100 p-2 rounded text-xs break-all">
            {typeof window !== 'undefined' ? localStorage.getItem('auth_token') : 'Loading...'}
          </div>
        </div>
      </div>
    </div>
  );
}