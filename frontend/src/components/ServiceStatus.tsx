'use client';

import { useState, useEffect } from 'react';

interface ServiceStatus {
  name: string;
  url: string;
  status: 'online' | 'offline' | 'checking';
  responseTime?: number;
  lastChecked?: Date;
  error?: string;
}

// We'll get all service status from the API Gateway's health endpoint
const GATEWAY_HEALTH_URL = '/health';

export default function ServiceStatus() {
  const [serviceStatuses, setServiceStatuses] = useState<ServiceStatus[]>([]);
  const [isExpanded, setIsExpanded] = useState(false);
  const [gatewayStatus, setGatewayStatus] = useState<'online' | 'offline' | 'checking'>('checking');

  const checkAllServices = async () => {
    const startTime = Date.now();
    
    try {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 3000); // Shorter timeout
      
      const response = await fetch(GATEWAY_HEALTH_URL, {
        method: 'GET',
        mode: 'cors',
        signal: controller.signal
      });
      
      clearTimeout(timeoutId);
      
      const responseTime = Date.now() - startTime;
      const lastChecked = new Date();
      
      if (response.ok) {
        const healthData = await response.json();
        setGatewayStatus('online');
        
        // Parse the health data into our service status format
        const services: ServiceStatus[] = [];
        
        // Add API Gateway itself
        services.push({
          name: 'API Gateway',
          url: GATEWAY_HEALTH_URL,
          status: healthData.gateway === 'healthy' ? 'online' : 'offline',
          responseTime,
          lastChecked,
          error: healthData.gateway !== 'healthy' ? 'Service degraded' : undefined
        });

        // Add individual services from health data
        if (healthData.services) {
          Object.entries(healthData.services).forEach(([serviceName, serviceData]) => {
            const serviceInfo = serviceData as string;
            const isHealthy = serviceInfo.includes('healthy');
            const match = serviceInfo.match(/\(([^)]+)\)/);
            const serviceResponseTime = match ? parseFloat(match[1]) : undefined;
            
            services.push({
              name: serviceName.replace('-service', '').replace('-', ' ').replace(/\b\w/g, l => l.toUpperCase()),
              url: `/health`, // All go through gateway
              status: isHealthy ? 'online' : 'offline',
              responseTime: serviceResponseTime || responseTime,
              lastChecked,
              error: !isHealthy ? 'Service unhealthy' : undefined
            });
          });
        }
        
        setServiceStatuses(services);
      } else {
        setGatewayStatus('offline');
        setServiceStatuses([{
          name: 'API Gateway',
          url: GATEWAY_HEALTH_URL,
          status: 'offline',
          responseTime,
          lastChecked,
          error: `HTTP ${response.status}`
        }]);
      }
    } catch (error) {
      const responseTime = Date.now() - startTime;
      setGatewayStatus('offline');
      setServiceStatuses([{
        name: 'API Gateway',
        url: GATEWAY_HEALTH_URL,
        status: 'offline',
        responseTime,
        lastChecked: new Date(),
        error: error instanceof Error ? error.message : 'Connection failed'
      }]);
    }
  };

  useEffect(() => {
    // Delay initial check to not block page loading
    const initialDelay = setTimeout(() => {
      checkAllServices();
    }, 2000);
    
    // Check every 30 seconds (less frequent to reduce load)
    const interval = setInterval(checkAllServices, 30000);
    
    return () => {
      clearTimeout(initialDelay);
      clearInterval(interval);
    };
  }, []);

  const overallStatus = serviceStatuses.every(s => s.status === 'online') 
    ? 'online' 
    : serviceStatuses.some(s => s.status === 'checking') 
    ? 'checking' 
    : 'offline';

  const StatusIndicator = ({ status }: { status: ServiceStatus['status'] }) => {
    const colors = {
      online: 'bg-green-500',
      offline: 'bg-red-500', 
      checking: 'bg-yellow-500 animate-pulse'
    };
    
    return (
      <div className={`w-3 h-3 rounded-full ${colors[status]}`} />
    );
  };

  return (
    <div className="fixed bottom-4 right-4 z-50">
      {/* Compact status indicator */}
      <div 
        className="bg-white dark:bg-gray-800 rounded-lg shadow-lg border cursor-pointer hover:shadow-xl transition-shadow"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <div className="flex items-center gap-2 px-3 py-2">
          <StatusIndicator status={overallStatus} />
          <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
            Services ({serviceStatuses.filter(s => s.status === 'online').length}/{serviceStatuses.length})
          </span>
          <div className={`transform transition-transform text-gray-600 dark:text-gray-300 ${isExpanded ? 'rotate-180' : ''}`}>
            ▼
          </div>
        </div>

        {/* Expanded details */}
        {isExpanded && (
          <div className="border-t p-3 space-y-2 min-w-64">
            <div className="flex justify-between items-center">
              <h3 className="font-semibold text-sm text-gray-900 dark:text-gray-100">Service Status</h3>
              <button 
                onClick={(e) => { e.stopPropagation(); checkAllServices(); }}
                className="text-xs bg-blue-500 text-white px-2 py-1 rounded hover:bg-blue-600"
              >
                Refresh
              </button>
            </div>
            
            {serviceStatuses.map((service) => (
              <div key={service.name} className="flex items-center justify-between text-xs">
                <div className="flex items-center gap-2">
                  <StatusIndicator status={service.status} />
                  <span className="font-medium text-gray-900 dark:text-gray-100">{service.name}</span>
                </div>
                <div className="text-right">
                  {service.responseTime && (
                    <div className="text-gray-600 dark:text-gray-300">{service.responseTime}ms</div>
                  )}
                  {service.error && (
                    <div className="text-red-600 dark:text-red-400" title={service.error}>
                      {service.error.length > 15 ? `${service.error.slice(0, 15)}...` : service.error}
                    </div>
                  )}
                  {service.lastChecked && (
                    <div className="text-gray-500 dark:text-gray-400">
                      {service.lastChecked.toLocaleTimeString()}
                    </div>
                  )}
                </div>
              </div>
            ))}
            
            {/* Overall system status */}
            <div className="border-t pt-2 mt-2">
              <div className="text-xs text-gray-600">
                <strong>System Status:</strong>{' '}
                <span className={
                  overallStatus === 'online' 
                    ? 'text-green-600' 
                    : overallStatus === 'checking' 
                    ? 'text-yellow-600' 
                    : 'text-red-600'
                }>
                  {overallStatus === 'online' ? 'All Systems Operational' : 
                   overallStatus === 'checking' ? 'Checking Services...' : 
                   'Service Disruption Detected'}
                </span>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}