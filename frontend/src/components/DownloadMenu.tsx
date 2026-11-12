import { useState } from 'react';
import { DOWNLOAD_FORMATS, downloadWork, workDownloadUrl, type DownloadFormat } from '@/lib/api';

interface DownloadMenuProps {
  workId: string;
  authToken?: string;
}

// AO3's Download menu: one entry per format the server can make. Each entry
// is a real link, so it works without scripts for a logged-out reader; with
// a login the file is fetched with it, because members-only works need it.
export default function DownloadMenu({ workId, authToken }: DownloadMenuProps) {
  const [open, setOpen] = useState(false);
  const [error, setError] = useState('');

  const handleClick = async (event: React.MouseEvent, format: DownloadFormat) => {
    if (!authToken) return; // let the link do its job
    event.preventDefault();
    setError('');
    try {
      await downloadWork(workId, format, authToken);
      setOpen(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Download failed');
    }
  };

  return (
    <div className="relative inline-block">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
        className="px-3 py-1.5 text-sm border border-slate-300 rounded-md bg-white text-slate-700 hover:bg-slate-50"
      >
        Download
      </button>
      {open && (
        <ul
          role="menu"
          aria-label="Download formats"
          className="absolute z-10 mt-1 w-40 rounded-md border border-slate-200 bg-white shadow-lg py-1"
        >
          {DOWNLOAD_FORMATS.map(({ ext, label }) => (
            <li key={ext} role="none">
              <a
                role="menuitem"
                href={workDownloadUrl(workId, ext)}
                download
                onClick={(e) => handleClick(e, ext)}
                className="block px-3 py-1.5 text-sm text-slate-700 hover:bg-slate-50"
              >
                {label}
              </a>
            </li>
          ))}
          {error && (
            <li role="alert" className="px-3 py-1.5 text-xs text-red-700">
              {error}
            </li>
          )}
        </ul>
      )}
    </div>
  );
}
