import { useRef, useCallback, useEffect, useState } from 'react';

interface WebSocketHookOptions {
  url: string;
  onMessage?: (data: any) => void;
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (error: Event) => void;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
}

interface WebSocketHookReturn {
  connect: () => void;
  disconnect: () => void;
  sendMessage: (message: any) => void;
  isConnected: boolean;
  reconnectAttempts: number;
}

export const useWebSocket = ({
  url,
  onMessage,
  onOpen,
  onClose,
  onError,
  reconnectInterval = 5000,
  maxReconnectAttempts = 5,
}: WebSocketHookOptions): WebSocketHookReturn => {
  const ws = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<NodeJS.Timeout | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [reconnectAttempts, setReconnectAttempts] = useState(0);

  const getAuthToken = (): string => {
    // Safe localStorage access for SSR compatibility
    if (typeof window !== 'undefined') {
      try {
        return localStorage.getItem('auth_token') || '';
      } catch (error) {
        console.warn('Could not access localStorage for auth token:', error);
      }
    }
    return '';
  };

  const getUserId = (): string => {
    // Safe localStorage access for SSR compatibility
    if (typeof window !== 'undefined') {
      try {
        return localStorage.getItem('user_id') || '';
      } catch (error) {
        console.warn('Could not access localStorage for user ID:', error);
      }
    }
    return '';
  };

  const cleanup = useCallback(() => {
    if (reconnectTimer.current) {
      clearTimeout(reconnectTimer.current);
      reconnectTimer.current = null;
    }
  }, []);

  const connect = useCallback(() => {
    if (ws.current?.readyState === WebSocket.OPEN) {
      return;
    }

    cleanup();

    try {
      // Add authentication to WebSocket URL
      const token = getAuthToken();
      const userId = getUserId();
      
      // Don't connect if we don't have a valid token and user ID
      if (!token || !userId) {
        console.log('Skipping WebSocket connection - no valid auth token or user ID');
        return;
      }
      
      const wsUrl = new URL(url);
      wsUrl.searchParams.set('token', token);
      wsUrl.searchParams.set('user_id', userId);
      
      ws.current = new WebSocket(wsUrl.toString());

      ws.current.onopen = () => {
        console.log('WebSocket connected');
        setIsConnected(true);
        setReconnectAttempts(0);
        onOpen?.();
      };

      ws.current.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          onMessage?.(data);
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };

      ws.current.onclose = (event) => {
        // Only log if it's not a normal closure or if we're authenticated
        const token = getAuthToken();
        if (event.code !== 1000 && token) {
          console.log('WebSocket disconnected:', event.code, event.reason);
        }
        setIsConnected(false);
        onClose?.();

        // Attempt to reconnect if not a manual disconnect and we have auth
        if (event.code !== 1000 && token) {
          setReconnectAttempts(prev => {
            const newAttempts = prev + 1;
            if (newAttempts <= maxReconnectAttempts) {
              reconnectTimer.current = setTimeout(() => {
                console.log(`Attempting to reconnect... (${newAttempts}/${maxReconnectAttempts})`);
                connect();
              }, reconnectInterval);
            }
            return newAttempts;
          });
        }
      };

      ws.current.onerror = (error) => {
        // Only log WebSocket errors if we have valid auth (otherwise it's expected)
        const token = getAuthToken();
        if (token) {
          console.error('WebSocket error:', error);
        }
        onError?.(error);
      };

    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
    }
  }, [url, onMessage, onOpen, onClose, onError, reconnectInterval, maxReconnectAttempts]);

  const disconnect = useCallback(() => {
    cleanup();
    setReconnectAttempts(0);
    
    if (ws.current) {
      ws.current.close(1000, 'Manual disconnect');
      ws.current = null;
    }
    setIsConnected(false);
  }, [cleanup]);

  const sendMessage = useCallback((message: any) => {
    if (ws.current?.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify(message));
    } else {
      console.warn('WebSocket is not connected. Message not sent:', message);
    }
  }, []);

  useEffect(() => {
    return () => {
      cleanup();
      if (ws.current) {
        ws.current.close();
      }
    };
  }, [cleanup]);

  return {
    connect,
    disconnect,
    sendMessage,
    isConnected,
    reconnectAttempts,
  };
};