'use client'

import { useState } from 'react'

interface APIEndpoint {
  method: string
  path: string
  description: string
  auth?: boolean
  params?: { name: string; type: string; required: boolean; description: string }[]
  body?: { name: string; type: string; required: boolean; description: string }[]
  response: string
  example?: string
}

interface APISection {
  title: string
  description: string
  endpoints: APIEndpoint[]
}

export default function APIDocsPage() {
  const [activeSection, setActiveSection] = useState<string>('auth')

  const apiSections: APISection[] = [
    {
      title: 'Authentication',
      description: 'User registration, login, and session management',
      endpoints: [
        {
          method: 'POST',
          path: '/api/v1/auth/register',
          description: 'Register a new user account',
          body: [
            { name: 'username', type: 'string', required: true, description: 'Unique username (3-20 characters)' },
            { name: 'email', type: 'string', required: true, description: 'Valid email address' },
            { name: 'password', type: 'string', required: true, description: 'Password (8+ characters)' },
            { name: 'confirmPassword', type: 'string', required: true, description: 'Password confirmation' },
            { name: 'dateOfBirth', type: 'string', required: true, description: 'ISO date (YYYY-MM-DD)' },
            { name: 'country', type: 'string', required: true, description: 'Country code' },
            { name: 'agreedToTerms', type: 'boolean', required: true, description: 'Must be true' }
          ],
          response: '{ "message": "User registered successfully", "user": User }',
          example: `curl -X POST http://localhost:8080/api/v1/auth/register \\
  -H "Content-Type: application/json" \\
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123",
    "confirmPassword": "password123",
    "dateOfBirth": "1990-01-01",
    "country": "US",
    "agreedToTerms": true
  }'`
        },
        {
          method: 'POST',
          path: '/api/v1/auth/login',
          description: 'Authenticate user and create session',
          body: [
            { name: 'username', type: 'string', required: true, description: 'Username or email' },
            { name: 'password', type: 'string', required: true, description: 'User password' }
          ],
          response: '{ "message": "Login successful", "user": User, "token": "jwt_token" }',
          example: `curl -X POST http://localhost:8080/api/v1/auth/login \\
  -H "Content-Type: application/json" \\
  -d '{"username": "testuser", "password": "password123"}'`
        },
        {
          method: 'POST',
          path: '/api/v1/auth/logout',
          description: 'End user session',
          auth: true,
          response: '{ "message": "Logged out successfully" }'
        },
        {
          method: 'GET',
          path: '/api/v1/auth/me',
          description: 'Get current user profile',
          auth: true,
          response: '{ "user": User }'
        }
      ]
    },
    {
      title: 'Works',
      description: 'Fanfiction works - creating, reading, updating, and deleting',
      endpoints: [
        {
          method: 'GET',
          path: '/api/v1/works',
          description: 'Browse and search works',
          params: [
            { name: 'q', type: 'string', required: false, description: 'Search query' },
            { name: 'fandom', type: 'string', required: false, description: 'Filter by fandom' },
            { name: 'rating', type: 'string', required: false, description: 'Content rating filter' },
            { name: 'page', type: 'number', required: false, description: 'Page number (default: 1)' },
            { name: 'limit', type: 'number', required: false, description: 'Results per page (default: 20)' }
          ],
          response: '{ "works": Work[], "pagination": PaginationInfo }',
          example: `curl "http://localhost:8080/api/v1/works?q=harry+potter&fandom=Harry+Potter&page=1"`
        },
        {
          method: 'GET',
          path: '/api/v1/works/:id',
          description: 'Get a specific work by ID',
          params: [
            { name: 'id', type: 'string', required: true, description: 'Work UUID or legacy ID' }
          ],
          response: '{ "work": Work }',
          example: `curl "http://localhost:8080/api/v1/works/123"`
        },
        {
          method: 'POST',
          path: '/api/v1/works',
          description: 'Create a new work',
          auth: true,
          body: [
            { name: 'title', type: 'string', required: true, description: 'Work title' },
            { name: 'summary', type: 'string', required: false, description: 'Work summary' },
            { name: 'fandoms', type: 'string[]', required: true, description: 'Array of fandom tags' },
            { name: 'rating', type: 'string', required: true, description: 'Content rating' },
            { name: 'warnings', type: 'string[]', required: true, description: 'Content warnings' },
            { name: 'categories', type: 'string[]', required: false, description: 'Relationship categories' },
            { name: 'characters', type: 'string[]', required: false, description: 'Character tags' },
            { name: 'relationships', type: 'string[]', required: false, description: 'Relationship tags' },
            { name: 'freeform_tags', type: 'string[]', required: false, description: 'Additional tags' },
            { name: 'chapters', type: 'Chapter[]', required: true, description: 'Array of chapters' }
          ],
          response: '{ "work": Work }',
          example: `curl -X POST http://localhost:8080/api/v1/works \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{
    "title": "My Fanfiction",
    "summary": "A great story...",
    "fandoms": ["Harry Potter"],
    "rating": "General Audiences",
    "warnings": ["No Archive Warnings Apply"],
    "chapters": [{
      "title": "Chapter 1",
      "content": "Once upon a time...",
      "summary": "The beginning"
    }]
  }'`
        },
        {
          method: 'GET',
          path: '/api/v1/works/:id/chapters',
          description: 'Get all chapters for a work',
          params: [
            { name: 'id', type: 'string', required: true, description: 'Work UUID' }
          ],
          response: '{ "chapters": Chapter[] }'
        },
        {
          method: 'GET',
          path: '/api/v1/works/:id/stats',
          description: 'Get work statistics',
          params: [
            { name: 'id', type: 'string', required: true, description: 'Work UUID' }
          ],
          response: '{ "stats": WorkStats }'
        }
      ]
    },
    {
      title: 'Search',
      description: 'Enhanced search with AI features and smart filtering',
      endpoints: [
        {
          method: 'POST',
          path: '/api/v1/search/enhanced',
          description: 'AI-powered enhanced search',
          body: [
            { name: 'query', type: 'string', required: true, description: 'Natural language search query' },
            { name: 'filters', type: 'object', required: false, description: 'Advanced filters' },
            { name: 'semantic', type: 'boolean', required: false, description: 'Enable semantic search' }
          ],
          response: '{ "results": Work[], "suggestions": string[], "analytics": SearchAnalytics }',
          example: `curl -X POST http://localhost:8080/api/v1/search/enhanced \\
  -H "Content-Type: application/json" \\
  -d '{
    "query": "enemies to lovers harry draco",
    "semantic": true,
    "filters": {"rating": "Teen And Up Audiences"}
  }'`
        },
        {
          method: 'GET',
          path: '/api/v1/search/suggestions',
          description: 'Get search suggestions and autocomplete',
          params: [
            { name: 'q', type: 'string', required: true, description: 'Partial query' },
            { name: 'type', type: 'string', required: false, description: 'Suggestion type (tag, fandom, etc.)' }
          ],
          response: '{ "suggestions": string[] }'
        }
      ]
    },
    {
      title: 'Tags',
      description: 'Tag management and suggestions',
      endpoints: [
        {
          method: 'GET',
          path: '/api/v1/tags/search',
          description: 'Search for tags with autocomplete',
          params: [
            { name: 'q', type: 'string', required: true, description: 'Tag search query' },
            { name: 'type', type: 'string', required: false, description: 'Tag type filter' },
            { name: 'limit', type: 'number', required: false, description: 'Max results (default: 10)' }
          ],
          response: '{ "tags": Tag[] }'
        },
        {
          method: 'GET',
          path: '/api/v1/tags/popular',
          description: 'Get popular tags by category',
          params: [
            { name: 'type', type: 'string', required: false, description: 'Tag type (fandom, character, etc.)' },
            { name: 'limit', type: 'number', required: false, description: 'Max results (default: 20)' }
          ],
          response: '{ "tags": Tag[] }'
        }
      ]
    },
    {
      title: 'Users',
      description: 'User profiles and social features',
      endpoints: [
        {
          method: 'GET',
          path: '/api/v1/users/:id',
          description: 'Get user profile',
          params: [
            { name: 'id', type: 'string', required: true, description: 'User UUID' }
          ],
          response: '{ "user": UserProfile }'
        },
        {
          method: 'GET',
          path: '/api/v1/users/:id/works',
          description: 'Get works by user',
          params: [
            { name: 'id', type: 'string', required: true, description: 'User UUID' },
            { name: 'page', type: 'number', required: false, description: 'Page number' }
          ],
          response: '{ "works": Work[], "pagination": PaginationInfo }'
        },
        {
          method: 'GET',
          path: '/api/v1/users/:id/bookmarks',
          description: 'Get user bookmarks',
          params: [
            { name: 'id', type: 'string', required: true, description: 'User UUID' }
          ],
          response: '{ "bookmarks": Bookmark[] }'
        }
      ]
    }
  ]

  const graphQLSchema = `
# Core Types
type User {
  id: ID!
  username: String!
  email: String!
  profile: UserProfile
  works: [Work!]!
  bookmarks: [Bookmark!]!
  createdAt: DateTime!
}

type Work {
  id: ID!
  title: String!
  summary: String
  fandoms: [String!]!
  rating: String!
  warnings: [String!]!
  characters: [String!]!
  relationships: [String!]!
  freeformTags: [String!]!
  wordCount: Int!
  chapterCount: Int!
  chapters: [Chapter!]!
  author: User!
  stats: WorkStats!
  publishedAt: DateTime!
}

type Chapter {
  id: ID!
  workId: ID!
  number: Int!
  title: String!
  content: String!
  summary: String
  wordCount: Int!
}

# Queries
type Query {
  # Works
  work(id: ID!): Work
  works(filters: WorkFilters): [Work!]!
  searchWorks(query: String!, filters: WorkFilters): SearchResult!
  
  # Users
  user(id: ID!): User
  currentUser: User
  
  # Tags
  searchTags(query: String!, type: TagType): [Tag!]!
  popularTags(type: TagType, limit: Int): [Tag!]!
}

# Mutations
type Mutation {
  # Auth
  register(input: RegisterInput!): AuthResult!
  login(input: LoginInput!): AuthResult!
  logout: Boolean!
  
  # Works
  createWork(input: CreateWorkInput!): Work!
  updateWork(id: ID!, input: UpdateWorkInput!): Work!
  deleteWork(id: ID!): Boolean!
  
  # Bookmarks
  createBookmark(workId: ID!, notes: String): Bookmark!
  deleteBookmark(id: ID!): Boolean!
}
  `

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-7xl mx-auto px-4 py-8">
        <div className="mb-8">
          <h1 className="text-4xl font-bold text-gray-900 mb-4">Nuclear AO3 API Documentation</h1>
          <p className="text-lg text-gray-600 mb-6">
            Complete API reference for the Nuclear AO3 platform. This documentation is automatically updated from the live API.
          </p>
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
            <div className="flex items-center">
              <div className="text-blue-600 text-sm font-medium">
                🚀 Base URL: <code className="bg-blue-100 px-2 py-1 rounded">http://localhost:8080</code>
              </div>
            </div>
          </div>
        </div>

        <div className="flex gap-8">
          {/* Sidebar */}
          <div className="w-64 flex-shrink-0">
            <div className="bg-white rounded-lg shadow-sm border p-4 sticky top-4">
              <h3 className="font-semibold text-gray-900 mb-4">API Sections</h3>
              <nav className="space-y-2">
                {apiSections.map((section) => (
                  <button
                    key={section.title.toLowerCase()}
                    onClick={() => setActiveSection(section.title.toLowerCase())}
                    className={`w-full text-left px-3 py-2 rounded-md text-sm font-medium transition-colors $${
                      activeSection === section.title.toLowerCase()
                        ? 'bg-orange-100 text-orange-700'
                        : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                    }`}
                  >
                    {section.title}
                  </button>
                ))}
                <button
                  onClick={() => setActiveSection('graphql')}
                  className={`w-full text-left px-3 py-2 rounded-md text-sm font-medium transition-colors $${
                    activeSection === 'graphql'
                      ? 'bg-orange-100 text-orange-700'
                      : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                  }`}
                >
                  GraphQL Schema
                </button>
              </nav>
            </div>
          </div>

          {/* Main Content */}
          <div className="flex-1">
            {activeSection === 'graphql' ? (
              <div className="bg-white rounded-lg shadow-sm border">
                <div className="p-6 border-b">
                  <h2 className="text-2xl font-bold text-gray-900">GraphQL Schema</h2>
                  <p className="text-gray-600 mt-2">
                    Complete GraphQL schema for type-safe API queries and mutations.
                  </p>
                  <div className="mt-4 p-3 bg-yellow-50 border border-yellow-200 rounded-md">
                    <p className="text-yellow-800 text-sm">
                      <strong>Note:</strong> GraphQL endpoint is available at <code>/graphql</code> with Apollo Studio for development.
                    </p>
                  </div>
                </div>
                <div className="p-6">
                  <pre className="bg-gray-900 text-gray-100 p-4 rounded-lg overflow-x-auto text-sm">
                    <code>{graphQLSchema}</code>
                  </pre>
                </div>
              </div>
            ) : (
              apiSections
                .filter(section => section.title.toLowerCase() === activeSection)
                .map((section) => (
                  <div key={section.title} className="space-y-6">
                    <div className="bg-white rounded-lg shadow-sm border">
                      <div className="p-6 border-b">
                        <h2 className="text-2xl font-bold text-gray-900">{section.title}</h2>
                        <p className="text-gray-600 mt-2">{section.description}</p>
                      </div>
                    </div>

                    {section.endpoints.map((endpoint, index) => (
                      <div key={index} className="bg-white rounded-lg shadow-sm border">
                        <div className="p-6">
                          <div className="flex items-center gap-3 mb-4">
                            <span className={`px-3 py-1 rounded-md text-sm font-bold $${
                              endpoint.method === 'GET' ? 'bg-green-100 text-green-800' :
                              endpoint.method === 'POST' ? 'bg-blue-100 text-blue-800' :
                              endpoint.method === 'PUT' ? 'bg-yellow-100 text-yellow-800' :
                              'bg-red-100 text-red-800'
                            }`}>
                              {endpoint.method}
                            </span>
                            <code className="text-lg font-mono bg-gray-100 px-3 py-1 rounded">
                              {endpoint.path}
                            </code>
                            {endpoint.auth && (
                              <span className="px-2 py-1 bg-orange-100 text-orange-800 text-xs rounded-md font-medium">
                                🔒 Auth Required
                              </span>
                            )}
                          </div>

                          <p className="text-gray-700 mb-4">{endpoint.description}</p>

                          {endpoint.params && endpoint.params.length > 0 && (
                            <div className="mb-4">
                              <h4 className="font-semibold text-gray-900 mb-2">Parameters</h4>
                              <div className="overflow-x-auto">
                                <table className="w-full text-sm">
                                  <thead>
                                    <tr className="border-b">
                                      <th className="text-left py-2">Name</th>
                                      <th className="text-left py-2">Type</th>
                                      <th className="text-left py-2">Required</th>
                                      <th className="text-left py-2">Description</th>
                                    </tr>
                                  </thead>
                                  <tbody>
                                    {endpoint.params.map((param, i) => (
                                      <tr key={i} className="border-b">
                                        <td className="py-2 font-mono text-blue-600">{param.name}</td>
                                        <td className="py-2 font-mono text-gray-600">{param.type}</td>
                                        <td className="py-2">
                                          {param.required ? (
                                            <span className="text-red-600 font-medium">Yes</span>
                                          ) : (
                                            <span className="text-gray-400">No</span>
                                          )}
                                        </td>
                                        <td className="py-2 text-gray-700">{param.description}</td>
                                      </tr>
                                    ))}
                                  </tbody>
                                </table>
                              </div>
                            </div>
                          )}

                          {endpoint.body && endpoint.body.length > 0 && (
                            <div className="mb-4">
                              <h4 className="font-semibold text-gray-900 mb-2">Request Body</h4>
                              <div className="overflow-x-auto">
                                <table className="w-full text-sm">
                                  <thead>
                                    <tr className="border-b">
                                      <th className="text-left py-2">Field</th>
                                      <th className="text-left py-2">Type</th>
                                      <th className="text-left py-2">Required</th>
                                      <th className="text-left py-2">Description</th>
                                    </tr>
                                  </thead>
                                  <tbody>
                                    {endpoint.body.map((field, i) => (
                                      <tr key={i} className="border-b">
                                        <td className="py-2 font-mono text-blue-600">{field.name}</td>
                                        <td className="py-2 font-mono text-gray-600">{field.type}</td>
                                        <td className="py-2">
                                          {field.required ? (
                                            <span className="text-red-600 font-medium">Yes</span>
                                          ) : (
                                            <span className="text-gray-400">No</span>
                                          )}
                                        </td>
                                        <td className="py-2 text-gray-700">{field.description}</td>
                                      </tr>
                                    ))}
                                  </tbody>
                                </table>
                              </div>
                            </div>
                          )}

                          <div className="mb-4">
                            <h4 className="font-semibold text-gray-900 mb-2">Response</h4>
                            <pre className="bg-gray-100 p-3 rounded text-sm overflow-x-auto">
                              <code>{endpoint.response}</code>
                            </pre>
                          </div>

                          {endpoint.example && (
                            <div className="mb-4">
                              <h4 className="font-semibold text-gray-900 mb-2">Example</h4>
                              <pre className="bg-gray-900 text-gray-100 p-3 rounded text-sm overflow-x-auto">
                                <code>{endpoint.example}</code>
                              </pre>
                            </div>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                ))
            )}
          </div>
        </div>
      </div>
    </div>
  )
}