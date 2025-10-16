'use client';

import { useState } from 'react';
import TagAutocomplete from '@/components/TagAutocomplete';
import RichTextEditor from '@/components/RichTextEditor';

// Enhanced tag input component with autocomplete - copied from works/new
interface EnhancedTagInputProps {
  tags: string[];
  onChange: (tags: string[]) => void;
  placeholder: string;
  tagType: 'fandom' | 'character' | 'relationship' | 'freeform';
  required?: boolean;
}

function EnhancedTagInput({ tags, onChange, placeholder, tagType, required }: EnhancedTagInputProps) {
  const [inputValue, setInputValue] = useState('');

  const handleTagSelect = (tag: any) => {
    const tagName = tag.name || tag;
    if (tagName && !tags.includes(tagName)) {
      onChange([...tags, tagName]);
    }
    setInputValue('');
  };

  const handleInputChange = (value: string) => {
    setInputValue(value);
    
    // Auto-add tag on comma
    if (value.endsWith(',')) {
      const newTag = value.slice(0, -1).trim();
      if (newTag && !tags.includes(newTag)) {
        onChange([...tags, newTag]);
        setInputValue('');
      }
    }
  };

  const removeTag = (indexToRemove: number) => {
    onChange(tags.filter((_, index) => index !== indexToRemove));
  };

  return (
    <div className="space-y-2">
      <TagAutocomplete
        value={inputValue}
        onChange={handleInputChange}
        onTagSelect={handleTagSelect}
        placeholder={placeholder}
        required={required}
        tagType={tagType}
        className="w-full"
      />
      
      {/* Display selected tags */}
      {tags.length > 0 && (
        <div className="flex flex-wrap gap-2 mt-2">
          {tags.map((tag, index) => (
            <span
              key={index}
              className="inline-flex items-center px-3 py-1 rounded-full text-sm bg-blue-100 text-blue-800 border border-blue-200"
            >
              {tag}
              <button
                type="button"
                onClick={() => removeTag(index)}
                className="ml-2 text-blue-600 hover:text-blue-800 focus:outline-none"
                aria-label={`Remove ${tag} tag`}
              >
                ×
              </button>
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

export default function TestComponents() {
  // State for testing tag inputs
  const [fandoms, setFandoms] = useState<string[]>([]);
  const [characters, setCharacters] = useState<string[]>([]);
  const [relationships, setRelationships] = useState<string[]>([]);
  const [freeformTags, setFreeformTags] = useState<string[]>([]);
  
  // State for testing rich text editor
  const [content, setContent] = useState('<p>Start typing to test the rich text editor...</p>');

  return (
    <div className="max-w-4xl mx-auto p-6 space-y-8">
      <div className="bg-white p-6 rounded-lg shadow">
        <h1 className="text-3xl font-bold text-gray-900 mb-6">Enhanced Form Components Test</h1>
        
        <div className="bg-yellow-50 border border-yellow-200 rounded-md p-4 mb-6">
          <p className="text-yellow-800 text-sm">
            <strong>Test Page:</strong> This page is for testing our enhanced form components without authentication.
            Test the tag autocomplete and rich text editor functionality.
          </p>
        </div>
      </div>

      {/* Tag Input Tests */}
      <div className="bg-white p-6 rounded-lg shadow">
        <h2 className="text-2xl font-semibold text-gray-900 mb-4">Enhanced Tag Inputs</h2>
        
        <div className="space-y-6">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Fandoms *
            </label>
            <EnhancedTagInput
              tags={fandoms}
              onChange={setFandoms}
              placeholder="Start typing fandom names (e.g., Marvel, Harry Potter)..."
              tagType="fandom"
              required
            />
            <p className="mt-1 text-xs text-gray-500">
              Try typing: "Marvel", "Harry Potter", "Supernatural". Use comma or Enter to add tags.
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Characters
            </label>
            <EnhancedTagInput
              tags={characters}
              onChange={setCharacters}
              placeholder="Character names..."
              tagType="character"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Relationships
            </label>
            <EnhancedTagInput
              tags={relationships}
              onChange={setRelationships}
              placeholder="Character relationships..."
              tagType="relationship"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Additional Tags
            </label>
            <EnhancedTagInput
              tags={freeformTags}
              onChange={setFreeformTags}
              placeholder="Additional tags..."
              tagType="freeform"
            />
          </div>
        </div>

        {/* Display current tag state */}
        <div className="mt-6 p-4 bg-gray-50 rounded-md">
          <h3 className="text-sm font-medium text-gray-700 mb-2">Current Tags State:</h3>
          <pre className="text-xs text-gray-600">
{JSON.stringify({
  fandoms,
  characters,
  relationships,
  freeformTags
}, null, 2)}
          </pre>
        </div>
      </div>

      {/* Rich Text Editor Test */}
      <div className="bg-white p-6 rounded-lg shadow">
        <h2 className="text-2xl font-semibold text-gray-900 mb-4">Rich Text Editor</h2>
        
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Content
          </label>
          <RichTextEditor
            content={content}
            onChange={setContent}
            placeholder="Start writing your story..."
            className="min-h-[300px]"
          />
          
          <p className="mt-2 text-xs text-gray-500">
            Try the formatting buttons, add links, create lists, and use headings.
          </p>
        </div>

        {/* Display current content */}
        <div className="mt-6 p-4 bg-gray-50 rounded-md">
          <h3 className="text-sm font-medium text-gray-700 mb-2">Generated HTML:</h3>
          <pre className="text-xs text-gray-600 whitespace-pre-wrap">
            {content}
          </pre>
        </div>
      </div>

      {/* Test Results Summary */}
      <div className="bg-green-50 p-6 rounded-lg">
        <h2 className="text-lg font-semibold text-green-900 mb-4">Test Instructions</h2>
        <div className="space-y-2 text-sm text-green-800">
          <div>✅ <strong>Tag Autocomplete:</strong> Type in tag fields and verify suggestions appear</div>
          <div>✅ <strong>Auto-conversion:</strong> Type text and press comma/Enter to convert to tags</div>
          <div>✅ <strong>Tag Removal:</strong> Click × button to remove tags</div>
          <div>✅ <strong>Rich Text:</strong> Use toolbar to format text, add links, create lists</div>
          <div>✅ <strong>Link Modal:</strong> Click link button and add URLs</div>
          <div>✅ <strong>Keyboard Navigation:</strong> Test accessibility with Tab/Enter</div>
        </div>
      </div>
    </div>
  );
}