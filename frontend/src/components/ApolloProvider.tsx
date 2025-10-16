'use client';

import { ApolloProvider as Provider } from '@apollo/client';
import { apolloClient } from '@/lib/graphql';

interface ApolloProviderProps {
  children: React.ReactNode;
}

export function ApolloProvider({ children }: ApolloProviderProps) {
  return <Provider client={apolloClient}>{children}</Provider>;
}