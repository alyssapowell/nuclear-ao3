import { useEffect, useState } from 'react';
import { getMyRoles } from '@/lib/admin';
import { getAuthToken } from '@/utils/auth';

/**
 * The signed-in person's own staff roles, read from the server (GET
 * /api/v1/auth/me), so they are never out of date and never kept in the
 * browser. Used only to show or hide admin links and pages: the server checks
 * the role again on every admin request.
 *
 * `loaded` is false until the server has answered. Someone who is logged out,
 * or whose login the server refuses, has no roles.
 */
export function useRoles(): { roles: string[]; loaded: boolean } {
  const [state, setState] = useState<{ roles: string[]; loaded: boolean }>({ roles: [], loaded: false });

  useEffect(() => {
    let current = true;
    const load = async () => {
      let token: string | null = null;
      try {
        token = getAuthToken();
      } catch {
        // Storage is blocked or unreadable: treat as logged out.
      }
      if (!token) {
        if (current) setState({ roles: [], loaded: true });
        return;
      }
      const r = await getMyRoles();
      if (!current) return;
      const roles = r.ok && Array.isArray(r.data?.roles) ? r.data.roles : [];
      setState({ roles, loaded: true });
    };
    load();
    window.addEventListener('authChange', load);
    return () => {
      current = false;
      window.removeEventListener('authChange', load);
    };
  }, []);

  return state;
}
