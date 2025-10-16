// GraphQL client configuration for Nuclear AO3
import { ApolloClient, InMemoryCache, createHttpLink, from } from '@apollo/client';
import { setContext } from '@apollo/client/link/context';
import { onError } from '@apollo/client/link/error';

// API Gateway endpoint
const API_GATEWAY_URL = process.env.NEXT_PUBLIC_API_GATEWAY_URL || 'http://localhost:8080';

// HTTP link to GraphQL endpoint
const httpLink = createHttpLink({
  uri: `${API_GATEWAY_URL}/graphql`,
});

// Auth link to include JWT token
const authLink = setContext((_, { headers }) => {
  // Get token from localStorage (or cookies in production)
  const token = typeof window !== 'undefined' ? localStorage.getItem('auth_token') : null;
  
  return {
    headers: {
      ...headers,
      authorization: token ? `Bearer ${token}` : '',
    }
  };
});

// Error link for handling GraphQL and network errors
const errorLink = onError(({ graphQLErrors, networkError, operation, forward }) => {
  if (graphQLErrors) {
    graphQLErrors.forEach(({ message, locations, path }) => {
      console.error(
        `[GraphQL error]: Message: ${message}, Location: ${locations}, Path: ${path}`
      );
    });
  }

  if (networkError) {
    console.error(`[Network error]: ${networkError}`);
    
    // Handle authentication errors
    if ('statusCode' in networkError && networkError.statusCode === 401) {
      // Clear token and redirect to login
      if (typeof window !== 'undefined') {
        localStorage.removeItem('auth_token');
        window.location.href = '/auth/login';
      }
    }
  }
});

// Apollo Client instance
export const apolloClient = new ApolloClient({
  link: from([errorLink, authLink, httpLink]),
  cache: new InMemoryCache({
    typePolicies: {
      Work: {
        fields: {
          chapters: {
            merge(existing = [], incoming) {
              return incoming;
            },
          },
          comments: {
            merge(existing = [], incoming) {
              return incoming;
            },
          },
          fandoms: {
            merge(existing = [], incoming) {
              // Ensure fandoms is always an array, never null
              return incoming || [];
            },
          },
          characters: {
            merge(existing = [], incoming) {
              // Ensure characters is always an array, never null
              return incoming || [];
            },
          },
          relationships: {
            merge(existing = [], incoming) {
              // Ensure relationships is always an array, never null
              return incoming || [];
            },
          },
          freeformTags: {
            merge(existing = [], incoming) {
              // Ensure freeformTags is always an array, never null
              return incoming || [];
            },
          },
          freeform_tags: {
            merge(existing = [], incoming) {
              // Handle both naming conventions
              return incoming || [];
            },
          },
        },
      },
      SearchResult: {
        fields: {
          works: {
            merge(existing = [], incoming) {
              return incoming;
            },
          },
          facets: {
            merge(existing = [], incoming) {
              return incoming;
            },
          },
        },
      },
    },
  }),
  defaultOptions: {
    watchQuery: {
      errorPolicy: 'all',
    },
    query: {
      errorPolicy: 'all',
    },
  },
});

// GraphQL Queries and Mutations
import { gql } from '@apollo/client';

// =============================================================================
// SEARCH QUERIES
// =============================================================================

export const SEARCH_WORKS = gql`
  query SearchWorks(
    $query: String
    $filters: WorkFilters
    $sort: SortOrder
    $pagination: PaginationInput
  ) {
    search {
      works(query: $query, filters: $filters, sort: $sort, pagination: $pagination) {
        total
        works {
          id
          title
          summary
          wordCount
          chapterCount
          maxChapters
          isComplete
          language
          rating
          warnings
          categories
          fandoms {
            id
            name
          }
          characters {
            id
            name
          }
          relationships {
            id
            name
          }
          freeformTags {
            id
            name
          }
          authors {
            id
            username
          }
          publishedAt
          updatedAt
          kudosCount
          commentCount
          bookmarkCount
          hitCount
        }
        facets {
          field
          values {
            value
            count
          }
        }
        recommendations {
          missingTags {
            type
            suggestions {
              name
              confidence
              reason
            }
          }
          qualityScore
        }
      }
    }
  }
`;

export const ENHANCED_SEARCH_WORKS = gql`
  query EnhancedSearchWorks(
    $query: String
    $filters: AdvancedWorkFilters
    $analysis: SearchAnalysisOptions
  ) {
    search {
      enhancedWorks(query: $query, filters: $filters, analysis: $analysis) {
        total
        works {
          id
          title
          summary
          wordCount
          chapterCount
          maxChapters
          isComplete
          language
          rating
          warnings
          categories
          fandoms {
            id
            name
            canonical
            useCount
          }
          characters {
            id
            name
            canonical
            useCount
          }
          relationships {
            id
            name
            canonical
            useCount
          }
          freeformTags {
            id
            name
            canonical
            useCount
          }
          authors {
            id
            username
          }
          publishedAt
          updatedAt
          kudosCount
          commentCount
          bookmarkCount
          hitCount
          tagQuality {
            score
            missingCharacters
            missingSuggestions
            inconsistencies
          }
        }
        analytics {
          tagDistribution {
            field
            distribution {
              value
              count
              percentage
            }
          }
          qualityMetrics {
            averageTagQuality
            poorlyTaggedWorks
            totalWorks
          }
          trends {
            popularTags
            emergingTags
            relatedFandoms
          }
        }
        smartSuggestions {
          characterSuggestions {
            tag
            confidence
            reasons
          }
          relationshipExpansions {
            original
            suggested {
              name
              confidence
            }
          }
          crossTaggingOpportunities {
            category
            suggestions
          }
        }
      }
    }
  }
`;

export const TAG_AUTOCOMPLETE = gql`
  query TagAutocomplete(
    $query: String!
    $types: [TagType!]
    $fandomId: ID
    $limit: Int = 10
    $excludeIds: [ID!]
  ) {
    tags {
      autocomplete(
        query: $query
        types: $types
        fandomId: $fandomId
        limit: $limit
        excludeIds: $excludeIds
      ) {
        suggestions {
          id
          name
          type
          canonical
          useCount
          description
          relationships {
            id
            name
          }
        }
      }
    }
  }
`;

// =============================================================================
// WORK QUERIES
// =============================================================================

export const GET_WORK = gql`
  query GetWork($id: ID!) {
    work(id: $id) {
      id
      title
      summary
      notes
      wordCount
      chapterCount
      maxChapters
      isComplete
      language
      rating
      warnings
      categories
      status
      publishedAt
      updatedAt
      createdAt
      fandoms {
        id
        name
        canonical
      }
      characters {
        id
        name
        canonical
      }
      relationships {
        id
        name
        canonical
      }
      freeformTags {
        id
        name
        canonical
      }
      authors {
        id
        username
        isAnonymous
      }
      chapters {
        id
        number
        title
        summary
        notes
        endNotes
        content
        wordCount
        publishedAt
        updatedAt
      }
      stats {
        kudosCount
        commentCount
        bookmarkCount
        hitCount
      }
      userInteractions {
        hasKudos
        isBookmarked
        canEdit
        canComment
      }
    }
  }
`;

export const GET_WORKS = gql`
  query GetWorks($filters: WorkFilters, $sort: SortOrder, $pagination: PaginationInput) {
    works(filters: $filters, sort: $sort, pagination: $pagination) {
      total
      works {
        id
        title
        summary
        wordCount
        chapterCount
        maxChapters
        isComplete
        language
        rating
        warnings
        categories
        fandoms {
          id
          name
        }
        characters {
          id
          name
        }
        relationships {
          id
          name
        }
        freeformTags {
          id
          name
        }
        authors {
          id
          username
        }
        publishedAt
        updatedAt
        kudosCount
        commentCount
        bookmarkCount
        hitCount
      }
    }
  }
`;

// =============================================================================
// TAG QUERIES
// =============================================================================

export const GET_TAGS = gql`
  query GetTags($type: TagType, $search: String, $pagination: PaginationInput) {
    tags {
      list(type: $type, search: $search, pagination: $pagination) {
        total
        tags {
          id
          name
          type
          canonical
          useCount
          description
          relationships {
            id
            name
            type
          }
          prominence {
            score
            ranking
            category
          }
        }
      }
    }
  }
`;

export const GET_TAG_RELATIONSHIPS = gql`
  query GetTagRelationships($tagId: ID!) {
    tags {
      relationships(tagId: $tagId) {
        parents {
          id
          name
          type
          canonical
        }
        children {
          id
          name
          type
          canonical
        }
        synonyms {
          id
          name
          canonical
        }
        related {
          id
          name
          type
          strength
        }
      }
    }
  }
`;

// =============================================================================
// USER QUERIES
// =============================================================================

export const GET_USER_PROFILE = gql`
  query GetUserProfile($userId: ID) {
    user(id: $userId) {
      id
      username
      email
      profile {
        bio
        location
        website
        socialLinks
      }
      stats {
        workCount
        seriesCount
        bookmarkCount
        kudosReceived
        commentsReceived
      }
      preferences {
        language
        timezone
        emailNotifications
        privacy {
          showEmail
          showBookmarks
          showKudos
        }
      }
      createdAt
      lastActiveAt
    }
  }
`;

// =============================================================================
// MUTATIONS
// =============================================================================

export const CREATE_WORK = gql`
  mutation CreateWork($input: CreateWorkInput!) {
    createWork(input: $input) {
      work {
        id
        title
        summary
        wordCount
        chapterCount
        authors {
          id
          username
        }
        fandoms {
          id
          name
        }
        publishedAt
      }
      errors {
        field
        message
      }
    }
  }
`;

export const UPDATE_WORK = gql`
  mutation UpdateWork($id: ID!, $input: UpdateWorkInput!) {
    updateWork(id: $id, input: $input) {
      work {
        id
        title
        summary
        updatedAt
      }
      errors {
        field
        message
      }
    }
  }
`;

export const GIVE_KUDOS = gql`
  mutation GiveKudos($workId: ID!) {
    giveKudos(workId: $workId) {
      success
      kudosCount
      hasKudos
      errors {
        message
      }
    }
  }
`;

export const CREATE_BOOKMARK = gql`
  mutation CreateBookmark($workId: ID!, $input: CreateBookmarkInput!) {
    createBookmark(workId: $workId, input: $input) {
      bookmark {
        id
        notes
        tags
        isPrivate
        createdAt
      }
      errors {
        field
        message
      }
    }
  }
`;

export const CREATE_COMMENT = gql`
  mutation CreateComment($workId: ID!, $input: CreateCommentInput!) {
    createComment(workId: $workId, input: $input) {
      comment {
        id
        content
        authorName
        isAnonymous
        createdAt
      }
      errors {
        field
        message
      }
    }
  }
`;

// =============================================================================
// AUTHENTICATION MUTATIONS
// =============================================================================

export const LOGIN = gql`
  mutation Login($input: LoginInput!) {
    auth {
      login(input: $input) {
        token
        user {
          id
          username
          email
        }
        errors {
          field
          message
        }
      }
    }
  }
`;

export const REGISTER = gql`
  mutation Register($input: RegisterInput!) {
    auth {
      register(input: $input) {
        token
        user {
          id
          username
          email
        }
        errors {
          field
          message
        }
      }
    }
  }
`;

export const LOGOUT = gql`
  mutation Logout {
    auth {
      logout {
        success
      }
    }
  }
`;

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// Set authentication token in both localStorage and cookies
export function setAuthToken(token: string) {
  if (typeof window !== 'undefined') {
    localStorage.setItem('auth_token', token);
    // Also set as cookie for middleware
    document.cookie = `auth_token=${token}; path=/; max-age=${7 * 24 * 60 * 60}; SameSite=Lax`;
    // Reset Apollo cache to refetch with new auth
    apolloClient.resetStore();
  }
}

// Clear authentication token from both localStorage and cookies
export function clearAuthToken() {
  if (typeof window !== 'undefined') {
    localStorage.removeItem('auth_token');
    // Remove cookie as well
    document.cookie = 'auth_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT';
    // Clear Apollo cache
    apolloClient.clearStore();
  }
}

// Get current authentication token
export function getAuthToken(): string | null {
  if (typeof window !== 'undefined') {
    return localStorage.getItem('auth_token');
  }
  return null;
}

// Check if user is authenticated
export function isAuthenticated(): boolean {
  return getAuthToken() !== null;
}