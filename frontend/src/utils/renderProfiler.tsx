import React, { useEffect, useRef } from 'react';

// Hook to track component renders and props changes
export function useRenderProfiler(componentName: string, props?: Record<string, any>) {
  const renderCount = useRef(0);
  const prevProps = useRef<Record<string, any>>();

  useEffect(() => {
    renderCount.current += 1;
    
    console.group(`🔄 ${componentName} Render #${renderCount.current}`);
    
    if (props && prevProps.current) {
      const changedProps: Record<string, { old: any; new: any }> = {};
      let hasChanges = false;
      
      Object.keys(props).forEach(key => {
        if (props[key] !== prevProps.current![key]) {
          changedProps[key] = {
            old: prevProps.current![key],
            new: props[key]
          };
          hasChanges = true;
        }
      });
      
      if (hasChanges) {
        console.log('📝 Changed props:', changedProps);
      } else {
        console.log('✅ No prop changes detected');
      }
    } else if (props) {
      console.log('🆕 Initial props:', props);
    }
    
    console.log('⏰ Render timestamp:', new Date().toISOString());
    console.groupEnd();
    
    prevProps.current = props ? { ...props } : undefined;
  });

  return renderCount.current;
}

// Component wrapper for profiling
export function withRenderProfiler<T extends Record<string, any>>(
  Component: React.ComponentType<T>,
  componentName: string
) {
  return React.memo((props: T) => {
    useRenderProfiler(componentName, props);
    return <Component {...props} />;
  }, (prevProps, nextProps) => {
    // Custom comparison for memo
    const keys = new Set([...Object.keys(prevProps), ...Object.keys(nextProps)]);
    for (const key of keys) {
      if (prevProps[key as keyof T] !== nextProps[key as keyof T]) {
        console.log(`🔍 ${componentName} memo: Props changed for key "${key}"`);
        return false;
      }
    }
    console.log(`✅ ${componentName} memo: Props unchanged, skipping render`);
    return true;
  });
}

// Hook to track why a component re-rendered
export function useWhyDidYouUpdate(name: string, props: Record<string, any>) {
  const previousProps = useRef<Record<string, any>>();
  
  useEffect(() => {
    if (previousProps.current) {
      const allKeys = Object.keys({...previousProps.current, ...props});
      const changedProps: Record<string, { from: any; to: any }> = {};
      
      allKeys.forEach(key => {
        if (previousProps.current![key] !== props[key]) {
          changedProps[key] = {
            from: previousProps.current![key],
            to: props[key],
          };
        }
      });
      
      if (Object.keys(changedProps).length) {
        console.log(`🔍 [WhyDidYouUpdate] ${name}`, changedProps);
      }
    }
    
    previousProps.current = props;
  });
}

// Performance boundary to measure render times
export function PerformanceBoundary({ 
  name, 
  children 
}: { 
  name: string; 
  children: React.ReactNode 
}) {
  useEffect(() => {
    const start = performance.now();
    return () => {
      const end = performance.now();
      console.log(`⏱️ ${name} render time: ${(end - start).toFixed(2)}ms`);
    };
  });

  return <>{children}</>;
}