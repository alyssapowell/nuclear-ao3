'use client';

import { useState } from 'react';
import { Download, FileText, BookOpen, Loader2 } from 'lucide-react';

interface ExportButtonsProps {
  workId: string;
  workTitle: string;
  className?: string;
}

interface ExportRequest {
  workId: string;
  format: 'epub' | 'mobi' | 'pdf';
  options?: {
    includeImages?: boolean;
    customStyling?: string;
    fontFamily?: string;
    fontSize?: string;
    chapterBreaks?: boolean;
  };
}

export default function ExportButtons({ workId, workTitle, className = '' }: ExportButtonsProps) {
  const [loadingFormat, setLoadingFormat] = useState<string | null>(null);
  const [downloadUrl, setDownloadUrl] = useState<string | null>(null);

  const requestExport = async (format: 'epub' | 'mobi' | 'pdf') => {
    setLoadingFormat(format);
    
    try {
      const exportRequest: ExportRequest = {
        workId,
        format,
        options: {
          includeImages: true,
          chapterBreaks: true,
          fontFamily: 'Georgia, serif',
          fontSize: '12pt'
        }
      };

      const response = await fetch('/api/v1/export/request', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(exportRequest)
      });

      if (!response.ok) {
        throw new Error('Export request failed');
      }

      const result = await response.json();
      
      if (result.downloadUrl) {
        // Direct download available
        window.open(result.downloadUrl, '_blank');
      } else if (result.exportId) {
        // Export is being processed, poll for completion
        pollExportStatus(result.exportId, format);
      }
    } catch (error) {
      console.error('Export failed:', error);
      alert(`Failed to export ${format.toUpperCase()}. Please try again.`);
    } finally {
      setLoadingFormat(null);
    }
  };

  const pollExportStatus = async (exportId: string, format: string) => {
    const maxAttempts = 30; // 30 seconds maximum wait
    let attempts = 0;

    const poll = async () => {
      try {
        const response = await fetch(`/api/v1/export/status/${exportId}`);
        const status = await response.json();

        if (status.status === 'completed' && status.downloadUrl) {
          window.open(status.downloadUrl, '_blank');
          return;
        } else if (status.status === 'failed') {
          throw new Error('Export processing failed');
        } else if (attempts < maxAttempts) {
          attempts++;
          setTimeout(poll, 1000); // Poll every second
        } else {
          throw new Error('Export timed out');
        }
      } catch (error) {
        console.error('Export status check failed:', error);
        alert(`Export ${format.toUpperCase()} processing failed. Please try again.`);
      }
    };

    poll();
  };

  const getIcon = (format: string) => {
    switch (format) {
      case 'epub':
        return <BookOpen className="w-4 h-4" />;
      case 'mobi':
        return <FileText className="w-4 h-4" />;
      case 'pdf':
        return <FileText className="w-4 h-4" />;
      default:
        return <Download className="w-4 h-4" />;
    }
  };

  const getFormatDescription = (format: string) => {
    switch (format) {
      case 'epub':
        return 'Compatible with most e-readers and reading apps';
      case 'mobi':
        return 'Kindle-compatible format';
      case 'pdf':
        return 'Print-friendly PDF document';
      default:
        return '';
    }
  };

  return (
    <div className={`space-y-4 ${className}`}>
      <div className="bg-slate-50 border border-slate-200 rounded-lg p-4">
        <h3 className="text-lg font-semibold text-slate-900 mb-2 flex items-center gap-2">
          <Download className="w-5 h-5" />
          Download Options
        </h3>
        <p className="text-sm text-slate-600 mb-4">
          Download "{workTitle}" in your preferred format. Files expire after 24 hours for security.
        </p>
        
        <div className="grid gap-3 sm:grid-cols-3">
          {(['epub', 'mobi', 'pdf'] as const).map((format) => (
            <button
              key={format}
              onClick={() => requestExport(format)}
              disabled={loadingFormat !== null}
              className="flex flex-col items-center gap-2 p-4 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 hover:border-slate-400 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loadingFormat === format ? (
                <Loader2 className="w-6 h-6 animate-spin text-orange-500" />
              ) : (
                <div className="flex items-center gap-2 text-slate-700">
                  {getIcon(format)}
                  <span className="font-medium text-sm uppercase">{format}</span>
                </div>
              )}
              <span className="text-xs text-slate-500 text-center">
                {getFormatDescription(format)}
              </span>
            </button>
          ))}
        </div>
        
        <div className="mt-4 text-xs text-slate-500 bg-slate-100 rounded p-3">
          <div className="flex items-start gap-2">
            <span>ℹ️</span>
            <div>
              <p className="font-medium">Export Features:</p>
              <ul className="mt-1 space-y-1">
                <li>• Files include all chapters and metadata</li>
                <li>• Images and external media are preserved where possible</li>
                <li>• Downloads expire after 24 hours for privacy and security</li>
                <li>• EPUB works with most e-readers (iPad, Kobo, Nook, etc.)</li>
                <li>• MOBI works with Kindle devices and apps</li>
                <li>• PDF is best for printing or desktop reading</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}