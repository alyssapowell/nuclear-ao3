'use client';

import { useMemo } from 'react';
import { type ContentFormat } from './ContentFormatEditor';

interface ContentRendererProps {
  content: string;
  format: ContentFormat;
  className?: string;
  maxLength?: number;
  showReadMore?: boolean;
}

export default function ContentRenderer({
  content,
  format,
  className = '',
  maxLength,
  showReadMore = false
}: ContentRendererProps) {
  const processedContent = useMemo(() => {
    if (!content) return '';

    // Truncate if maxLength is specified
    let processedText = content;
    if (maxLength && content.length > maxLength) {
      processedText = content.substring(0, maxLength);
      
      // Try to break at word boundary
      const lastSpace = processedText.lastIndexOf(' ');
      if (lastSpace > maxLength * 0.8) {
        processedText = processedText.substring(0, lastSpace);
      }
      
      if (showReadMore) {
        processedText += '...';
      }
    }

    switch (format) {
      case 'html':
        return processedText;
      
      case 'markdown':
        // Basic markdown to HTML conversion
        return processedText
          .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
          .replace(/\*(.*?)\*/g, '<em>$1</em>')
          .replace(/^### (.*$)/gm, '<h3>$1</h3>')
          .replace(/^## (.*$)/gm, '<h2>$1</h2>')
          .replace(/^# (.*$)/gm, '<h1>$1</h1>')
          .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>')
          .replace(/`([^`]+)`/g, '<code>$1</code>')
          .replace(/\n\n/g, '</p><p>')
          .replace(/\n/g, '<br>');
      
      case 'plain':
        // Escape HTML and preserve line breaks
        return processedText
          .replace(/&/g, '&amp;')
          .replace(/</g, '&lt;')
          .replace(/>/g, '&gt;')
          .replace(/\n/g, '<br>');
      
      case 'richtext':
        // Rich text is already processed HTML
        return processedText;
      
      default:
        return processedText;
    }
  }, [content, format, maxLength, showReadMore]);

  const wrapContent = (htmlContent: string) => {
    if (format === 'markdown' && htmlContent && !htmlContent.startsWith('<h')) {
      return `<p>${htmlContent}</p>`;
    }
    return htmlContent;
  };

  const getBaseClasses = () => {
    const baseClasses = 'prose prose-sm max-w-none';
    
    switch (format) {
      case 'html':
      case 'richtext':
        return `${baseClasses} prose-orange`;
      case 'markdown':
        return `${baseClasses} prose-blue`;
      case 'plain':
        return 'whitespace-pre-wrap text-slate-700';
      default:
        return baseClasses;
    }
  };

  if (!content) {
    return null;
  }

  if (format === 'plain') {
    return (
      <div className={`${getBaseClasses()} ${className}`}>
        <div dangerouslySetInnerHTML={{ __html: processedContent }} />
      </div>
    );
  }

  return (
    <div className={`${getBaseClasses()} ${className}`}>
      <div 
        dangerouslySetInnerHTML={{ 
          __html: wrapContent(processedContent) 
        }} 
      />
    </div>
  );
}

// Utility function to get content preview for cards/summaries
export function getContentPreview(
  content: string, 
  format: ContentFormat, 
  maxLength: number = 200
): string {
  if (!content) return '';

  let preview = content;
  
  // Strip markdown/HTML formatting for preview
  switch (format) {
    case 'html':
    case 'richtext':
      // Basic HTML tag removal
      preview = content.replace(/<[^>]*>/g, '');
      break;
    case 'markdown':
      // Basic markdown removal
      preview = content
        .replace(/\*\*(.*?)\*\*/g, '$1')
        .replace(/\*(.*?)\*/g, '$1')
        .replace(/^#+\s+/gm, '')
        .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
        .replace(/`([^`]+)`/g, '$1');
      break;
    case 'plain':
    default:
      // Plain text is already clean
      break;
  }

  // Truncate and clean up
  if (preview.length > maxLength) {
    preview = preview.substring(0, maxLength);
    const lastSpace = preview.lastIndexOf(' ');
    if (lastSpace > maxLength * 0.8) {
      preview = preview.substring(0, lastSpace);
    }
    preview += '...';
  }

  return preview.replace(/\s+/g, ' ').trim();
}