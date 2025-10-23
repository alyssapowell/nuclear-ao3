'use client';

import { useState } from 'react';
import { ArrowDownTrayIcon, Cog6ToothIcon, DocumentArrowDownIcon } from '@heroicons/react/24/outline';
import ExportModal from './ExportModal';
import ExportProgress from './ExportProgress';

interface ExportButtonProps {
  workId: string;
  workTitle: string;
  authToken?: string;
}

export default function ExportButton({ workId, workTitle, authToken }: ExportButtonProps) {
  const [showModal, setShowModal] = useState(false);
  const [showQuickMenu, setShowQuickMenu] = useState(false);
  const [activeExports, setActiveExports] = useState<string[]>([]);

  const handleQuickExport = async (format: 'epub' | 'mobi' | 'pdf') => {
    setShowQuickMenu(false);
    
    try {
      const response = await fetch('http://localhost:8086/api/v1/export', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': authToken ? `Bearer ${authToken}` : '',
        },
        body: JSON.stringify({
          work_id: workId,
          format: format,
          options: {
            include_metadata: true,
            include_tags: true,
            chapter_breaks: true,
            include_images: false,
            include_comments: false,
          }
        })
      });

      if (!response.ok) {
        throw new Error(`Export failed: ${response.statusText}`);
      }

      const exportData = await response.json();
      setActiveExports(prev => [...prev, exportData.id]);
      
      // Show success notification
      console.log(`${format.toUpperCase()} export started:`, exportData);
      
    } catch (error) {
      console.error('Export error:', error);
      alert(`Failed to start ${format.toUpperCase()} export. Please try again.`);
    }
  };

  const handleCustomExport = () => {
    setShowQuickMenu(false);
    setShowModal(true);
  };

  return (
    <>
      <div className="relative">
        {/* Main Export Button */}
        <button
          onClick={() => setShowQuickMenu(!showQuickMenu)}
          className="inline-flex items-center px-3 py-2 text-sm font-medium text-slate-700 hover:text-orange-600 border border-slate-300 rounded-md hover:border-orange-500 transition-colors duration-200"
          title="Export this work"
        >
          <ArrowDownTrayIcon className="w-4 h-4 mr-1.5" />
          Export
        </button>

        {/* Quick Export Menu */}
        {showQuickMenu && (
          <div className="absolute top-full left-0 mt-1 w-48 bg-white border border-slate-200 rounded-lg shadow-lg z-50">
            <div className="py-1">
              <div className="px-3 py-2 text-xs font-medium text-slate-500 uppercase tracking-wider border-b border-slate-100">
                Quick Export
              </div>
              
              <button
                onClick={() => handleQuickExport('epub')}
                className="w-full flex items-center px-3 py-2 text-sm text-slate-700 hover:bg-orange-50 hover:text-orange-600"
              >
                <DocumentArrowDownIcon className="w-4 h-4 mr-2" />
                EPUB
                <span className="ml-auto text-xs text-slate-500">E-readers</span>
              </button>
              
              <button
                onClick={() => handleQuickExport('mobi')}
                className="w-full flex items-center px-3 py-2 text-sm text-slate-700 hover:bg-orange-50 hover:text-orange-600"
              >
                <DocumentArrowDownIcon className="w-4 h-4 mr-2" />
                MOBI
                <span className="ml-auto text-xs text-slate-500">Kindle</span>
              </button>
              
              <button
                onClick={() => handleQuickExport('pdf')}
                className="w-full flex items-center px-3 py-2 text-sm text-slate-700 hover:bg-orange-50 hover:text-orange-600"
              >
                <DocumentArrowDownIcon className="w-4 h-4 mr-2" />
                PDF
                <span className="ml-auto text-xs text-slate-500">Print</span>
              </button>
              
              <div className="border-t border-slate-100 mt-1">
                <button
                  onClick={handleCustomExport}
                  className="w-full flex items-center px-3 py-2 text-sm text-slate-700 hover:bg-orange-50 hover:text-orange-600"
                >
                  <Cog6ToothIcon className="w-4 h-4 mr-2" />
                  Custom Options...
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Click outside to close */}
        {showQuickMenu && (
          <div 
            className="fixed inset-0 z-40"
            onClick={() => setShowQuickMenu(false)}
          />
        )}
      </div>

      {/* Export Progress Panels */}
      {activeExports.map(exportId => (
        <ExportProgress
          key={exportId}
          exportId={exportId}
          onComplete={(id) => setActiveExports(prev => prev.filter(eid => eid !== id))}
          onClose={(id) => setActiveExports(prev => prev.filter(eid => eid !== id))}
        />
      ))}

      {/* Custom Export Modal */}
      {showModal && (
        <ExportModal
          workId={workId}
          workTitle={workTitle}
          authToken={authToken}
          onClose={() => setShowModal(false)}
          onExportStart={(exportId) => {
            setActiveExports(prev => [...prev, exportId]);
            setShowModal(false);
          }}
        />
      )}
    </>
  );
}