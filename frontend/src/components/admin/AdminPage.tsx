// The frame every admin screen shares: a heading, a way back to the admin
// home, and the role check. A person without the role sees a plain refusal
// and the screen makes no admin requests; the server refuses them anyway.
import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { hasRole, type Role } from '@/lib/admin';
import { useRoles } from '@/hooks/useRoles';

interface Props {
  title: string;
  intro?: string;
  need: Role;
  /** Who the screen is for, in words, shown to people without the role. */
  audience: string;
  home?: boolean;
  children: ReactNode;
}

export default function AdminPage({ title, intro, need, audience, home, children }: Props) {
  const { roles, loaded } = useRoles();
  const allowed = loaded && hasRole(roles, need);
  return (
    <div className="max-w-4xl mx-auto mt-8 p-6 bg-white rounded-lg shadow-md">
      {!home && (
        <p className="mb-2 text-sm"><Link to="/admin" className="text-orange-800 underline">Admin home</Link></p>
      )}
      <h1 className="text-2xl font-bold text-gray-900 mb-2">{title}</h1>
      {!loaded && <p className="text-sm text-gray-700">Loading…</p>}
      {loaded && !allowed && <p role="alert" className="text-sm text-red-700">This page is for {audience}.</p>}
      {allowed && (
        <>
          {intro && <p className="mb-4 text-sm text-gray-700">{intro}</p>}
          {children}
        </>
      )}
    </div>
  );
}
