'use client';

import { useState } from 'react';
import { XMarkIcon, DocumentArrowDownIcon, Cog6ToothIcon, PhotoIcon } from '@heroicons/react/24/outline';

interface ExportModalProps {
  workId: string;
  workTitle: string;
  authToken?: string;
  onClose: () => void;
  onExportStart: (exportId: string) => void;
}

interface ExportOptions {
  format: 'epub' | 'mobi' | 'pdf';
  include_metadata: boolean;
  include_tags: boolean;
  include_comments: boolean;
  include_images: boolean;
  chapter_breaks: boolean;
  font_family?: string;
  font_size?: string;
  cover_image?: string;
  custom_css?: string;
}

export default function ExportModal({ workId, workTitle, authToken, onClose, onExportStart }: ExportModalProps) {
  const [options, setOptions] = useState<ExportOptions>({
    format: 'epub',
    include_metadata: true,
    include_tags: true,
    include_comments: false,
    include_images: true,
    chapter_breaks: true,
    font_family: 'serif',
    font_size: '12',
  });

  const [isExporting, setIsExporting] = useState(false);
  const [selectedCoverImage, setSelectedCoverImage] = useState<File | null>(null);
  const [coverImagePreview, setCoverImagePreview] = useState<string | null>(null);

  const handleOptionChange = (key: keyof ExportOptions, value: any) => {
    setOptions(prev => ({ ...prev, [key]: value }));
  };

  const handleCoverImageChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      // Validate file type
      if (!file.type.startsWith('image/')) {
        alert('Please select an image file');
        return;
      }
      
      // Validate file size (max 5MB)
      if (file.size > 5 * 1024 * 1024) {
        alert('Image file must be smaller than 5MB');
        return;
      }

      setSelectedCoverImage(file);
      
      // Create preview
      const reader = new FileReader();
      reader.onload = (e) => {
        setCoverImagePreview(e.target?.result as string);
      };
      reader.readAsDataURL(file);
    }
  };

  const clearCoverImage = () => {
    setSelectedCoverImage(null);
    setCoverImagePreview(null);
    handleOptionChange('cover_image', undefined);
  };

  const handleExport = async () => {
    setIsExporting(true);
    
    try {
      // Prepare form data for file upload
      const formData = new FormData();
      formData.append('work_id', workId);
      formData.append('format', options.format);
      formData.append('options', JSON.stringify(options));
      
      // Add cover image if selected
      if (selectedCoverImage) {
        formData.append('cover_image', selectedCoverImage);
      }

      const response = await fetch('http://localhost:8086/api/v1/export', {
        method: 'POST',
        headers: {
          'Authorization': authToken ? `Bearer ${authToken}` : '',
          // Don't set Content-Type - let browser set it with boundary for FormData
        },
        body: formData
      });

      if (!response.ok) {
        throw new Error(`Export failed: ${response.statusText}`);
      }

      const exportData = await response.json();
      onExportStart(exportData.id);
      
    } catch (error) {
      console.error('Export error:', error);
      alert(`Failed to start ${options.format.toUpperCase()} export. Please try again.`);
    } finally {
      setIsExporting(false);
    }
  };

  return (
    <>
      {/* Backdrop */}
      <div className="fixed inset-0 bg-black bg-opacity-50 z-50" onClick={onClose} />
      
      {/* Modal */}
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
          {/* Header */}
          <div className="flex items-center justify-between p-6 border-b border-slate-200">
            <div>
              <h2 className="text-lg font-semibold text-slate-900">Export Options</h2>
              <p className="text-sm text-slate-600 mt-1">
                Customize your export of "{workTitle}"
              </p>
            </div>
            <button
              onClick={onClose}
              className="text-slate-400 hover:text-slate-600"
            >
              <XMarkIcon className="w-6 h-6" />
            </button>
          </div>

          {/* Content */}
          <div className="p-6 space-y-6">
            {/* Format Selection */}
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-3">
                Export Format
              </label>
              <div className="grid grid-cols-3 gap-3">
                {(['epub', 'mobi', 'pdf'] as const).map((format) => (
                  <button
                    key={format}
                    onClick={() => handleOptionChange('format', format)}
                    className={`flex flex-col items-center p-4 border-2 rounded-lg transition-colors ${
                      options.format === format
                        ? 'border-orange-500 bg-orange-50 text-orange-700'
                        : 'border-slate-200 hover:border-slate-300 text-slate-700'
                    }`}
                  >
                    <DocumentArrowDownIcon className="w-6 h-6 mb-2" />
                    <span className="font-medium uppercase">{format}</span>
                    <span className="text-xs text-slate-500 mt-1">
                      {format === 'epub' && 'E-readers, iBooks'}
                      {format === 'mobi' && 'Kindle devices'}
                      {format === 'pdf' && 'Print, universal'}
                    </span>
                  </button>
                ))}
              </div>
            </div>

            {/* Content Options */}
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-3">
                Include Content
              </label>
              <div className="space-y-3">
                {[
                  { key: 'include_metadata', label: 'Metadata (title, author, summary, tags)', desc: 'Work information and details' },
                  { key: 'include_tags', label: 'Tags and Warnings', desc: 'Content tags, relationships, characters' },
                  { key: 'include_comments', label: 'Comments', desc: 'Reader comments and kudos' },
                  { key: 'include_images', label: 'Images', desc: 'Embedded images and artwork' },
                  { key: 'chapter_breaks', label: 'Chapter Breaks', desc: 'Clear chapter separation' },
                ].map(({ key, label, desc }) => (
                  <div key={key} className="flex items-start space-x-3">
                    <input
                      type="checkbox"
                      id={key}
                      checked={options[key as keyof ExportOptions] as boolean}
                      onChange={(e) => handleOptionChange(key as keyof ExportOptions, e.target.checked)}
                      className="mt-1 w-4 h-4 text-orange-600 border-slate-300 rounded focus:ring-orange-500"
                    />
                    <div className="flex-1">
                      <label htmlFor={key} className="text-sm font-medium text-slate-700 cursor-pointer">
                        {label}
                      </label>
                      <p className="text-xs text-slate-500 mt-0.5">{desc}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Typography Options (EPUB/MOBI only) */}
            {(options.format === 'epub' || options.format === 'mobi') && (
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-3">
                  Typography
                </label>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-medium text-slate-600 mb-1">
                      Font Family
                    </label>
                    <select
                      value={options.font_family}
                      onChange={(e) => handleOptionChange('font_family', e.target.value)}
                      className="w-full px-3 py-2 border border-slate-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-orange-500"
                    >
                      <option value="serif">Serif (Georgia, Times)</option>
                      <option value="sans-serif">Sans-serif (Helvetica, Arial)</option>
                      <option value="monospace">Monospace (Courier)</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-slate-600 mb-1">
                      Font Size
                    </label>
                    <select
                      value={options.font_size}
                      onChange={(e) => handleOptionChange('font_size', e.target.value)}
                      className="w-full px-3 py-2 border border-slate-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-orange-500"
                    >
                      <option value="10">10pt (Small)</option>
                      <option value="12">12pt (Normal)</option>
                      <option value="14">14pt (Large)</option>
                      <option value="16">16pt (Extra Large)</option>
                    </select>
                  </div>
                </div>
              </div>
            )}

            {/* Cover Image Options */}
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-3">
                Cover Options
              </label>
              <div className="border-2 border-dashed border-slate-300 rounded-lg p-6 text-center">
                {coverImagePreview ? (
                  <div className="space-y-3">
                    <img 
                      src={coverImagePreview} 
                      alt="Cover preview" 
                      className="w-20 h-28 object-cover mx-auto rounded border"
                    />
                    <p className="text-sm text-slate-600">{selectedCoverImage?.name}</p>
                    <div className="flex justify-center space-x-2">
                      <label className="px-3 py-1 text-xs border border-slate-300 rounded cursor-pointer hover:bg-slate-50">
                        Replace
                        <input
                          type="file"
                          accept="image/*"
                          onChange={handleCoverImageChange}
                          className="hidden"
                        />
                      </label>
                      <button
                        onClick={clearCoverImage}
                        className="px-3 py-1 text-xs text-red-600 border border-red-300 rounded hover:bg-red-50"
                      >
                        Remove
                      </button>
                    </div>
                  </div>
                ) : (
                  <>
                    <PhotoIcon className="w-8 h-8 text-slate-400 mx-auto mb-2" />
                    <p className="text-sm text-slate-600 mb-2">Custom Cover Image</p>
                    <p className="text-xs text-slate-500 mb-3">
                      Upload a custom cover or use the default generated cover
                    </p>
                    <label className="inline-block px-4 py-2 text-sm border border-slate-300 rounded-md hover:bg-slate-50 transition-colors cursor-pointer">
                      Choose Image
                      <input
                        type="file"
                        accept="image/*"
                        onChange={handleCoverImageChange}
                        className="hidden"
                      />
                    </label>
                  </>
                )}
              </div>
            </div>

            {/* Advanced Options */}
            <details className="border border-slate-200 rounded-lg">
              <summary className="px-4 py-3 cursor-pointer text-sm font-medium text-slate-700 hover:bg-slate-50 flex items-center">
                <Cog6ToothIcon className="w-4 h-4 mr-2" />
                Advanced Options
              </summary>
              <div className="px-4 pb-4">
                <div className="mt-3">
                  <label className="block text-xs font-medium text-slate-600 mb-2">
                    Custom CSS (EPUB only)
                  </label>
                  <textarea
                    value={options.custom_css || ''}
                    onChange={(e) => handleOptionChange('custom_css', e.target.value)}
                    placeholder="/* Custom styles for your export */&#10;body { line-height: 1.6; }&#10;.chapter { margin-bottom: 2em; }"
                    rows={4}
                    className="w-full px-3 py-2 border border-slate-300 rounded-md text-sm font-mono focus:outline-none focus:ring-2 focus:ring-orange-500"
                  />
                  <p className="text-xs text-slate-500 mt-1">
                    Add custom CSS to style your EPUB export
                  </p>
                </div>
              </div>
            </details>
          </div>

          {/* Footer */}
          <div className="flex items-center justify-between p-6 border-t border-slate-200 bg-slate-50">
            <div className="text-sm text-slate-600">
              Export will be available for download for 24 hours
            </div>
            <div className="flex space-x-3">
              <button
                onClick={onClose}
                className="px-4 py-2 text-sm font-medium text-slate-700 border border-slate-300 rounded-md hover:bg-slate-50 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleExport}
                disabled={isExporting}
                className="px-4 py-2 text-sm font-medium text-white bg-orange-600 rounded-md hover:bg-orange-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center"
              >
                {isExporting ? (
                  <>
                    <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin mr-2" />
                    Starting Export...
                  </>
                ) : (
                  <>
                    <DocumentArrowDownIcon className="w-4 h-4 mr-2" />
                    Export {options.format.toUpperCase()}
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}