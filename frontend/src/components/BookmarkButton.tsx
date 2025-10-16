'use client';

import { useState, useEffect } from 'react';
import Button from '@/components/ui/Button';
import { createBookmark, deleteBookmark, isWorkBookmarked, type CreateBookmarkRequest } from '@/lib/api';

interface BookmarkButtonProps {
  workId: string;
  authToken?: string;
  onBookmarkChange?: (isBookmarked: boolean) => void;
  variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger';
  size?: 'sm' | 'md' | 'lg';
}

export function BookmarkButton({ 
  workId, 
  authToken, 
  onBookmarkChange,
  variant = 'outline',
  size = 'sm'
}: BookmarkButtonProps) {
  const [isBookmarked, setIsBookmarked] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string>('');
  const [showBookmarkDialog, setShowBookmarkDialog] = useState(false);

  // Check initial bookmark status
  useEffect(() => {
    const checkBookmarkStatus = async () => {
      if (!authToken) return;
      
      try {
        const bookmarked = await isWorkBookmarked(workId, authToken);
        setIsBookmarked(bookmarked);
      } catch (error) {
        console.error('Failed to check bookmark status:', error);
      }
    };

    checkBookmarkStatus();
  }, [workId, authToken]);

  const handleBookmarkClick = async () => {
    if (!authToken) {
      setError('Please log in to bookmark works');
      return;
    }

    if (isBookmarked) {
      // TODO: Implement removing bookmark (need bookmark ID)
      setError('Bookmark removal not yet implemented');
      return;
    }

    // Show bookmark creation dialog
    setShowBookmarkDialog(true);
  };

  const handleCreateBookmark = async (bookmarkData: CreateBookmarkRequest) => {
    setIsLoading(true);
    setError('');

    try {
      await createBookmark(workId, bookmarkData, authToken);
      setIsBookmarked(true);
      setShowBookmarkDialog(false);
      onBookmarkChange?.(true);
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Failed to create bookmark');
    } finally {
      setIsLoading(false);
    }
  };

  const handleCancelBookmark = () => {
    setShowBookmarkDialog(false);
    setError('');
  };

  if (!authToken) {
    return (
      <Button 
        variant={variant} 
        size={size}
        disabled
        title="Log in to bookmark works"
      >
        🔖 Bookmark
      </Button>
    );
  }

  return (
    <>
      <Button
        variant={isBookmarked ? 'primary' : variant}
        size={size}
        onClick={handleBookmarkClick}
        disabled={isLoading}
        title={isBookmarked ? 'Already bookmarked' : 'Bookmark this work'}
      >
        {isLoading ? '...' : isBookmarked ? '🔖 Bookmarked' : '🔖 Bookmark'}
      </Button>

      {error && (
        <div className="text-red-600 text-sm mt-1">
          {error}
        </div>
      )}

      {showBookmarkDialog && (
        <BookmarkDialog
          onSubmit={handleCreateBookmark}
          onCancel={handleCancelBookmark}
          isLoading={isLoading}
        />
      )}
    </>
  );
}

interface BookmarkDialogProps {
  onSubmit: (data: CreateBookmarkRequest) => void;
  onCancel: () => void;
  isLoading: boolean;
}

function BookmarkDialog({ onSubmit, onCancel, isLoading }: BookmarkDialogProps) {
  const [notes, setNotes] = useState('');
  const [tags, setTags] = useState('');
  const [isPrivate, setIsPrivate] = useState(false);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    const tagArray = tags
      .split(',')
      .map(tag => tag.trim())
      .filter(tag => tag.length > 0);

    onSubmit({
      notes: notes || undefined,
      tags: tagArray.length > 0 ? tagArray : undefined,
      is_private: isPrivate,
    });
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 w-full max-w-md mx-4">
        <h2 className="text-xl font-bold mb-4">Add Bookmark</h2>
        
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="notes" className="block text-sm font-medium mb-1">
              Notes (optional)
            </label>
            <textarea
              id="notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              className="w-full p-2 border border-gray-300 rounded-md"
              rows={3}
              placeholder="Add personal notes about this work..."
            />
          </div>

          <div>
            <label htmlFor="tags" className="block text-sm font-medium mb-1">
              Tags (optional)
            </label>
            <input
              type="text"
              id="tags"
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              className="w-full p-2 border border-gray-300 rounded-md"
              placeholder="tag1, tag2, tag3"
            />
            <p className="text-xs text-gray-500 mt-1">
              Separate tags with commas
            </p>
          </div>

          <div className="flex items-center">
            <input
              type="checkbox"
              id="isPrivate"
              checked={isPrivate}
              onChange={(e) => setIsPrivate(e.target.checked)}
              className="mr-2"
            />
            <label htmlFor="isPrivate" className="text-sm">
              Make bookmark private
            </label>
          </div>

          <div className="flex gap-2 pt-4">
            <Button
              type="submit"
              disabled={isLoading}
              className="flex-1"
            >
              {isLoading ? 'Adding...' : 'Add Bookmark'}
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={onCancel}
              disabled={isLoading}
            >
              Cancel
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}