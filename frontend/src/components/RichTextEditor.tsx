'use client';

import { useEditor, EditorContent } from '@tiptap/react';
import StarterKit from '@tiptap/starter-kit';
import Link from '@tiptap/extension-link';
import Placeholder from '@tiptap/extension-placeholder';
import { useState } from 'react';

interface RichTextEditorProps {
  content: string;
  onChange: (content: string) => void;
  placeholder?: string;
  className?: string;
  disabled?: boolean;
}

export default function RichTextEditor({ 
  content, 
  onChange, 
  placeholder = "Start writing...",
  className = "",
  disabled = false
}: RichTextEditorProps) {
  const [isLinkModalOpen, setIsLinkModalOpen] = useState(false);
  const [linkUrl, setLinkUrl] = useState('');

  const editor = useEditor({
    immediatelyRender: false, // Fix SSR hydration error
    extensions: [
      StarterKit.configure({
        // Only allow safe formatting
        heading: {
          levels: [2, 3] // Only h2 and h3 for chapter subheadings
        },
        // Disable code blocks and code marks for security
        code: false,
        codeBlock: false,
      }),
      Link.configure({
        openOnClick: false,
        HTMLAttributes: {
          // Add security attributes
          target: '_blank',
          rel: 'noopener noreferrer nofollow',
        },
      }),
      Placeholder.configure({
        placeholder,
      }),
    ],
    content,
    onUpdate: ({ editor }) => {
      onChange(editor.getHTML());
    },
    editable: !disabled,
  });

  if (!editor) {
    return null;
  }

  const addLink = () => {
    if (linkUrl) {
      // If text is selected, make it a link
      if (editor.state.selection.empty) {
        editor.chain().focus().insertContent(`<a href="${linkUrl}" target="_blank" rel="noopener noreferrer">${linkUrl}</a>`).run();
      } else {
        editor.chain().focus().setLink({ href: linkUrl }).run();
      }
    }
    setLinkUrl('');
    setIsLinkModalOpen(false);
  };

  const removeLink = () => {
    editor.chain().focus().unsetLink().run();
  };

  return (
    <div className={`border border-slate-300 rounded-md focus-within:ring-2 focus-within:ring-orange-500 focus-within:border-orange-500 ${className}`}>
      {/* Toolbar */}
      <div className="flex items-center gap-1 p-2 border-b border-slate-200 bg-slate-50">
        <button
          type="button"
          onClick={() => editor.chain().focus().toggleBold().run()}
          disabled={disabled}
          className={`px-2 py-1 text-sm rounded ${
            editor.isActive('bold') 
              ? 'bg-orange-200 text-orange-800' 
              : 'hover:bg-slate-200'
          } disabled:opacity-50`}
          title="Bold"
        >
          <strong>B</strong>
        </button>
        
        <button
          type="button"
          onClick={() => editor.chain().focus().toggleItalic().run()}
          disabled={disabled}
          className={`px-2 py-1 text-sm rounded ${
            editor.isActive('italic') 
              ? 'bg-orange-200 text-orange-800' 
              : 'hover:bg-slate-200'
          } disabled:opacity-50`}
          title="Italic"
        >
          <em>I</em>
        </button>

        <button
          type="button"
          onClick={() => editor.chain().focus().toggleStrike().run()}
          disabled={disabled}
          className={`px-2 py-1 text-sm rounded ${
            editor.isActive('strike') 
              ? 'bg-orange-200 text-orange-800' 
              : 'hover:bg-slate-200'
          } disabled:opacity-50`}
          title="Strikethrough"
        >
          <s>S</s>
        </button>

        <div className="w-px h-4 bg-slate-300 mx-1" />

        <button
          type="button"
          onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
          disabled={disabled}
          className={`px-2 py-1 text-sm rounded ${
            editor.isActive('heading', { level: 2 }) 
              ? 'bg-orange-200 text-orange-800' 
              : 'hover:bg-slate-200'
          } disabled:opacity-50`}
          title="Heading 2"
        >
          H2
        </button>

        <button
          type="button"
          onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
          disabled={disabled}
          className={`px-2 py-1 text-sm rounded ${
            editor.isActive('heading', { level: 3 }) 
              ? 'bg-orange-200 text-orange-800' 
              : 'hover:bg-slate-200'
          } disabled:opacity-50`}
          title="Heading 3"
        >
          H3
        </button>

        <div className="w-px h-4 bg-slate-300 mx-1" />

        <button
          type="button"
          onClick={() => editor.chain().focus().toggleBulletList().run()}
          disabled={disabled}
          className={`px-2 py-1 text-sm rounded ${
            editor.isActive('bulletList') 
              ? 'bg-orange-200 text-orange-800' 
              : 'hover:bg-slate-200'
          } disabled:opacity-50`}
          title="Bullet List"
        >
          •
        </button>

        <button
          type="button"
          onClick={() => editor.chain().focus().toggleOrderedList().run()}
          disabled={disabled}
          className={`px-2 py-1 text-sm rounded ${
            editor.isActive('orderedList') 
              ? 'bg-orange-200 text-orange-800' 
              : 'hover:bg-slate-200'
          } disabled:opacity-50`}
          title="Numbered List"
        >
          1.
        </button>

        <div className="w-px h-4 bg-slate-300 mx-1" />

        <button
          type="button"
          onClick={() => setIsLinkModalOpen(true)}
          disabled={disabled}
          className={`px-2 py-1 text-sm rounded ${
            editor.isActive('link') 
              ? 'bg-orange-200 text-orange-800' 
              : 'hover:bg-slate-200'
          } disabled:opacity-50`}
          title="Add Link"
        >
          🔗
        </button>

        {editor.isActive('link') && (
          <button
            type="button"
            onClick={removeLink}
            disabled={disabled}
            className="px-2 py-1 text-sm rounded hover:bg-red-200 disabled:opacity-50"
            title="Remove Link"
          >
            🚫
          </button>
        )}

        <div className="w-px h-4 bg-slate-300 mx-1" />

        <button
          type="button"
          onClick={() => editor.chain().focus().toggleBlockquote().run()}
          disabled={disabled}
          className={`px-2 py-1 text-sm rounded ${
            editor.isActive('blockquote') 
              ? 'bg-orange-200 text-orange-800' 
              : 'hover:bg-slate-200'
          } disabled:opacity-50`}
          title="Quote"
        >
          "
        </button>
      </div>

      {/* Editor Content */}
      <div 
        className="p-3 min-h-[200px] cursor-text"
        onClick={() => editor?.chain().focus().run()}
      >
        <EditorContent 
          editor={editor} 
          className="prose prose-sm max-w-none focus:outline-none focus-within:outline-none"
        />
      </div>

      {/* Link Modal */}
      {isLinkModalOpen && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold mb-4">Add Link</h3>
            <input
              type="url"
              value={linkUrl}
              onChange={(e) => setLinkUrl(e.target.value)}
              placeholder="https://example.com"
              className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500"
              autoFocus
            />
            <div className="flex justify-end gap-2 mt-4">
              <button
                type="button"
                onClick={() => setIsLinkModalOpen(false)}
                className="px-4 py-2 text-slate-600 hover:text-slate-800"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={addLink}
                className="px-4 py-2 bg-orange-600 text-white rounded-md hover:bg-orange-700"
              >
                Add Link
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Help Text */}
      <div className="px-3 py-2 text-xs text-slate-500 border-t border-slate-200 bg-slate-50">
        Use the toolbar for formatting. Links will open in new tabs for safety.
      </div>
    </div>
  );
}