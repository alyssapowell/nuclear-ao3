'use client';

import { useState, useEffect, useRef } from 'react';
import { Eye, EyeOff, FileText, Code, AlignLeft, Edit3, Image, Video, Music, Link } from 'lucide-react';

export type ContentFormat = 'html' | 'markdown' | 'plain' | 'richtext';

// AO3-compatible media hosting sites
export const APPROVED_MEDIA_HOSTS = {
  video: [
    'youtube.com', 'youtu.be', 'vimeo.com', 'archive.org',
    'bilibili.com', 'criticalcommons.org', 'google.com',
    'vidders.net', 'viddertube.com'
  ],
  audio: [
    'soundcloud.com', 'podfic.com'
  ],
  playlists: [
    'spotify.com', '8tracks.com'
  ],
  images: [
    // Any HTTPS site with direct image URLs, excluding known problematic ones
    '!discord.com', '!dreamwidth.org', '!facebook.com', 
    '!drive.google.com', '!imgur.com', '!livejournal.com',
    '!onedrive.live.com', '!photobucket.com', '!tinypic.com', '!tumblr.com'
  ]
};

interface MediaEmbed {
  type: 'image' | 'video' | 'audio' | 'playlist';
  url: string;
  provider?: string;
  embedCode?: string;
  altText?: string;
  title?: string;
}

interface ContentFormatEditorProps {
  value: string;
  onChange: (value: string) => void;
  format: ContentFormat;
  onFormatChange: (format: ContentFormat) => void;
  placeholder?: string;
  rows?: number;
  className?: string;
  showPreview?: boolean;
  label?: string;
  required?: boolean;
}

export default function ContentFormatEditor({
  value,
  onChange,
  format,
  onFormatChange,
  placeholder = 'Enter your content...',
  rows = 10,
  className = '',
  showPreview = true,
  label = 'Content',
  required = false
}: ContentFormatEditorProps) {
  const [showPreviewPane, setShowPreviewPane] = useState(false);
  const [wordCount, setWordCount] = useState(0);
  const [showMediaDialog, setShowMediaDialog] = useState(false);
  const [mediaType, setMediaType] = useState<'image' | 'video' | 'audio' | 'playlist'>('image');
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    const words = value.trim().split(/\s+/).filter(word => word.length > 0);
    setWordCount(words.length);
  }, [value]);

  const formatIcons = {
    html: <Code className="w-4 h-4" />,
    markdown: <FileText className="w-4 h-4" />,
    plain: <AlignLeft className="w-4 h-4" />,
    richtext: <Edit3 className="w-4 h-4" />
  };

  const formatLabels = {
    html: 'HTML',
    markdown: 'Markdown',
    plain: 'Plain Text',
    richtext: 'Rich Text'
  };

  const formatDescriptions = {
    html: 'Full HTML markup support',
    markdown: 'Markdown syntax with automatic formatting',
    plain: 'Simple plain text',
    richtext: 'Rich text with basic formatting'
  };

  const renderPreview = () => {
    switch (format) {
      case 'html':
        return <div dangerouslySetInnerHTML={{ __html: value }} />;
      
      case 'markdown':
        // Basic markdown preview (in production, use a proper markdown parser)
        let processed = value
          .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
          .replace(/\*(.*?)\*/g, '<em>$1</em>')
          .replace(/^### (.*$)/gm, '<h3>$1</h3>')
          .replace(/^## (.*$)/gm, '<h2>$1</h2>')
          .replace(/^# (.*$)/gm, '<h1>$1</h1>')
          .replace(/\n\n/g, '</p><p>')
          .replace(/\n/g, '<br>');
        
        if (processed && !processed.startsWith('<h')) {
          processed = '<p>' + processed + '</p>';
        }
        
        return <div dangerouslySetInnerHTML={{ __html: processed }} />;
      
      case 'plain':
        return <div className="whitespace-pre-wrap">{value}</div>;
      
      case 'richtext':
        // For now, treat richtext like HTML
        return <div dangerouslySetInnerHTML={{ __html: value }} />;
      
      default:
        return <div className="whitespace-pre-wrap">{value}</div>;
    }
  };

  const insertMarkdown = (before: string, after: string = '') => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const selectedText = value.substring(start, end);
    const newText = value.substring(0, start) + before + selectedText + after + value.substring(end);
    
    onChange(newText);
    
    // Reset cursor position after state update
    setTimeout(() => {
      textarea.focus();
      textarea.setSelectionRange(start + before.length, end + before.length);
    }, 0);
  };

  const insertMedia = (mediaEmbed: MediaEmbed) => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    let embedCode = '';

    if (mediaEmbed.embedCode) {
      // Use provided embed code (for video/audio platforms)
      embedCode = mediaEmbed.embedCode;
    } else if (mediaEmbed.type === 'image') {
      // Generate image HTML
      embedCode = `<img src="${mediaEmbed.url}" alt="${mediaEmbed.altText || ''}" />`;
    } else {
      // Generic link fallback
      embedCode = `<a href="${mediaEmbed.url}" target="_blank" rel="noopener noreferrer">${mediaEmbed.title || mediaEmbed.url}</a>`;
    }

    const newText = value.substring(0, start) + embedCode + value.substring(end);
    onChange(newText);
    
    setTimeout(() => {
      textarea.focus();
      textarea.setSelectionRange(start + embedCode.length, start + embedCode.length);
    }, 0);
  };

  const validateMediaUrl = (url: string, type: 'image' | 'video' | 'audio' | 'playlist'): boolean => {
    try {
      const urlObj = new URL(url);
      
      // Must be HTTPS for security
      if (urlObj.protocol !== 'https:') return false;
      
      const hostname = urlObj.hostname.toLowerCase();
      
      // Check against approved hosts
      const hostKey = type === 'playlist' ? 'playlists' : type === 'image' ? 'images' : type;
      const approvedHosts = APPROVED_MEDIA_HOSTS[hostKey as keyof typeof APPROVED_MEDIA_HOSTS];
      
      if (type === 'image') {
        // For images, check against exclusion list
        const excludedHosts = APPROVED_MEDIA_HOSTS.images
          .filter(host => host.startsWith('!'))
          .map(host => host.substring(1));
        
        return !excludedHosts.some(excluded => hostname.includes(excluded));
      } else {
        // For other media, must be in approved list
        return approvedHosts.some(approved => hostname.includes(approved));
      }
    } catch {
      return false;
    }
  };

  const editorToolbar = (format === 'markdown' || format === 'html') && (
    <div className="border-b border-slate-200 p-2 flex gap-2 flex-wrap">
      {format === 'markdown' && (
        <>
          <button
            type="button"
            onClick={() => insertMarkdown('**', '**')}
            className="px-2 py-1 text-sm bg-slate-100 hover:bg-slate-200 rounded"
            title="Bold"
          >
            <strong>B</strong>
          </button>
          <button
            type="button"
            onClick={() => insertMarkdown('*', '*')}
            className="px-2 py-1 text-sm bg-slate-100 hover:bg-slate-200 rounded italic"
            title="Italic"
          >
            I
          </button>
          <button
            type="button"
            onClick={() => insertMarkdown('### ')}
            className="px-2 py-1 text-sm bg-slate-100 hover:bg-slate-200 rounded"
            title="Heading"
          >
            H3
          </button>
          <button
            type="button"
            onClick={() => insertMarkdown('\n- ')}
            className="px-2 py-1 text-sm bg-slate-100 hover:bg-slate-200 rounded"
            title="List"
          >
            •
          </button>
          <div className="border-l border-slate-300 mx-2"></div>
        </>
      )}
      
      {/* Media insertion buttons */}
      <button
        type="button"
        onClick={() => { setMediaType('image'); setShowMediaDialog(true); }}
        className="px-2 py-1 text-sm bg-slate-100 hover:bg-slate-200 rounded flex items-center gap-1"
        title="Insert Image"
      >
        <Image className="w-3 h-3" />
        Image
      </button>
      <button
        type="button"
        onClick={() => { setMediaType('video'); setShowMediaDialog(true); }}
        className="px-2 py-1 text-sm bg-slate-100 hover:bg-slate-200 rounded flex items-center gap-1"
        title="Insert Video"
      >
        <Video className="w-3 h-3" />
        Video
      </button>
      <button
        type="button"
        onClick={() => { setMediaType('audio'); setShowMediaDialog(true); }}
        className="px-2 py-1 text-sm bg-slate-100 hover:bg-slate-200 rounded flex items-center gap-1"
        title="Insert Audio"
      >
        <Music className="w-3 h-3" />
        Audio
      </button>
    </div>
  );

  return (
    <div className={`space-y-4 ${className}`}>
      {/* Format Selection */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="block text-sm font-medium text-slate-700">
            {label}
            {required && <span className="text-red-500 ml-1">*</span>}
          </label>
          
          <div className="flex items-center gap-2">
            <span className="text-xs text-slate-500">Format:</span>
            <div className="flex rounded-lg border border-slate-300 overflow-hidden">
              {Object.entries(formatLabels).map(([formatKey, formatLabel]) => (
                <button
                  key={formatKey}
                  type="button"
                  onClick={() => onFormatChange(formatKey as ContentFormat)}
                  className={`px-3 py-1 text-xs flex items-center gap-1 transition-colors ${
                    format === formatKey
                      ? 'bg-orange-500 text-white'
                      : 'bg-white text-slate-700 hover:bg-slate-50'
                  }`}
                  title={formatDescriptions[formatKey as ContentFormat]}
                >
                  {formatIcons[formatKey as ContentFormat]}
                  {formatLabel}
                </button>
              ))}
            </div>
          </div>
        </div>

        <p className="text-xs text-slate-500 mb-3">
          {formatDescriptions[format]}
        </p>
      </div>

      {/* Editor/Preview Layout */}
      <div className="border border-slate-300 rounded-lg overflow-hidden">
        {/* Preview Toggle */}
        {showPreview && (
          <div className="border-b border-slate-200 p-2 bg-slate-50 flex items-center justify-between">
            <div className="text-sm text-slate-600">
              {wordCount} words
            </div>
            <button
              type="button"
              onClick={() => setShowPreviewPane(!showPreviewPane)}
              className="flex items-center gap-1 px-2 py-1 text-sm text-slate-600 hover:text-slate-800 transition-colors"
            >
              {showPreviewPane ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              {showPreviewPane ? 'Hide Preview' : 'Show Preview'}
            </button>
          </div>
        )}

        {editorToolbar}

        <div className={`${showPreviewPane ? 'grid grid-cols-2' : ''}`}>
          {/* Editor */}
          <div className={showPreviewPane ? 'border-r border-slate-200' : ''}>
            <textarea
              ref={textareaRef}
              value={value}
              onChange={(e) => onChange(e.target.value)}
              placeholder={placeholder}
              rows={rows}
              required={required}
              className={`w-full px-4 py-3 border-0 resize-none focus:outline-none focus:ring-0 ${
                format === 'html' ? 'font-mono text-sm' : ''
              }`}
              style={{ minHeight: `${rows * 1.5}rem` }}
            />
          </div>

          {/* Preview Pane */}
          {showPreviewPane && (
            <div className="p-4 bg-slate-50 overflow-auto prose prose-sm max-w-none">
              <div className="text-xs text-slate-500 mb-3 font-semibold uppercase tracking-wide">
                Preview
              </div>
              {value ? (
                <div className="text-slate-800">
                  {renderPreview()}
                </div>
              ) : (
                <div className="text-slate-400 italic">
                  Preview will appear here as you type...
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Format-specific help */}
      {format === 'markdown' && (
        <div className="text-xs text-slate-500 bg-blue-50 border border-blue-200 rounded p-2">
          <strong>Markdown Tips:</strong> **bold**, *italic*, ### headers, - lists
        </div>
      )}
      
      {format === 'html' && (
        <div className="text-xs text-slate-500 bg-yellow-50 border border-yellow-200 rounded p-2">
          <strong>HTML:</strong> Full HTML markup is supported. Be careful with script tags and styling.
        </div>
      )}

      {/* Media Insertion Dialog */}
      {showMediaDialog && <MediaInsertDialog />}
    </div>
  );

  function MediaInsertDialog() {
    const [mediaUrl, setMediaUrl] = useState('');
    const [embedCode, setEmbedCode] = useState('');
    const [altText, setAltText] = useState('');
    const [title, setTitle] = useState('');
    const [isValid, setIsValid] = useState(false);

    const handleUrlChange = (url: string) => {
      setMediaUrl(url);
      setIsValid(validateMediaUrl(url, mediaType));
    };

    const handleInsert = () => {
      if (!isValid && !embedCode) return;

      const media: MediaEmbed = {
        type: mediaType,
        url: mediaUrl,
        embedCode: embedCode || undefined,
        altText: altText || undefined,
        title: title || undefined
      };

      insertMedia(media);
      setShowMediaDialog(false);
      setMediaUrl('');
      setEmbedCode('');
      setAltText('');
      setTitle('');
    };

    const getApprovedSites = () => {
      switch (mediaType) {
        case 'video':
          return APPROVED_MEDIA_HOSTS.video.join(', ');
        case 'audio':
          return APPROVED_MEDIA_HOSTS.audio.join(', ');
        case 'playlist':
          return APPROVED_MEDIA_HOSTS.playlists.join(', ');
        case 'image':
          return 'Any HTTPS image URL (avoiding Discord, Facebook, Imgur, etc.)';
        default:
          return '';
      }
    };

    return (
      <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
          <h3 className="text-lg font-semibold mb-4">Insert {mediaType}</h3>
          
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                {mediaType === 'image' ? 'Image URL' : 'Media URL or Embed Code'}
              </label>
              
              {mediaType !== 'image' ? (
                <textarea
                  value={embedCode || mediaUrl}
                  onChange={(e) => {
                    const value = e.target.value;
                    if (value.includes('<') || value.includes('iframe')) {
                      setEmbedCode(value);
                      setMediaUrl('');
                    } else {
                      setEmbedCode('');
                      handleUrlChange(value);
                    }
                  }}
                  placeholder={`Paste ${mediaType} URL or embed code from approved sites`}
                  className="w-full px-3 py-2 border border-slate-300 rounded-md focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                  rows={3}
                />
              ) : (
                <input
                  type="url"
                  value={mediaUrl}
                  onChange={(e) => handleUrlChange(e.target.value)}
                  placeholder="https://example.com/image.jpg"
                  className="w-full px-3 py-2 border border-slate-300 rounded-md focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                />
              )}
              
              <p className="text-xs text-slate-500 mt-1">
                <strong>Approved sites:</strong> {getApprovedSites()}
              </p>
              
              {mediaUrl && !isValid && !embedCode && (
                <p className="text-xs text-red-600 mt-1">
                  URL must be HTTPS and from an approved hosting site
                </p>
              )}
            </div>

            {mediaType === 'image' && (
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Alt Text (for accessibility)
                </label>
                <input
                  type="text"
                  value={altText}
                  onChange={(e) => setAltText(e.target.value)}
                  placeholder="Describe the image for screen readers"
                  className="w-full px-3 py-2 border border-slate-300 rounded-md focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                />
              </div>
            )}

            {mediaType !== 'image' && (
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Title (optional)
                </label>
                <input
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder={`${mediaType} title`}
                  className="w-full px-3 py-2 border border-slate-300 rounded-md focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                />
              </div>
            )}
          </div>

          <div className="flex gap-3 mt-6">
            <button
              onClick={handleInsert}
              disabled={!isValid && !embedCode}
              className="flex-1 bg-orange-500 text-white px-4 py-2 rounded-md hover:bg-orange-600 disabled:bg-slate-300 disabled:cursor-not-allowed"
            >
              Insert {mediaType}
            </button>
            <button
              onClick={() => setShowMediaDialog(false)}
              className="flex-1 bg-slate-200 text-slate-800 px-4 py-2 rounded-md hover:bg-slate-300"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
    );
  }
}