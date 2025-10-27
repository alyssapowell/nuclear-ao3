import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { rest } from 'msw';
import { setupServer } from 'msw/node';
import WorksList from '../components/works/WorksList';
import UserProfile from '../components/profile/UserProfile';
import SearchResults from '../components/search/SearchResults';
import LoginForm from '../components/auth/LoginForm';
import WorkForm from '../components/works/WorkForm';

// Mock server for API testing
const server = setupServer(
  // Auth endpoints
  rest.post('/api/auth/login', (req, res, ctx) => {
    return res(
      ctx.status(200),
      ctx.json({
        token: 'mock-jwt-token',
        user: {
          id: '1',
          username: 'testuser',
          email: 'test@example.com'
        }
      })
    );
  }),

  rest.post('/api/auth/logout', (req, res, ctx) => {
    return res(ctx.status(200), ctx.json({ success: true }));
  }),

  // Works endpoints
  rest.get('/api/works', (req, res, ctx) => {
    const page = req.url.searchParams.get('page') || '1';
    const limit = req.url.searchParams.get('limit') || '20';
    
    return res(
      ctx.status(200),
      ctx.json({
        works: [
          {
            id: '1',
            title: 'Test Work 1',
            summary: 'This is a test work',
            author: { id: '1', username: 'author1' },
            tags: ['Test Tag'],
            kudos: 10,
            hits: 100,
            createdAt: '2023-01-01T00:00:00Z'
          },
          {
            id: '2',
            title: 'Test Work 2',
            summary: 'Another test work',
            author: { id: '2', username: 'author2' },
            tags: ['Another Tag'],
            kudos: 15,
            hits: 150,
            createdAt: '2023-01-02T00:00:00Z'
          }
        ],
        pagination: {
          page: parseInt(page),
          limit: parseInt(limit),
          total: 100,
          totalPages: 5
        }
      })
    );
  }),

  rest.get('/api/works/:id', (req, res, ctx) => {
    const { id } = req.params;
    return res(
      ctx.status(200),
      ctx.json({
        id,
        title: `Test Work ${id}`,
        summary: 'This is a test work',
        content: 'This is the work content.',
        author: { id: '1', username: 'author1' },
        tags: ['Test Tag'],
        kudos: 10,
        hits: 100,
        createdAt: '2023-01-01T00:00:00Z'
      })
    );
  }),

  rest.post('/api/works', (req, res, ctx) => {
    return res(
      ctx.status(201),
      ctx.json({
        id: '3',
        title: 'New Test Work',
        summary: 'A newly created work',
        author: { id: '1', username: 'testuser' },
        createdAt: new Date().toISOString()
      })
    );
  }),

  // Search endpoints
  rest.get('/api/search/works', (req, res, ctx) => {
    const query = req.url.searchParams.get('q') || '';
    return res(
      ctx.status(200),
      ctx.json({
        results: [
          {
            id: '1',
            title: `Search Result for "${query}"`,
            summary: 'This work matches your search',
            author: { id: '1', username: 'author1' },
            tags: [query],
            kudos: 5,
            hits: 50
          }
        ],
        total: 1,
        query
      })
    );
  }),

  // User endpoints
  rest.get('/api/users/:id', (req, res, ctx) => {
    const { id } = req.params;
    return res(
      ctx.status(200),
      ctx.json({
        id,
        username: `user${id}`,
        bio: 'This is a test user bio',
        joinedAt: '2023-01-01T00:00:00Z',
        worksCount: 5,
        kudosGiven: 50,
        kudosReceived: 25
      })
    );
  }),

  // Tags endpoints
  rest.get('/api/tags/autocomplete', (req, res, ctx) => {
    const query = req.url.searchParams.get('q') || '';
    return res(
      ctx.status(200),
      ctx.json([
        { id: '1', name: `${query}tag1`, type: 'freeform' },
        { id: '2', name: `${query}tag2`, type: 'character' },
        { id: '3', name: `${query}tag3`, type: 'relationship' }
      ])
    );
  }),

  // Comments endpoints
  rest.get('/api/works/:workId/comments', (req, res, ctx) => {
    return res(
      ctx.status(200),
      ctx.json([
        {
          id: '1',
          content: 'Great work!',
          author: { id: '2', username: 'commenter1' },
          createdAt: '2023-01-01T12:00:00Z'
        },
        {
          id: '2',
          content: 'I loved this chapter',
          author: { id: '3', username: 'commenter2' },
          createdAt: '2023-01-01T14:00:00Z'
        }
      ])
    );
  }),

  rest.post('/api/works/:workId/comments', (req, res, ctx) => {
    return res(
      ctx.status(201),
      ctx.json({
        id: '3',
        content: 'New comment',
        author: { id: '1', username: 'testuser' },
        createdAt: new Date().toISOString()
      })
    );
  }),

  // Notifications endpoints
  rest.get('/api/notifications', (req, res, ctx) => {
    return res(
      ctx.status(200),
      ctx.json([
        {
          id: '1',
          type: 'comment',
          title: 'New comment',
          message: 'Someone commented on your work',
          read: false,
          createdAt: '2023-01-01T10:00:00Z'
        }
      ])
    );
  }),

  // Error handlers
  rest.get('/api/error-test', (req, res, ctx) => {
    return res(ctx.status(500), ctx.json({ error: 'Internal server error' }));
  }),

  rest.get('/api/timeout-test', (req, res, ctx) => {
    return res(ctx.delay(5000), ctx.status(200));
  })
);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

// Mock authentication utilities
jest.mock('../utils/auth', () => ({
  useAuth: jest.fn(() => ({
    user: { id: '1', username: 'testuser' },
    isAuthenticated: true,
    token: 'mock-jwt-token',
    logout: jest.fn(),
  })),
  getAuthState: jest.fn(() => ({
    user: { id: '1', username: 'testuser' },
    isAuthenticated: true,
    token: 'mock-jwt-token',
  })),
  isAuthenticated: jest.fn(() => true),
}));

describe('API Integration Tests', () => {
  describe('Authentication API', () => {
    test('successfully logs in user with valid credentials', async () => {
      const user = userEvent.setup();
      
      render(<LoginForm />);

      await user.type(screen.getByLabelText(/username/i), 'testuser');
      await user.type(screen.getByLabelText(/password/i), 'password123');
      
      const loginButton = screen.getByRole('button', { name: /login/i });
      await user.click(loginButton);

      await waitFor(() => {
        expect(screen.getByText(/login successful/i)).toBeInTheDocument();
      });
    });

    test('handles authentication errors gracefully', async () => {
      server.use(
        rest.post('/api/auth/login', (req, res, ctx) => {
          return res(
            ctx.status(401),
            ctx.json({ error: 'Invalid credentials' })
          );
        })
      );

      const user = userEvent.setup();
      render(<LoginForm />);

      await user.type(screen.getByLabelText(/username/i), 'wronguser');
      await user.type(screen.getByLabelText(/password/i), 'wrongpass');
      
      const loginButton = screen.getByRole('button', { name: /login/i });
      await user.click(loginButton);

      await waitFor(() => {
        expect(screen.getByText(/invalid credentials/i)).toBeInTheDocument();
      });
    });

    test('handles network errors during authentication', async () => {
      server.use(
        rest.post('/api/auth/login', (req, res, ctx) => {
          return res.networkError('Network error');
        })
      );

      const user = userEvent.setup();
      render(<LoginForm />);

      await user.type(screen.getByLabelText(/username/i), 'testuser');
      await user.type(screen.getByLabelText(/password/i), 'password123');
      
      const loginButton = screen.getByRole('button', { name: /login/i });
      await user.click(loginButton);

      await waitFor(() => {
        expect(screen.getByText(/network error/i)).toBeInTheDocument();
      });
    });
  });

  describe('Works API', () => {
    test('fetches and displays works list', async () => {
      render(<WorksList />);

      await waitFor(() => {
        expect(screen.getByText('Test Work 1')).toBeInTheDocument();
        expect(screen.getByText('Test Work 2')).toBeInTheDocument();
      });
    });

    test('handles pagination correctly', async () => {
      const user = userEvent.setup();
      render(<WorksList />);

      await waitFor(() => {
        expect(screen.getByText('Test Work 1')).toBeInTheDocument();
      });

      // Test pagination controls
      const nextButton = screen.getByRole('button', { name: /next/i });
      expect(nextButton).toBeInTheDocument();

      const pageInfo = screen.getByText(/page 1 of 5/i);
      expect(pageInfo).toBeInTheDocument();
    });

    test('creates new work successfully', async () => {
      const user = userEvent.setup();
      render(<WorkForm />);

      // Fill out form
      await user.type(screen.getByLabelText(/title/i), 'New Test Work');
      await user.type(screen.getByLabelText(/summary/i), 'A newly created work');
      await user.type(screen.getByLabelText(/content/i), 'This is the work content.');

      // Add tags
      const tagInput = screen.getByLabelText(/tags/i);
      await user.type(tagInput, 'test tag');
      await user.keyboard('{Enter}');

      const publishButton = screen.getByRole('button', { name: /publish/i });
      await user.click(publishButton);

      await waitFor(() => {
        expect(screen.getByText(/work published successfully/i)).toBeInTheDocument();
      });
    });

    test('handles work creation errors', async () => {
      server.use(
        rest.post('/api/works', (req, res, ctx) => {
          return res(
            ctx.status(400),
            ctx.json({ 
              errors: { 
                title: 'Title is required',
                summary: 'Summary is required' 
              } 
            })
          );
        })
      );

      const user = userEvent.setup();
      render(<WorkForm />);

      const publishButton = screen.getByRole('button', { name: /publish/i });
      await user.click(publishButton);

      await waitFor(() => {
        expect(screen.getByText(/title is required/i)).toBeInTheDocument();
        expect(screen.getByText(/summary is required/i)).toBeInTheDocument();
      });
    });

    test('loads individual work details', async () => {
      render(<WorkDetails workId="1" />);

      await waitFor(() => {
        expect(screen.getByText('Test Work 1')).toBeInTheDocument();
        expect(screen.getByText('This is the work content.')).toBeInTheDocument();
      });
    });
  });

  describe('Search API', () => {
    test('performs search and displays results', async () => {
      const user = userEvent.setup();
      render(<SearchResults />);

      const searchInput = screen.getByLabelText(/search/i);
      await user.type(searchInput, 'test query');
      
      const searchButton = screen.getByRole('button', { name: /search/i });
      await user.click(searchButton);

      await waitFor(() => {
        expect(screen.getByText('Search Result for "test query"')).toBeInTheDocument();
      });
    });

    test('handles empty search results', async () => {
      server.use(
        rest.get('/api/search/works', (req, res, ctx) => {
          return res(
            ctx.status(200),
            ctx.json({ results: [], total: 0, query: 'nosuchwork' })
          );
        })
      );

      const user = userEvent.setup();
      render(<SearchResults />);

      const searchInput = screen.getByLabelText(/search/i);
      await user.type(searchInput, 'nosuchwork');
      
      const searchButton = screen.getByRole('button', { name: /search/i });
      await user.click(searchButton);

      await waitFor(() => {
        expect(screen.getByText(/no results found/i)).toBeInTheDocument();
      });
    });

    test('handles search API errors', async () => {
      server.use(
        rest.get('/api/search/works', (req, res, ctx) => {
          return res(ctx.status(500), ctx.json({ error: 'Search service unavailable' }));
        })
      );

      const user = userEvent.setup();
      render(<SearchResults />);

      const searchInput = screen.getByLabelText(/search/i);
      await user.type(searchInput, 'test');
      
      const searchButton = screen.getByRole('button', { name: /search/i });
      await user.click(searchButton);

      await waitFor(() => {
        expect(screen.getByText(/search service unavailable/i)).toBeInTheDocument();
      });
    });
  });

  describe('User Profile API', () => {
    test('loads user profile data', async () => {
      render(<UserProfile userId="1" />);

      await waitFor(() => {
        expect(screen.getByText('user1')).toBeInTheDocument();
        expect(screen.getByText('This is a test user bio')).toBeInTheDocument();
        expect(screen.getByText('5 works')).toBeInTheDocument();
      });
    });

    test('handles user not found errors', async () => {
      server.use(
        rest.get('/api/users/:id', (req, res, ctx) => {
          return res(ctx.status(404), ctx.json({ error: 'User not found' }));
        })
      );

      render(<UserProfile userId="999" />);

      await waitFor(() => {
        expect(screen.getByText(/user not found/i)).toBeInTheDocument();
      });
    });
  });

  describe('Tags API', () => {
    test('provides tag autocomplete suggestions', async () => {
      const user = userEvent.setup();
      render(<WorkForm />);

      const tagInput = screen.getByLabelText(/tags/i);
      await user.type(tagInput, 'test');

      await waitFor(() => {
        expect(screen.getByText('testtag1')).toBeInTheDocument();
        expect(screen.getByText('testtag2')).toBeInTheDocument();
        expect(screen.getByText('testtag3')).toBeInTheDocument();
      });
    });

    test('handles tag autocomplete errors', async () => {
      server.use(
        rest.get('/api/tags/autocomplete', (req, res, ctx) => {
          return res(ctx.status(500), ctx.json({ error: 'Tag service unavailable' }));
        })
      );

      const user = userEvent.setup();
      render(<WorkForm />);

      const tagInput = screen.getByLabelText(/tags/i);
      await user.type(tagInput, 'test');

      await waitFor(() => {
        expect(screen.getByText(/tag suggestions unavailable/i)).toBeInTheDocument();
      });
    });
  });

  describe('Comments API', () => {
    test('loads work comments', async () => {
      render(<WorkComments workId="1" />);

      await waitFor(() => {
        expect(screen.getByText('Great work!')).toBeInTheDocument();
        expect(screen.getByText('I loved this chapter')).toBeInTheDocument();
      });
    });

    test('posts new comment successfully', async () => {
      const user = userEvent.setup();
      render(<CommentForm workId="1" />);

      const commentField = screen.getByLabelText(/comment/i);
      await user.type(commentField, 'This is a great work!');

      const postButton = screen.getByRole('button', { name: /post comment/i });
      await user.click(postButton);

      await waitFor(() => {
        expect(screen.getByText(/comment posted successfully/i)).toBeInTheDocument();
      });
    });

    test('handles comment posting errors', async () => {
      server.use(
        rest.post('/api/works/:workId/comments', (req, res, ctx) => {
          return res(
            ctx.status(400),
            ctx.json({ error: 'Comment content is required' })
          );
        })
      );

      const user = userEvent.setup();
      render(<CommentForm workId="1" />);

      const postButton = screen.getByRole('button', { name: /post comment/i });
      await user.click(postButton);

      await waitFor(() => {
        expect(screen.getByText(/comment content is required/i)).toBeInTheDocument();
      });
    });
  });

  describe('Notifications API', () => {
    test('fetches user notifications', async () => {
      render(<NotificationsList />);

      await waitFor(() => {
        expect(screen.getByText('New comment')).toBeInTheDocument();
        expect(screen.getByText('Someone commented on your work')).toBeInTheDocument();
      });
    });

    test('handles notifications API errors', async () => {
      server.use(
        rest.get('/api/notifications', (req, res, ctx) => {
          return res(ctx.status(500), ctx.json({ error: 'Notifications service unavailable' }));
        })
      );

      render(<NotificationsList />);

      await waitFor(() => {
        expect(screen.getByText(/notifications service unavailable/i)).toBeInTheDocument();
      });
    });
  });

  describe('Error Handling and Recovery', () => {
    test('displays generic error message for 500 errors', async () => {
      render(<TestComponent apiEndpoint="/api/error-test" />);

      await waitFor(() => {
        expect(screen.getByText(/something went wrong/i)).toBeInTheDocument();
      });
    });

    test('handles request timeouts', async () => {
      render(<TestComponent apiEndpoint="/api/timeout-test" />);

      await waitFor(() => {
        expect(screen.getByText(/request timed out/i)).toBeInTheDocument();
      }, { timeout: 6000 });
    });

    test('provides retry functionality after errors', async () => {
      const user = userEvent.setup();
      
      let callCount = 0;
      server.use(
        rest.get('/api/works', (req, res, ctx) => {
          callCount++;
          if (callCount === 1) {
            return res(ctx.status(500), ctx.json({ error: 'Server error' }));
          }
          return res(
            ctx.status(200),
            ctx.json({ works: [], pagination: { page: 1, total: 0 } })
          );
        })
      );

      render(<WorksList />);

      await waitFor(() => {
        expect(screen.getByText(/server error/i)).toBeInTheDocument();
      });

      const retryButton = screen.getByRole('button', { name: /retry/i });
      await user.click(retryButton);

      await waitFor(() => {
        expect(screen.queryByText(/server error/i)).not.toBeInTheDocument();
      });
    });

    test('handles malformed JSON responses', async () => {
      server.use(
        rest.get('/api/works', (req, res, ctx) => {
          return res(ctx.text('Not JSON'));
        })
      );

      render(<WorksList />);

      await waitFor(() => {
        expect(screen.getByText(/invalid response format/i)).toBeInTheDocument();
      });
    });
  });

  describe('API Request Optimization', () => {
    test('implements request caching', async () => {
      const requestSpy = jest.fn();
      
      server.use(
        rest.get('/api/works/:id', (req, res, ctx) => {
          requestSpy();
          return res(
            ctx.status(200),
            ctx.json({ id: '1', title: 'Cached Work' })
          );
        })
      );

      // First render
      const { rerender } = render(<WorkDetails workId="1" />);
      
      await waitFor(() => {
        expect(screen.getByText('Cached Work')).toBeInTheDocument();
      });

      // Second render should use cache
      rerender(<WorkDetails workId="1" />);

      expect(requestSpy).toHaveBeenCalledTimes(1); // Should only be called once due to caching
    });

    test('implements request deduplication', async () => {
      const requestSpy = jest.fn();
      
      server.use(
        rest.get('/api/works', (req, res, ctx) => {
          requestSpy();
          return res(
            ctx.status(200),
            ctx.json({ works: [], pagination: { page: 1, total: 0 } })
          );
        })
      );

      // Render multiple components that make the same request
      render(
        <div>
          <WorksList />
          <WorksList />
          <WorksList />
        </div>
      );

      await waitFor(() => {
        expect(requestSpy).toHaveBeenCalledTimes(1); // Should be deduplicated
      });
    });

    test('cancels pending requests on component unmount', async () => {
      const { unmount } = render(<WorksList />);
      
      // Unmount before request completes
      unmount();

      // Should not show any error or warning about cancelled requests
      expect(screen.queryByText(/error/i)).not.toBeInTheDocument();
    });
  });

  describe('API Security', () => {
    test('includes authentication token in authorized requests', async () => {
      let authHeader = '';
      
      server.use(
        rest.get('/api/user/profile', (req, res, ctx) => {
          authHeader = req.headers.get('Authorization') || '';
          return res(ctx.status(200), ctx.json({ username: 'testuser' }));
        })
      );

      render(<UserProfile userId="1" />);

      await waitFor(() => {
        expect(authHeader).toBe('Bearer mock-jwt-token');
      });
    });

    test('handles token expiration', async () => {
      server.use(
        rest.get('/api/works', (req, res, ctx) => {
          return res(
            ctx.status(401),
            ctx.json({ error: 'Token expired' })
          );
        })
      );

      render(<WorksList />);

      await waitFor(() => {
        expect(screen.getByText(/session expired.*please log in again/i)).toBeInTheDocument();
      });
    });

    test('sanitizes user input in API requests', async () => {
      let requestBody = '';
      
      server.use(
        rest.post('/api/works/:workId/comments', async (req, res, ctx) => {
          requestBody = await req.text();
          return res(ctx.status(201), ctx.json({ id: '1' }));
        })
      );

      const user = userEvent.setup();
      render(<CommentForm workId="1" />);

      const maliciousInput = '<script>alert("XSS")</script>Legitimate comment';
      const commentField = screen.getByLabelText(/comment/i);
      await user.type(commentField, maliciousInput);

      const postButton = screen.getByRole('button', { name: /post comment/i });
      await user.click(postButton);

      await waitFor(() => {
        // Should escape or strip malicious content
        expect(requestBody).not.toContain('<script>');
        expect(requestBody).toContain('Legitimate comment');
      });
    });
  });
});

// Helper components for testing
const TestComponent = ({ apiEndpoint }: { apiEndpoint: string }) => {
  const [data, setData] = React.useState(null);
  const [error, setError] = React.useState('');

  React.useEffect(() => {
    fetch(apiEndpoint)
      .then(res => res.json())
      .then(setData)
      .catch(err => setError(err.message));
  }, [apiEndpoint]);

  if (error) return <div>Error: {error}</div>;
  return <div>Data loaded</div>;
};

const WorkDetails = ({ workId }: { workId: string }) => {
  const [work, setWork] = React.useState(null);

  React.useEffect(() => {
    fetch(`/api/works/${workId}`)
      .then(res => res.json())
      .then(setWork);
  }, [workId]);

  if (!work) return <div>Loading...</div>;
  return (
    <div>
      <h1>{work.title}</h1>
      <p>{work.content}</p>
    </div>
  );
};

const WorkComments = ({ workId }: { workId: string }) => {
  const [comments, setComments] = React.useState([]);

  React.useEffect(() => {
    fetch(`/api/works/${workId}/comments`)
      .then(res => res.json())
      .then(setComments);
  }, [workId]);

  return (
    <div>
      {comments.map(comment => (
        <div key={comment.id}>{comment.content}</div>
      ))}
    </div>
  );
};

const CommentForm = ({ workId }: { workId: string }) => {
  const [content, setContent] = React.useState('');
  const [message, setMessage] = React.useState('');
  const [error, setError] = React.useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      const response = await fetch(`/api/works/${workId}/comments`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content })
      });
      
      if (response.ok) {
        setMessage('Comment posted successfully');
        setContent('');
      } else {
        const errorData = await response.json();
        setError(errorData.error);
      }
    } catch (err) {
      setError(err.message);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <label htmlFor="comment">Comment</label>
      <textarea
        id="comment"
        value={content}
        onChange={(e) => setContent(e.target.value)}
      />
      <button type="submit">Post Comment</button>
      {message && <div>{message}</div>}
      {error && <div>{error}</div>}
    </form>
  );
};

const NotificationsList = () => {
  const [notifications, setNotifications] = React.useState([]);
  const [error, setError] = React.useState('');

  React.useEffect(() => {
    fetch('/api/notifications')
      .then(res => {
        if (!res.ok) throw new Error('Notifications service unavailable');
        return res.json();
      })
      .then(setNotifications)
      .catch(err => setError(err.message));
  }, []);

  if (error) return <div>{error}</div>;

  return (
    <div>
      {notifications.map(notification => (
        <div key={notification.id}>
          <h3>{notification.title}</h3>
          <p>{notification.message}</p>
        </div>
      ))}
    </div>
  );
};