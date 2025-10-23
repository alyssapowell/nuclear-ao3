'use client';

import { useState } from 'react';
import { ArrowDownTrayIcon, Cog6ToothIcon, DocumentArrowDownIcon, DevicePhoneMobileIcon, XMarkIcon, InformationCircleIcon } from '@heroicons/react/24/outline';
import ExportModal from './ExportModal';
import ExportProgress from './ExportProgress';

interface RespectfulExportButtonProps {
  workId: string;
  workTitle: string;
  authToken?: string;
  authorOfflinePreference: 'files_and_pwa' | 'pwa_only' | 'none';
  isAuthor?: boolean;
}

export default function RespectfulExportButton({ 
  workId, 
  workTitle, 
  authToken, 
  authorOfflinePreference,
  isAuthor = false 
}: RespectfulExportButtonProps) {
  const [showModal, setShowModal] = useState(false);
  const [showQuickMenu, setShowQuickMenu] = useState(false);
  const [activeExports, setActiveExports] = useState<string[]>([]);
  const [showEducationalModal, setShowEducationalModal] = useState(false);

  // Determine what's allowed based on author's preference
  const allowsDownloads = authorOfflinePreference === 'files_and_pwa';
  const allowsPWA = authorOfflinePreference === 'files_and_pwa' || authorOfflinePreference === 'pwa_only';
  const isOnlineOnly = authorOfflinePreference === 'none';

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

  const handlePWAOfflineReading = () => {
    // TODO: Implement PWA offline reading
    alert('PWA Offline reading feature coming soon!');
  };

  const getButtonText = () => {
    if (isOnlineOnly) return 'Online Only';
    if (allowsDownloads) return 'Export';
    return 'Read Offline';
  };

  const getButtonIcon = () => {
    if (isOnlineOnly) return XMarkIcon;
    if (allowsDownloads) return ArrowDownTrayIcon;
    return DevicePhoneMobileIcon;
  };

  const getButtonColor = () => {
    if (isOnlineOnly) return 'text-slate-400 border-slate-200 cursor-not-allowed';
    return 'text-slate-700 hover:text-orange-600 border-slate-300 hover:border-orange-500';
  };

  const ButtonIcon = getButtonIcon();

  if (isOnlineOnly && !isAuthor) {
    return (
      <div className="relative">
        <button
          onClick={() => setShowEducationalModal(true)}
          className="inline-flex items-center px-3 py-2 text-sm font-medium text-slate-400 border border-slate-200 rounded-md cursor-help"
          title="This author has chosen online-only reading"
          data-testid="respectful-export-button"
        >
          <XMarkIcon className="w-4 h-4 mr-1.5" />
          Online Only
        </button>

        {/* Educational Modal */}
        {showEducationalModal && (
          <>
            <div className="fixed inset-0 bg-black bg-opacity-50 z-50" onClick={() => setShowEducationalModal(false)} />
            <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
              <div className="bg-white rounded-lg shadow-xl max-w-md w-full p-6">
                <div className="flex items-start space-x-3">
                  <InformationCircleIcon className="w-6 h-6 text-blue-600 flex-shrink-0 mt-0.5" />
                  <div>
                    <h3 className="text-lg font-semibold text-slate-900 mb-2">
                      Author's Choice: Online Only
                    </h3>
                    <p className="text-sm text-slate-600 mb-4">
                      The author of this work has chosen to make it available for online reading only. 
                      This means no downloads or offline reading are available.
                    </p>
                    <p className="text-sm text-slate-600 mb-4">
                      This choice helps authors maintain control over their content and is part of 
                      respecting creator autonomy in fanfiction communities.
                    </p>
                    <div className="flex justify-end">
                      <button
                        onClick={() => setShowEducationalModal(false)}
                        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
                        data-testid="close-modal"
                      >
                        Understood
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </>
        )}
      </div>
    );
  }

  return (
    <>
      <div className="relative">
        {/* Main Export Button */}
        <button
          onClick={() => setShowQuickMenu(!showQuickMenu)}
          className={`inline-flex items-center px-3 py-2 text-sm font-medium border rounded-md transition-colors duration-200 ${getButtonColor()}`}
          title={allowsDownloads ? "Export this work" : "Read offline in app"}
          disabled={isOnlineOnly && !isAuthor}
          data-testid="respectful-export-button"
        >
          <ButtonIcon className="w-4 h-4 mr-1.5" />
          {getButtonText()}
        </button>

        {/* Quick Export/Offline Menu */}
        {showQuickMenu && !isOnlineOnly && (
          <div className="absolute top-full left-0 mt-1 w-64 bg-white border border-slate-200 rounded-lg shadow-lg z-50" data-testid="export-modal">
            <div className="py-1">
              {/* Author's Choice Indicator */}
              <div className="px-3 py-2 text-xs font-medium text-slate-500 uppercase tracking-wider border-b border-slate-100 bg-slate-50">
                Author allows: {allowsDownloads ? 'Downloads + Offline Reading' : 'App Offline Reading Only'}
              </div>
              
              {/* File Downloads (if allowed) */}
              {allowsDownloads && (
                <>
                  <div className="px-3 py-1 text-xs font-medium text-slate-500 uppercase tracking-wider">
                    Download Files
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
          onClick={() => setShowEducationalModal(false)}
          className="text-slate-400 hover:text-slate-600"
          data-testid="close-modal"
        >
          <XMarkIcon className="w-4 h-4" />
        </button>
                  
                  <div className="border-t border-slate-100 my-1"></div>
                </>
              )}

              {/* PWA Offline Reading (if allowed) */}
              {allowsPWA && (
                <>
                  <div className="px-3 py-1 text-xs font-medium text-slate-500 uppercase tracking-wider">
                    App Offline Reading
                  </div>
                  <button
                    onClick={handlePWAOfflineReading}
                    className="w-full flex items-center px-3 py-2 text-sm text-slate-700 hover:bg-blue-50 hover:text-blue-600"
                  >
                    <DevicePhoneMobileIcon className="w-4 h-4 mr-2" />
                    Read Offline
                    <span className="ml-auto text-xs text-slate-500">Respectful</span>
                  </button>
                  
                  <div className="border-t border-slate-100 my-1"></div>
                </>
              )}

              {/* Custom Options */}
              {allowsDownloads && (
                <button
                  onClick={handleCustomExport}
                  className="w-full flex items-center px-3 py-2 text-sm text-slate-700 hover:bg-orange-50 hover:text-orange-600"
                >
                  <Cog6ToothIcon className="w-4 h-4 mr-2" />
                  Custom Options...
                </button>
              )}

              {/* Educational Footer */}
              <div className="px-3 py-2 text-xs text-slate-500 bg-slate-50 border-t border-slate-100">
                {allowsDownloads ? (
                  'Files can be kept permanently • Respecting author choice'
                ) : (
                  'Respects author deletions • Temporary offline access'
                )}
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

      {/* Custom Export Modal (only for file downloads) */}
      {showModal && allowsDownloads && (
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