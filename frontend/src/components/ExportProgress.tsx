'use client';

import { useState, useEffect } from 'react';
import { XMarkIcon, ArrowDownTrayIcon, CheckCircleIcon, ExclamationCircleIcon, ClockIcon } from '@heroicons/react/24/outline';

interface ExportProgressProps {
  exportId: string;
  onComplete: (exportId: string) => void;
  onClose: (exportId: string) => void;
}

interface ExportStatus {
  id: string;
  work_id: string;
  format: string;
  status: string; // pending, processing, completed, failed, expired
  progress: number; // 0-100
  download_url?: string;
  error?: string;
  created_at: string;
  expires_at: string;
  ttl_seconds: number;
}

export default function ExportProgress({ exportId, onComplete, onClose }: ExportProgressProps) {
  const [exportStatus, setExportStatus] = useState<ExportStatus | null>(null);
  const [timeLeft, setTimeLeft] = useState<string>('');

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const response = await fetch(`http://localhost:8086/api/v1/export/${exportId}`);
        if (response.ok) {
          const status = await response.json();
          setExportStatus(status);
          
          if (status.status === 'completed' || status.status === 'failed') {
            // Stop polling after completion
            return;
          }
        }
      } catch (error) {
        console.error('Failed to fetch export status:', error);
      }
    };

    // Initial fetch
    fetchStatus();

    // Poll every 2 seconds for updates
    const interval = setInterval(fetchStatus, 2000);

    return () => clearInterval(interval);
  }, [exportId]);

  useEffect(() => {
    if (!exportStatus) return;

    // Update time left
    const updateTimeLeft = () => {
      const now = new Date().getTime();
      const expiresAt = new Date(exportStatus.expires_at).getTime();
      const timeDiff = expiresAt - now;

      if (timeDiff <= 0) {
        setTimeLeft('Expired');
        return;
      }

      const hours = Math.floor(timeDiff / (1000 * 60 * 60));
      const minutes = Math.floor((timeDiff % (1000 * 60 * 60)) / (1000 * 60));
      
      if (hours > 0) {
        setTimeLeft(`${hours}h ${minutes}m`);
      } else {
        setTimeLeft(`${minutes}m`);
      }
    };

    updateTimeLeft();
    const timeInterval = setInterval(updateTimeLeft, 60000); // Update every minute

    return () => clearInterval(timeInterval);
  }, [exportStatus]);

  const handleDownload = () => {
    if (exportStatus?.download_url) {
      const link = document.createElement('a');
      link.href = exportStatus.download_url;
      link.download = `work-${exportStatus.work_id}.${exportStatus.format}`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    }
  };

  const handleRefreshTTL = async () => {
    try {
      const response = await fetch(`http://localhost:8086/api/v1/export/${exportId}/refresh`, {
        method: 'POST',
      });
      
      if (response.ok) {
        // Refresh the export status to get updated TTL
        const statusResponse = await fetch(`http://localhost:8086/api/v1/export/${exportId}`);
        if (statusResponse.ok) {
          const updatedStatus = await statusResponse.json();
          setExportStatus(updatedStatus);
        }
      }
    } catch (error) {
      console.error('Failed to refresh TTL:', error);
    }
  };

  if (!exportStatus) {
    return null;
  }

  const getStatusIcon = () => {
    switch (exportStatus.status) {
      case 'completed':
        return <CheckCircleIcon className="w-5 h-5 text-green-500" />;
      case 'failed':
        return <ExclamationCircleIcon className="w-5 h-5 text-red-500" />;
      default:
        return (
          <div className="w-5 h-5 border-2 border-orange-500 border-t-transparent rounded-full animate-spin" />
        );
    }
  };

  const getStatusColor = () => {
    switch (exportStatus.status) {
      case 'completed':
        return 'border-green-200 bg-green-50';
      case 'failed':
        return 'border-red-200 bg-red-50';
      case 'processing':
        return 'border-orange-200 bg-orange-50';
      default:
        return 'border-slate-200 bg-slate-50';
    }
  };

  return (
    <div className={`fixed bottom-4 right-4 w-80 border rounded-lg shadow-lg p-4 z-50 ${getStatusColor()}`}>
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center space-x-2">
          {getStatusIcon()}
          <div>
            <p className="text-sm font-medium text-slate-900">
              {exportStatus.format.toUpperCase()} Export
            </p>
            <p className="text-xs text-slate-600 capitalize">
              {exportStatus.status === 'processing' ? 'Processing...' : exportStatus.status}
            </p>
          </div>
        </div>
        
        <button
          onClick={() => onClose(exportId)}
          className="text-slate-400 hover:text-slate-600"
        >
          <XMarkIcon className="w-4 h-4" />
        </button>
      </div>

      {/* Progress Bar */}
      {exportStatus.status === 'processing' && (
        <div className="mb-3">
          <div className="flex justify-between text-xs text-slate-600 mb-1">
            <span>Progress</span>
            <span>{exportStatus.progress}%</span>
          </div>
          <div className="w-full bg-slate-200 rounded-full h-2">
            <div 
              className="bg-orange-500 h-2 rounded-full transition-all duration-500"
              style={{ width: `${exportStatus.progress}%` }}
            />
          </div>
        </div>
      )}

      {/* Error Message */}
      {exportStatus.status === 'failed' && exportStatus.error && (
        <div className="mb-3 p-2 bg-red-100 border border-red-200 rounded text-xs text-red-700">
          {exportStatus.error}
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center justify-between text-xs">
        {exportStatus.status === 'completed' && (
          <button
            onClick={handleDownload}
            className="flex items-center space-x-1 text-green-600 hover:text-green-700 font-medium"
          >
            <ArrowDownTrayIcon className="w-4 h-4" />
            <span>Download</span>
          </button>
        )}
        
        {exportStatus.status === 'completed' && (
          <div className="flex items-center space-x-2">
            <span className="text-slate-500">
              Expires in {timeLeft}
            </span>
            {timeLeft !== 'Expired' && (
              <button
                onClick={handleRefreshTTL}
                className="flex items-center text-xs text-orange-600 hover:text-orange-700 font-medium"
                title="Extend download time by 24 hours"
              >
                <ClockIcon className="w-3 h-3 mr-1" />
                +24h
              </button>
            )}
          </div>
        )}

        {exportStatus.status === 'processing' && (
          <span className="text-slate-500">
            Preparing your {exportStatus.format.toUpperCase()}...
          </span>
        )}
      </div>
    </div>
  );
}