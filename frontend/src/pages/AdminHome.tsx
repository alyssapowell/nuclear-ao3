// Admin home: one link to each admin screen the person's role allows.
import { Link } from 'react-router-dom';
import AdminPage from '@/components/admin/AdminPage';
import { hasRole, type Role } from '@/lib/admin';
import { useRoles } from '@/hooks/useRoles';

const SECTIONS: { to: string; title: string; text: string; need: Role }[] = [
  { to: '/admin/accounts', title: 'Accounts', text: 'Find an account, suspend or ban it, end its logins, and change roles (admins only).', need: 'moderator' },
  { to: '/admin/content', title: 'Content', text: 'Hide or unhide a work, chapter, series, bookmark or comment, and mark spam.', need: 'moderator' },
  { to: '/admin/reports', title: 'Reports', text: 'Abuse reports people sent about works and comments.', need: 'moderator' },
  { to: '/admin/audit', title: 'Audit log', text: 'Everything admins and moderators have done.', need: 'admin' },
  { to: '/admin/apps', title: 'App review', text: 'Apps people registered, waiting for approval.', need: 'admin' },
];

function Sections() {
  const { roles } = useRoles();
  return (
    <ul className="divide-y divide-gray-200">
      {SECTIONS.filter((s) => hasRole(roles, s.need)).map((s) => (
        <li key={s.to} className="py-3">
          <Link to={s.to} className="font-medium text-orange-800 underline">{s.title}</Link>
          <p className="text-sm text-gray-700">{s.text}</p>
        </li>
      ))}
    </ul>
  );
}

export default function AdminHomePage() {
  return (
    <AdminPage title="Admin" need="moderator" audience="the archive's admins and moderators" home
      intro="Every action here is checked again by the server and written to the audit log.">
      <Sections />
    </AdminPage>
  );
}
