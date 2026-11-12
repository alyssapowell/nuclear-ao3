import type { MockedFunction } from 'vitest';
import { render, screen, fireEvent, waitFor, act, within } from '@testing-library/react';
import '@testing-library/jest-dom';
import Comments from '../Comments';
import { getWorkComments, createComment, updateComment, deleteComment } from '@/lib/api';

// Mock the API functions
vi.mock('@/lib/api', () => ({
  getWorkComments: vi.fn(),
  createComment: vi.fn(),
  updateComment: vi.fn(),
  deleteComment: vi.fn(),
}));

const mockGetWorkComments = getWorkComments as MockedFunction<typeof getWorkComments>;
const mockCreateComment = createComment as MockedFunction<typeof createComment>;
const mockUpdateComment = updateComment as MockedFunction<typeof updateComment>;
const mockDeleteComment = deleteComment as MockedFunction<typeof deleteComment>;

// Helper function to render Comments component wrapped in act
const renderCommentsWithAct = async (props: any) => {
  let result: any;
  await act(async () => {
    result = render(<Comments {...props} />);
  });
  return result;
};

describe('Comments', () => {
  const mockComments = [
    {
      id: '1',
      work_id: 'work-1',
      content: 'This is a great story!',
      pseud: { id: 'pseud-1', name: 'TestUser' },
      author_type: 'user' as const,
      status: 'published' as const,
      is_anonymous: false,
      kudos_count: 5,
      has_kudos: false,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
    {
      id: '2',
      work_id: 'work-1',
      parent_comment_id: '1',
      content: 'Thank you for reading!',
      pseud: { id: 'pseud-2', name: 'Author' },
      author_type: 'user' as const,
      status: 'published' as const,
      is_anonymous: false,
      kudos_count: 2,
      has_kudos: true,
      created_at: '2024-01-01T01:00:00Z',
      updated_at: '2024-01-01T01:00:00Z',
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    mockGetWorkComments.mockResolvedValue({ comments: mockComments });
    mockCreateComment.mockResolvedValue({ 
      comment: {
        id: '3',
        work_id: 'work-1',
        content: 'New comment',
        pseud: { id: 'pseud-3', name: 'NewUser' },
        author_type: 'user' as const,
        status: 'published' as const,
        is_anonymous: false,
        kudos_count: 0,
        has_kudos: false,
        created_at: '2024-01-01T02:00:00Z',
        updated_at: '2024-01-01T02:00:00Z',
      }
    });
  });

  it('renders comments correctly', async () => {
    await renderCommentsWithAct({ workId: "work-1" });

    await waitFor(() => {
      expect(screen.getByText('Comments (2)')).toBeInTheDocument();
      expect(screen.getByText('This is a great story!')).toBeInTheDocument();
      expect(screen.getByText('Thank you for reading!')).toBeInTheDocument();
      expect(screen.getByText('TestUser')).toBeInTheDocument();
      expect(screen.getByText('Author')).toBeInTheDocument();
    });
  });

  it('shows loading state initially', async () => {
    // Delay the mock to see loading state
    mockGetWorkComments.mockImplementation(() => new Promise(resolve => setTimeout(() => resolve({ comments: mockComments }), 100)));
    
    await renderCommentsWithAct({ workId: "work-1" });
    
    expect(screen.getByText('Loading comments...')).toBeInTheDocument();
  });

  it('displays comment form when comments are allowed', async () => {
    await renderCommentsWithAct({ workId: "work-1", allowComments: true });

    await waitFor(() => {
      expect(screen.getByText('Leave a Comment')).toBeInTheDocument();
      expect(screen.getByPlaceholderText('Share your thoughts about this work...')).toBeInTheDocument();
      expect(screen.getByText('Post Comment')).toBeInTheDocument();
    });
  });

  it('hides comment form when comments are not allowed', async () => {
    await renderCommentsWithAct({ workId: "work-1", allowComments: false });

    await waitFor(() => {
      expect(screen.queryByText('Leave a Comment')).not.toBeInTheDocument();
      expect(screen.queryByPlaceholderText('Share your thoughts about this work...')).not.toBeInTheDocument();
    });
  });

  it('submits a new comment', async () => {
    await renderCommentsWithAct({ workId: "work-1", allowComments: true });

    await waitFor(() => {
      expect(screen.getByText('Leave a Comment')).toBeInTheDocument();
    });

    const textarea = screen.getByPlaceholderText('Share your thoughts about this work...');
    const submitButton = screen.getByText('Post Comment');

    fireEvent.change(textarea, { target: { value: 'This is my new comment' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(mockCreateComment).toHaveBeenCalledWith(
        'work-1',
        {
          content: 'This is my new comment',
          chapter_id: undefined,
          parent_comment_id: undefined,
          is_anonymous: false,
        },
        undefined
      );
    });
  });

  it('shows reply form when reply button is clicked', async () => {
    await renderCommentsWithAct({ workId: "work-1", allowComments: true });

    await waitFor(() => {
      expect(screen.getByText('This is a great story!')).toBeInTheDocument();
    });

    const replyButtons = screen.getAllByText('Reply');
    fireEvent.click(replyButtons[0]);

    expect(screen.getByPlaceholderText('Write your reply...')).toBeInTheDocument();
    expect(screen.getByText('Post Reply')).toBeInTheDocument();
  });

  it('submits a reply to a comment', async () => {
    await renderCommentsWithAct({ workId: "work-1", allowComments: true });

    await waitFor(() => {
      expect(screen.getByText('This is a great story!')).toBeInTheDocument();
    });

    // Click reply button
    const replyButtons = screen.getAllByText('Reply');
    fireEvent.click(replyButtons[0]);

    // Fill in reply
    const replyTextarea = screen.getByPlaceholderText('Write your reply...');
    fireEvent.change(replyTextarea, { target: { value: 'This is my reply' } });

    // Submit reply
    const postReplyButton = screen.getByText('Post Reply');
    fireEvent.click(postReplyButton);

    await waitFor(() => {
      expect(mockCreateComment).toHaveBeenCalledWith(
        'work-1',
        {
          content: 'This is my reply',
          chapter_id: undefined,
          parent_comment_id: '1',
          is_anonymous: false,
        },
        undefined
      );
    });
  });

  it('handles anonymous comments', async () => {
    await renderCommentsWithAct({ workId: "work-1", allowComments: true });

    await waitFor(() => {
      expect(screen.getByText('Leave a Comment')).toBeInTheDocument();
    });

    const textarea = screen.getByPlaceholderText('Share your thoughts about this work...');
    const anonymousCheckbox = screen.getByLabelText('Post anonymously');
    const submitButton = screen.getByText('Post Comment');

    fireEvent.change(textarea, { target: { value: 'Anonymous comment' } });
    fireEvent.click(anonymousCheckbox);
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(mockCreateComment).toHaveBeenCalledWith(
        'work-1',
        {
          content: 'Anonymous comment',
          chapter_id: undefined,
          parent_comment_id: undefined,
          is_anonymous: true,
        },
        undefined
      );
    });
  });

  it('shows empty state when no comments exist', async () => {
    mockGetWorkComments.mockResolvedValue({ comments: [] });
    
    await renderCommentsWithAct({ workId: "work-1" });

    await waitFor(() => {
      expect(screen.getByText('No comments yet.')).toBeInTheDocument();
    });
  });

  it('handles comment loading error', async () => {
    mockGetWorkComments.mockRejectedValue(new Error('Failed to load comments'));
    
    await renderCommentsWithAct({ workId: "work-1" });

    await waitFor(() => {
      expect(screen.getByText('Failed to load comments')).toBeInTheDocument();
    });
  });

  it('shows threaded comments correctly', async () => {
    await renderCommentsWithAct({ workId: "work-1" });

    await waitFor(() => {
      // Parent comment
      expect(screen.getByText('This is a great story!')).toBeInTheDocument();
      
      // Reply should be indented/nested - look for the wrapper div with ml-6 class
      const replyText = screen.getByText('Thank you for reading!');
      const replyContainer = replyText.closest('.ml-6');
      expect(replyContainer).toBeInTheDocument();
      expect(replyContainer).toHaveClass('ml-6', 'border-l', 'border-slate-200', 'pl-4');
    });
  });

  it('disables submit button when comment is empty', async () => {
    await renderCommentsWithAct({ workId: "work-1", allowComments: true });

    await waitFor(() => {
      const submitButton = screen.getByText('Post Comment');
      expect(submitButton).toBeDisabled();
    });
  });

  it('enables submit button when comment has content', async () => {
    await renderCommentsWithAct({ workId: "work-1", allowComments: true });

    await waitFor(() => {
      const textarea = screen.getByPlaceholderText('Share your thoughts about this work...');
      const submitButton = screen.getByText('Post Comment');

      fireEvent.change(textarea, { target: { value: 'Some content' } });
      
      expect(submitButton).not.toBeDisabled();
    });
  });

  describe('Comment editing', () => {
    it('should show edit button for user comments when authenticated', async () => {
      const commentsWithAuthType = mockComments.map(comment => ({
        ...comment,
        author_type: 'user' as const
      }));
      
      mockGetWorkComments.mockResolvedValue({ comments: commentsWithAuthType });
      
      await renderCommentsWithAct({ workId: "work-1", authToken: "test-token" });
      
      waitFor(() => {
        expect(screen.getAllByText('Edit')).toHaveLength(2);
      });
    });

    it('should not show edit button for guest comments', async () => {
      const guestComments = mockComments.map(comment => ({
        ...comment,
        author_type: 'guest' as const
      }));
      
      mockGetWorkComments.mockResolvedValue({ comments: guestComments });
      
      await renderCommentsWithAct({ workId: "work-1", authToken: "test-token" });
      
      waitFor(() => {
        expect(screen.queryByText('Edit')).not.toBeInTheDocument();
      });
    });

    it('should open edit form when edit button is clicked', async () => {
      const commentsWithAuthType = mockComments.map(comment => ({
        ...comment,
        author_type: 'user' as const
      }));
      
      mockGetWorkComments.mockResolvedValue({ comments: commentsWithAuthType });
      
      await renderCommentsWithAct({ workId: "work-1", authToken: "test-token" });
      
      await waitFor(() => {
        expect(screen.getByText('This is a great story!')).toBeInTheDocument();
      });
      
      const editButtons = screen.getAllByText('Edit');
      fireEvent.click(editButtons[0]);
      
      await waitFor(() => {
        expect(screen.getByDisplayValue('This is a great story!')).toBeInTheDocument();
        expect(screen.getByText('Save Changes')).toBeInTheDocument();
        expect(screen.getByText('Cancel')).toBeInTheDocument();
      });
    });

    it('should successfully update comment when edit form is submitted', async () => {
      const commentsWithAuthType = mockComments.map(comment => ({
        ...comment,
        author_type: 'user' as const
      }));
      
      mockGetWorkComments.mockResolvedValue({ comments: commentsWithAuthType });
      mockUpdateComment.mockResolvedValue({
        ...commentsWithAuthType[0],
        content: 'Updated comment content',
        updated_at: '2024-01-01T02:00:00Z',
        edited_at: '2024-01-01T02:00:00Z'
      });
      
      await renderCommentsWithAct({ workId: "work-1", authToken: "test-token" });
      
      await waitFor(() => {
        expect(screen.getByText('This is a great story!')).toBeInTheDocument();
      });
      
      const editButtons = screen.getAllByText('Edit');
      fireEvent.click(editButtons[0]);
      
      const textArea = screen.getByDisplayValue('This is a great story!');
      fireEvent.change(textArea, { target: { value: 'Updated comment content' } });
      
      const saveButton = screen.getByText('Save Changes');
      fireEvent.click(saveButton);
      
      await waitFor(() => {
        expect(mockUpdateComment).toHaveBeenCalledWith(
          '1',
          { content: 'Updated comment content' },
          'test-token'
        );
      });
    });

    it('should cancel edit when cancel button is clicked', async () => {
      const commentsWithAuthType = mockComments.map(comment => ({
        ...comment,
        author_type: 'user' as const
      }));
      
      mockGetWorkComments.mockResolvedValue({ comments: commentsWithAuthType });
      
      await renderCommentsWithAct({ workId: "work-1", authToken: "test-token" });
      
      await waitFor(() => {
        expect(screen.getByText('This is a great story!')).toBeInTheDocument();
      });
      
      const editButtons = screen.getAllByText('Edit');
      fireEvent.click(editButtons[0]);
      
      const cancelButton = screen.getByText('Cancel');
      fireEvent.click(cancelButton);
      
      await waitFor(() => {
        expect(screen.queryByDisplayValue('This is a great story!')).not.toBeInTheDocument();
        expect(screen.queryByText('Save Changes')).not.toBeInTheDocument();
      });
    });
  });

  describe('Comment deletion', () => {
    it('should show delete button for user comments when authenticated', async () => {
      const commentsWithAuthType = mockComments.map(comment => ({
        ...comment,
        author_type: 'user' as const
      }));
      
      mockGetWorkComments.mockResolvedValue({ comments: commentsWithAuthType });
      
      await renderCommentsWithAct({ workId: "work-1", authToken: "test-token" });
      
      waitFor(() => {
        expect(screen.getAllByText('Delete')).toHaveLength(2);
      });
    });

    it('should show delete confirmation when delete button is clicked', async () => {
      const commentsWithAuthType = mockComments.map(comment => ({
        ...comment,
        author_type: 'user' as const
      }));
      
      mockGetWorkComments.mockResolvedValue({ comments: commentsWithAuthType });
      
      await renderCommentsWithAct({ workId: "work-1", authToken: "test-token" });
      
      await waitFor(() => {
        expect(screen.getByText('This is a great story!')).toBeInTheDocument();
      });
      
      const deleteButtons = screen.getAllByText('Delete');
      fireEvent.click(deleteButtons[0]);
      
      await waitFor(() => {
        expect(screen.getByText(/Are you sure you want to delete this comment/)).toBeInTheDocument();
        expect(screen.getByText('Delete Comment')).toBeInTheDocument();
        expect(screen.getByText('Cancel')).toBeInTheDocument();
      });
    });

    it('should successfully delete comment when confirmed', async () => {
      const commentsWithAuthType = mockComments.map(comment => ({
        ...comment,
        author_type: 'user' as const
      }));
      
      mockGetWorkComments.mockResolvedValue({ comments: commentsWithAuthType });
      mockDeleteComment.mockResolvedValue({ message: 'Comment deleted successfully' });
      
      await renderCommentsWithAct({ workId: "work-1", authToken: "test-token" });
      
      await waitFor(() => {
        expect(screen.getByText('This is a great story!')).toBeInTheDocument();
      });
      
      const deleteButtons = screen.getAllByText('Delete');
      fireEvent.click(deleteButtons[0]);
      
      const confirmButton = screen.getByText('Delete Comment');
      fireEvent.click(confirmButton);
      
      await waitFor(() => {
        expect(mockDeleteComment).toHaveBeenCalledWith('1', 'test-token');
      });
    });

    it('should cancel delete when cancel button is clicked', async () => {
      const commentsWithAuthType = mockComments.map(comment => ({
        ...comment,
        author_type: 'user' as const
      }));
      
      mockGetWorkComments.mockResolvedValue({ comments: commentsWithAuthType });
      
      await renderCommentsWithAct({ workId: "work-1", authToken: "test-token" });
      
      await waitFor(() => {
        expect(screen.getByText('This is a great story!')).toBeInTheDocument();
      });
      
      const deleteButtons = screen.getAllByText('Delete');
      fireEvent.click(deleteButtons[0]);
      
      const cancelButton = screen.getByText('Cancel');
      fireEvent.click(cancelButton);
      
      await waitFor(() => {
        expect(screen.queryByText(/Are you sure you want to delete this comment/)).not.toBeInTheDocument();
        expect(screen.queryByText('Delete Comment')).not.toBeInTheDocument();
      });
    });
  });

  describe('Comment permissions', () => {
    it('should not show edit/delete buttons when not authenticated', async () => {
      await renderCommentsWithAct({ workId: "work-1" });
      
      waitFor(() => {
        expect(screen.queryByText('Edit')).not.toBeInTheDocument();
        expect(screen.queryByText('Delete')).not.toBeInTheDocument();
      });
    });

    it('should show edited timestamp when comment has been edited', async () => {
      const editedComments = mockComments.map(comment => ({
        ...comment,
        edited_at: '2024-01-01T02:00:00Z'
      }));
      
      mockGetWorkComments.mockResolvedValue({ comments: editedComments });
      
      await renderCommentsWithAct({ workId: "work-1" });
      
      await waitFor(() => {
        expect(screen.getAllByText(/edited/)).toHaveLength(2);
      });
    });
  });
});