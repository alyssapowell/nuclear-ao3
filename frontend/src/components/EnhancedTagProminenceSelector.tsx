'use client';

import { useState } from 'react';
import { Switch } from '@headlessui/react';
import TagInput from './TagInput';
import { TagSuggestion } from '@/lib/api';

interface TagProminence {
  tagName: string;
  tagType: string;
  prominence: 'primary' | 'secondary' | 'micro';
  autoSuggested?: boolean;
  canonical?: boolean;
  useCount?: number;
}

interface EnhancedTagProminenceSelectorProps {
  tags: TagProminence[];
  onTagsChange: (tags: TagProminence[]) => void;
  fandomId?: string;
  className?: string;
}

export default function EnhancedTagProminenceSelector({ 
  tags, 
  onTagsChange, 
  fandomId,
  className = '' 
}: EnhancedTagProminenceSelectorProps) {
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [currentInput, setCurrentInput] = useState('');
  const [currentTagType, setCurrentTagType] = useState<'relationship' | 'character' | 'freeform'>('relationship');

  // AI inference for tag prominence
  const inferProminence = (tagName: string, tagType: string, suggestion?: TagSuggestion): 'primary' | 'secondary' | 'micro' => {
    const lowerTag = tagName.toLowerCase();
    
    // Background/side/past tags should be micro
    if (lowerTag.includes('background') || lowerTag.includes('side') || 
        lowerTag.includes('minor') || lowerTag.includes('past') ||
        lowerTag.includes('mentioned') || lowerTag.includes('cameo')) {
      return 'micro';
    }
    
    // For relationships
    if (tagType === 'relationship') {
      // If we already have many relationships, make new ones secondary by default
      const existingRelationships = tags.filter(t => t.tagType === 'relationship');
      if (existingRelationships.length >= 2) {
        return 'secondary';
      }
      return 'primary';
    }
    
    // For freeform tags
    if (tagType === 'freeform') {
      // Major themes should be primary
      if (lowerTag.includes('slow burn') || lowerTag.includes('enemies to lovers') ||
          lowerTag.includes('hurt/comfort') || lowerTag.includes('angst') ||
          lowerTag.includes('fluff') || lowerTag.includes('smut')) {
        return 'primary';
      }
      return 'secondary';
    }
    
    // Characters default to secondary
    return 'secondary';
  };

  const addTag = (tagName: string, suggestion?: TagSuggestion) => {
    // Don't add if already exists
    if (tags.some(t => t.tagName === tagName)) {
      return;
    }

    const prominence = inferProminence(tagName, currentTagType, suggestion);
    
    const newTag: TagProminence = {
      tagName,
      tagType: currentTagType,
      prominence,
      autoSuggested: true,
      canonical: suggestion?.canonical,
      useCount: suggestion?.use_count,
    };

    onTagsChange([...tags, newTag]);
  };

  const updateTagProminence = (tagName: string, prominence: 'primary' | 'secondary' | 'micro') => {
    const updatedTags = tags.map(tag => 
      tag.tagName === tagName 
        ? { ...tag, prominence, autoSuggested: false }
        : tag
    );
    onTagsChange(updatedTags);
  };

  const removeTag = (tagName: string) => {
    const updatedTags = tags.filter(tag => tag.tagName !== tagName);
    onTagsChange(updatedTags);
  };

  const getTagsByProminence = (prominence: 'primary' | 'secondary' | 'micro') => {
    return tags.filter(t => t.prominence === prominence);
  };

  const renderTagBadge = (tag: TagProminence, className: string = '') => (
    <div key={tag.tagName} className={`tag-badge ${className}`}>
      <span className="tag-name">{tag.tagName}</span>
      
      {/* Show canonical/popularity indicators */}
      <div className="tag-meta">
        {tag.canonical && (
          <span className="canonical-indicator" title="Canonical tag">
            ✓
          </span>
        )}
        {tag.useCount && tag.useCount > 100 && (
          <span className="popular-indicator" title={`${tag.useCount} uses`}>
            🔥
          </span>
        )}
        {tag.autoSuggested && (
          <span className="auto-suggested-indicator" title="AI suggested prominence">
            🤖
          </span>
        )}
      </div>

      <div className="tag-actions">
        {/* Prominence controls */}
        <select
          value={tag.prominence}
          onChange={(e) => updateTagProminence(tag.tagName, e.target.value as 'primary' | 'secondary' | 'micro')}
          className="prominence-select"
        >
          <option value="primary">Primary</option>
          <option value="secondary">Secondary</option>
          <option value="micro">Micro</option>
        </select>
        
        <button
          onClick={() => removeTag(tag.tagName)}
          className="remove-button"
          title="Remove tag"
        >
          ×
        </button>
      </div>
    </div>
  );

  const primaryTags = getTagsByProminence('primary');
  const secondaryTags = getTagsByProminence('secondary');
  const microTags = getTagsByProminence('micro');

  return (
    <div className={`enhanced-tag-prominence-selector ${className}`}>
      <style jsx>{`
        .enhanced-tag-prominence-selector {
          space-y: 1.5rem;
        }
        .tag-input-section {
          background: #f8fafc;
          border: 1px solid #e2e8f0;
          border-radius: 0.5rem;
          padding: 1rem;
        }
        .tag-type-selector {
          display: flex;
          gap: 0.5rem;
          margin-bottom: 1rem;
        }
        .tag-type-button {
          padding: 0.5rem 1rem;
          border: 1px solid #e2e8f0;
          border-radius: 0.375rem;
          background: white;
          cursor: pointer;
          transition: all 0.2s;
        }
        .tag-type-button:hover {
          background: #f1f5f9;
        }
        .tag-type-button.active {
          background: #f97316;
          color: white;
          border-color: #f97316;
        }
        .prominence-section {
          border: 1px solid #e2e8f0;
          border-radius: 0.5rem;
          padding: 1rem;
          margin-bottom: 1rem;
        }
        .prominence-section.primary {
          border-color: #22c55e;
          background: #f0fdf4;
        }
        .prominence-section.secondary {
          border-color: #3b82f6;
          background: #eff6ff;
        }
        .prominence-section.micro {
          border-color: #64748b;
          background: #f8fafc;
        }
        .prominence-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          margin-bottom: 0.75rem;
        }
        .prominence-title {
          font-weight: 600;
          font-size: 1.125rem;
          color: #1e293b;
        }
        .prominence-count {
          background: #e2e8f0;
          color: #475569;
          padding: 0.25rem 0.5rem;
          border-radius: 9999px;
          font-size: 0.75rem;
          font-weight: 500;
        }
        .tags-grid {
          display: flex;
          flex-wrap: wrap;
          gap: 0.5rem;
        }
        .tag-badge {
          display: flex;
          align-items: center;
          gap: 0.5rem;
          background: white;
          border: 1px solid #d1d5db;
          border-radius: 0.375rem;
          padding: 0.5rem;
          transition: all 0.2s;
        }
        .tag-badge:hover {
          box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .tag-name {
          font-weight: 500;
          color: #374151;
        }
        .tag-meta {
          display: flex;
          align-items: center;
          gap: 0.25rem;
        }
        .canonical-indicator {
          color: #22c55e;
          font-weight: bold;
        }
        .popular-indicator {
          font-size: 0.75rem;
        }
        .auto-suggested-indicator {
          font-size: 0.75rem;
          opacity: 0.7;
        }
        .tag-actions {
          display: flex;
          align-items: center;
          gap: 0.5rem;
        }
        .prominence-select {
          font-size: 0.75rem;
          padding: 0.25rem;
          border: 1px solid #d1d5db;
          border-radius: 0.25rem;
          background: white;
        }
        .remove-button {
          color: #ef4444;
          background: none;
          border: none;
          font-size: 1.25rem;
          cursor: pointer;
          padding: 0.125rem;
          border-radius: 0.125rem;
          transition: all 0.2s;
        }
        .remove-button:hover {
          background: #fee2e2;
        }
        .empty-state {
          text-align: center;
          color: #64748b;
          font-style: italic;
          padding: 2rem;
        }
      `}</style>

      {/* Tag Input Section */}
      <div className="tag-input-section">
        <h3 className="font-semibold text-slate-900 mb-3">Add Tags</h3>
        
        {/* Tag Type Selector */}
        <div className="tag-type-selector">
          <button
            className={`tag-type-button ${currentTagType === 'relationship' ? 'active' : ''}`}
            onClick={() => setCurrentTagType('relationship')}
          >
            Relationships
          </button>
          <button
            className={`tag-type-button ${currentTagType === 'character' ? 'active' : ''}`}
            onClick={() => setCurrentTagType('character')}
          >
            Characters  
          </button>
          <button
            className={`tag-type-button ${currentTagType === 'freeform' ? 'active' : ''}`}
            onClick={() => setCurrentTagType('freeform')}
          >
            Additional Tags
          </button>
        </div>

        {/* Tag Input with Autocomplete */}
        <TagInput
          value={currentInput}
          onChange={setCurrentInput}
          onTagAdd={addTag}
          tagType={[currentTagType]}
          fandomId={fandomId}
          placeholder={`Type to search for ${currentTagType} tags...`}
        />

        <p className="text-sm text-slate-600 mt-2">
          💡 Try typing &quot;Background&quot; or &quot;Past&quot; before relationship names to automatically set micro prominence
        </p>
      </div>

      {/* Primary Tags */}
      <div className="prominence-section primary">
        <div className="prominence-header">
          <h3 className="prominence-title">Primary Tags</h3>
          <span className="prominence-count">{primaryTags.length}</span>
        </div>
        <p className="text-sm text-slate-600 mb-3">
          The main focus of your work. These elements are central to the story.
        </p>
        {primaryTags.length > 0 ? (
          <div className="tags-grid">
            {primaryTags.map(tag => renderTagBadge(tag))}
          </div>
        ) : (
          <div className="empty-state">
            No primary tags yet. Add the main relationships and themes of your work.
          </div>
        )}
      </div>

      {/* Secondary Tags */}
      <div className="prominence-section secondary">
        <div className="prominence-header">
          <h3 className="prominence-title">Secondary Tags</h3>
          <span className="prominence-count">{secondaryTags.length}</span>
        </div>
        <p className="text-sm text-slate-600 mb-3">
          Important elements that appear throughout but aren&apos;t the main focus.
        </p>
        {secondaryTags.length > 0 ? (
          <div className="tags-grid">
            {secondaryTags.map(tag => renderTagBadge(tag))}
          </div>
        ) : (
          <div className="empty-state">
            No secondary tags yet. Add supporting characters and themes.
          </div>
        )}
      </div>

      {/* Micro Tags */}
      <div className="prominence-section micro">
        <div className="prominence-header">
          <h3 className="prominence-title">Micro Tags</h3>
          <span className="prominence-count">{microTags.length}</span>
        </div>
        <p className="text-sm text-slate-600 mb-3">
          Background elements, brief mentions, or minor appearances.
        </p>
        {microTags.length > 0 ? (
          <div className="tags-grid">
            {microTags.map(tag => renderTagBadge(tag))}
          </div>
        ) : (
          <div className="empty-state">
            No micro tags yet. Add background elements and minor characters.
          </div>
        )}
      </div>

      {/* Advanced Settings */}
      <div className="mt-6">
        <Switch.Group>
          <div className="flex items-center justify-between">
            <Switch.Label className="text-sm font-medium text-slate-700">
              Show advanced tag options
            </Switch.Label>
            <Switch
              checked={showAdvanced}
              onChange={setShowAdvanced}
              className={`${
                showAdvanced ? 'bg-orange-600' : 'bg-slate-200'
              } relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-orange-500 focus:ring-offset-2`}
            >
              <span
                className={`${
                  showAdvanced ? 'translate-x-6' : 'translate-x-1'
                } inline-block h-4 w-4 transform rounded-full bg-white transition-transform`}
              />
            </Switch>
          </div>
        </Switch.Group>

        {showAdvanced && (
          <div className="mt-4 p-4 bg-slate-50 rounded-lg">
            <h4 className="font-medium text-slate-900 mb-2">Tag Statistics</h4>
            <div className="grid grid-cols-3 gap-4 text-sm">
              <div>
                <span className="font-medium">Total tags:</span> {tags.length}
              </div>
              <div>
                <span className="font-medium">AI suggested:</span>{' '}
                {tags.filter(t => t.autoSuggested).length}
              </div>
              <div>
                <span className="font-medium">Canonical:</span>{' '}
                {tags.filter(t => t.canonical).length}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}