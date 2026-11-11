import { Link } from 'react-router-dom';
import { getPseudUrl } from '@/lib/urls';

// How an author is shown anywhere on the site: the pseud's name, linking to
// the pseud's page. Never anything about the account behind it.

export interface PseudRefLike {
  id?: string | null;
  name: string;
}

export function PseudLink({ pseud, className }: { pseud: PseudRefLike; className?: string }) {
  if (!pseud.id) return <span className={className}>{pseud.name}</span>;
  return (
    <Link to={getPseudUrl(pseud.id)} className={className ?? 'text-blue-700 hover:text-blue-800 hover:underline'} rel="author">
      {pseud.name}
    </Link>
  );
}

/**
 * A list of pseuds separated by commas. With none (a work that hides its
 * creator) it says "Anonymous", or shows `fallback` text when given one.
 */
export function PseudList({ pseuds, fallback, className }: { pseuds?: PseudRefLike[] | null; fallback?: string; className?: string }) {
  if (!pseuds || pseuds.length === 0) {
    return <span>{fallback || 'Anonymous'}</span>;
  }
  return (
    <>
      {pseuds.map((p, i) => (
        <span key={p.id ?? `${p.name}-${i}`}>
          <PseudLink pseud={p} className={className} />
          {i < pseuds.length - 1 && ', '}
        </span>
      ))}
    </>
  );
}

export default PseudLink;
