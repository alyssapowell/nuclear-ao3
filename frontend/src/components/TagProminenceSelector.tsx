import { useState } from 'react';
import { Switch } from '@headlessui/react';

interface TagProminence {
  tagName: string;
  tagType: string;
  prominence: 'primary' | 'secondary' | 'micro';
  autoSuggested?: boolean;
}

interface TagProminenceSelectorProps {
  tags: TagProminence[];
  onTagsChange: (tags: TagProminence[]) => void;
  className?: string;
}

export default function TagProminenceSelector({ 
  tags, 
  onTagsChange, 
  className = '' 
}: TagProminenceSelectorProps) {
  const [showAdvanced, setShowAdvanced] = useState(false);

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

  const primaryRelationships = tags.filter(t => t.tagType === 'relationship' && t.prominence === 'primary');
  const secondaryTags = tags.filter(t => t.prominence === 'secondary');
  const microTags = tags.filter(t => t.prominence === 'micro');

  return (
    <div className={`tag-prominence-selector ${className}`}>
      {/* Primary Relationships Section */}
      <div className="mb-6">
        <div className="flex justify-between items-center mb-3">
          <h3 className="text-lg font-semibold text-slate-900">
            Primary Relationships
            <span className="text-sm font-normal text-slate-600 ml-2">
              (Main focus of your work)
            </span>
          </h3>
          <span className="text-xs text-slate-500">
            {primaryRelationships.length}/3 max
          </span>
        </div>
        
        <div className="space-y-2 mb-3">
          {primaryRelationships.map((tag) => (
            <div key={tag.tagName} className="flex items-center justify-between bg-orange-50 border border-orange-200 rounded-lg p-3">
              <div className="flex items-center">
                <span className="text-orange-800 font-medium">{tag.tagName}</span>
                {tag.autoSuggested && (
                  <span className="ml-2 text-xs bg-blue-100 text-blue-800 px-2 py-1 rounded">
                    AI suggested
                  </span>
                )}
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => updateTagProminence(tag.tagName, 'secondary')}
                  className="text-xs text-slate-600 hover:text-slate-800"
                  title="Move to secondary"
                >
                  ↓ Secondary
                </button>
                <button
                  onClick={() => removeTag(tag.tagName)}
                  className="text-xs text-red-600 hover:text-red-800"
                  title="Remove tag"
                >
                  ✕
                </button>
              </div>
            </div>
          ))}
        </div>

        {primaryRelationships.length === 0 && (
          <div className="text-center py-4 text-slate-500 bg-slate-50 rounded-lg border-2 border-dashed border-slate-200">
            No primary relationships selected. This will be marked as Gen fic.
          </div>
        )}
      </div>

      {/* Advanced Toggle */}
      <div className="mb-4">
        <Switch.Group>
          <div className="flex items-center">
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
            <Switch.Label className="ml-3 text-sm font-medium text-slate-700">
              Show all tags & prominence levels
            </Switch.Label>
          </div>
        </Switch.Group>
      </div>

      {showAdvanced && (
        <>
          {/* Secondary Tags Section */}
          <div className="mb-6">
            <h3 className="text-lg font-semibold text-slate-900 mb-3">
              Secondary Tags
              <span className="text-sm font-normal text-slate-600 ml-2">
                (Significant but not main focus)
              </span>
            </h3>
            
            <div className="space-y-2">
              {secondaryTags.map((tag) => (
                <div key={tag.tagName} className="flex items-center justify-between bg-blue-50 border border-blue-200 rounded-lg p-3">
                  <div className="flex items-center">
                    <span className="text-blue-800">{tag.tagName}</span>
                    <span className="ml-2 text-xs text-blue-600 bg-blue-100 px-2 py-1 rounded">
                      {tag.tagType}
                    </span>
                    {tag.autoSuggested && (
                      <span className="ml-2 text-xs bg-gray-100 text-gray-700 px-2 py-1 rounded">
                        AI suggested
                      </span>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => updateTagProminence(tag.tagName, 'primary')}
                      className="text-xs text-orange-600 hover:text-orange-800"
                      title="Move to primary"
                      disabled={primaryRelationships.length >= 3 && tag.tagType === 'relationship'}
                    >
                      ↑ Primary
                    </button>
                    <button
                      onClick={() => updateTagProminence(tag.tagName, 'micro')}
                      className="text-xs text-slate-600 hover:text-slate-800"
                      title="Move to micro"
                    >
                      ↓ Micro
                    </button>
                    <button
                      onClick={() => removeTag(tag.tagName)}
                      className="text-xs text-red-600 hover:text-red-800"
                      title="Remove tag"
                    >
                      ✕
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Micro Tags Section */}
          <div className="mb-6">
            <h3 className="text-lg font-semibold text-slate-900 mb-3">
              Background/Micro Tags
              <span className="text-sm font-normal text-slate-600 ml-2">
                (Brief mentions, cameos, drabble collections)
              </span>
            </h3>
            
            <div className="space-y-2">
              {microTags.map((tag) => (
                <div key={tag.tagName} className="flex items-center justify-between bg-gray-50 border border-gray-200 rounded-lg p-2">
                  <div className="flex items-center">
                    <span className="text-gray-700 text-sm">{tag.tagName}</span>
                    <span className="ml-2 text-xs text-gray-500 bg-gray-100 px-2 py-1 rounded">
                      {tag.tagType}
                    </span>
                    {tag.autoSuggested && (
                      <span className="ml-2 text-xs bg-yellow-100 text-yellow-700 px-2 py-1 rounded">
                        AI suggested
                      </span>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => updateTagProminence(tag.tagName, 'secondary')}
                      className="text-xs text-blue-600 hover:text-blue-800"
                      title="Move to secondary"
                    >
                      ↑ Secondary
                    </button>
                    <button
                      onClick={() => removeTag(tag.tagName)}
                      className="text-xs text-red-600 hover:text-red-800"
                      title="Remove tag"
                    >
                      ✕
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </>
      )}

      {/* Tag Guidelines */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 text-sm">
        <h4 className="font-semibold text-blue-900 mb-2">Tagging Guidelines</h4>
        <ul className="text-blue-800 space-y-1">
          <li><strong>Primary:</strong> Main relationships/themes (1-3 max for relationships)</li>
          <li><strong>Secondary:</strong> Important but not central elements</li>
          <li><strong>Micro:</strong> Brief mentions, background elements, drabble collections</li>
          <li><strong>Tip:</strong> Readers can filter to "primary only" to find works truly focused on their interests</li>
        </ul>
      </div>

      {/* Tag Abuse Warning */}
      {tags.length > 20 && (
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 text-sm mt-4">
          <h4 className="font-semibold text-yellow-900 mb-2">🚨 Tag Spam Warning</h4>
          <p className="text-yellow-800">
            You have {tags.length} tags. Consider if all these tags truly apply to your work. 
            Excessive tagging makes it harder for readers to find what they're looking for.
          </p>
        </div>
      )}
    </div>
  );
}