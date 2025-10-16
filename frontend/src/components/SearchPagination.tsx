'use client';

import React, { useState, useId } from 'react';
import { ChevronLeft, ChevronRight, MoreHorizontal } from 'lucide-react';

interface SearchPaginationProps {
  currentPage: number;
  totalPages: number;
  totalItems: number;
  itemsPerPage: number;
  onPageChange: (page: number) => void;
  className?: string;
  isLoading?: boolean;
  showSummary?: boolean;
  showPageSizeSelector?: boolean;
  pageSizeOptions?: number[];
  onPageSizeChange?: (pageSize: number) => void;
}

const SearchPagination: React.FC<SearchPaginationProps> = ({
  currentPage,
  totalPages,
  totalItems,
  itemsPerPage,
  onPageChange,
  className = '',
  isLoading = false,
  showSummary = true,
  showPageSizeSelector = false,
  pageSizeOptions = [10, 20, 50, 100],
  onPageSizeChange
}) => {
  const [announceUpdate, setAnnounceUpdate] = useState('');
  
  // Accessibility IDs
  const paginationId = useId();
  const summaryId = useId();
  const liveRegionId = useId();

  // Calculate visible page range
  const getVisiblePages = () => {
    const maxVisible = 7; // Show up to 7 page numbers
    const halfVisible = Math.floor(maxVisible / 2);
    
    let startPage = Math.max(1, currentPage - halfVisible);
    let endPage = Math.min(totalPages, currentPage + halfVisible);
    
    // Adjust if we're near the beginning or end
    if (endPage - startPage + 1 < maxVisible) {
      if (startPage === 1) {
        endPage = Math.min(totalPages, startPage + maxVisible - 1);
      } else if (endPage === totalPages) {
        startPage = Math.max(1, endPage - maxVisible + 1);
      }
    }
    
    return { startPage, endPage };
  };

  // Handle page change with accessibility announcements
  const handlePageChange = (page: number, reason: string) => {
    if (page === currentPage || page < 1 || page > totalPages || isLoading) {
      return;
    }
    
    onPageChange(page);
    setAnnounceUpdate(`${reason} Page ${page} of ${totalPages} loaded`);
  };

  // Handle keyboard navigation
  const handleKeyDown = (e: React.KeyboardEvent, page: number, reason: string) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handlePageChange(page, reason);
    }
  };

  // Handle page size change
  const handlePageSizeChange = (newPageSize: number) => {
    onPageSizeChange?.(newPageSize);
    setAnnounceUpdate(`Page size changed to ${newPageSize} items per page`);
  };

  // Calculate current range
  const startItem = (currentPage - 1) * itemsPerPage + 1;
  const endItem = Math.min(currentPage * itemsPerPage, totalItems);

  if (totalPages <= 1) {
    return null; // Don't show pagination if there's only one page
  }

  const { startPage, endPage } = getVisiblePages();

  return (
    <nav
      role="navigation"
      aria-labelledby={`${paginationId}-label`}
      className={`flex items-center justify-between ${className}`}
    >
      {/* Live Region for Announcements */}
      <div
        id={liveRegionId}
        aria-live="polite"
        aria-atomic="true"
        className="sr-only"
      >
        {announceUpdate}
      </div>

      {/* Pagination Label */}
      <h3 id={`${paginationId}-label`} className="sr-only">
        Search Results Pagination
      </h3>

      {/* Summary */}
      {showSummary && (
        <div 
          id={summaryId}
          className="text-sm text-gray-700"
          role="status"
          aria-live="polite"
        >
          <span className="font-medium">
            Showing {startItem.toLocaleString()} to {endItem.toLocaleString()} of{' '}
            {totalItems.toLocaleString()} results
          </span>
        </div>
      )}

      {/* Pagination Controls */}
      <div className="flex items-center space-x-2">
        {/* Page Size Selector */}
        {showPageSizeSelector && onPageSizeChange && (
          <div className="flex items-center space-x-2 mr-4">
            <label 
              htmlFor={`${paginationId}-page-size`}
              className="text-sm text-gray-700 whitespace-nowrap"
            >
              Show:
            </label>
            <select
              id={`${paginationId}-page-size`}
              value={itemsPerPage}
              onChange={(e) => handlePageSizeChange(Number(e.target.value))}
              className="px-2 py-1 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              disabled={isLoading}
            >
              {pageSizeOptions.map(size => (
                <option key={size} value={size}>
                  {size} per page
                </option>
              ))}
            </select>
          </div>
        )}

        {/* Previous Button */}
        <button
          type="button"
          onClick={() => handlePageChange(currentPage - 1, 'Previous page.')}
          onKeyDown={(e) => handleKeyDown(e, currentPage - 1, 'Previous page.')}
          disabled={currentPage <= 1 || isLoading}
          className="relative inline-flex items-center px-2 py-2 text-sm font-medium text-gray-500 bg-white border border-gray-300 rounded-l-md hover:bg-gray-50 focus:z-10 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
          aria-label={`Go to previous page, page ${currentPage - 1}`}
          title="Previous page"
        >
          <ChevronLeft className="h-5 w-5" aria-hidden="true" />
          <span className="sr-only">Previous</span>
        </button>

        {/* Page Numbers */}
        <div className="relative z-0 inline-flex shadow-sm -space-x-px" role="group" aria-label="Pagination">
          {/* First page if not in visible range */}
          {startPage > 1 && (
            <>
              <button
                type="button"
                onClick={() => handlePageChange(1, 'First page.')}
                onKeyDown={(e) => handleKeyDown(e, 1, 'First page.')}
                className="relative inline-flex items-center px-4 py-2 text-sm font-medium text-gray-500 bg-white border border-gray-300 hover:bg-gray-50 focus:z-10 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                aria-label="Go to page 1"
                disabled={isLoading}
              >
                1
              </button>
              
              {startPage > 2 && (
                <span 
                  className="relative inline-flex items-center px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300"
                  aria-hidden="true"
                >
                  <MoreHorizontal className="h-4 w-4" />
                </span>
              )}
            </>
          )}

          {/* Visible page range */}
          {Array.from({ length: endPage - startPage + 1 }, (_, i) => {
            const page = startPage + i;
            const isCurrent = page === currentPage;
            
            return (
              <button
                key={page}
                type="button"
                onClick={() => !isCurrent && handlePageChange(page, `Page ${page}.`)}
                onKeyDown={(e) => !isCurrent && handleKeyDown(e, page, `Page ${page}.`)}
                className={`relative inline-flex items-center px-4 py-2 text-sm font-medium border focus:z-10 focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                  isCurrent
                    ? 'z-10 bg-blue-50 border-blue-500 text-blue-600'
                    : 'text-gray-500 bg-white border-gray-300 hover:bg-gray-50 focus:border-blue-500'
                }`}
                aria-label={isCurrent ? `Current page, page ${page}` : `Go to page ${page}`}
                aria-current={isCurrent ? 'page' : undefined}
                disabled={isCurrent || isLoading}
              >
                {page}
              </button>
            );
          })}

          {/* Last page if not in visible range */}
          {endPage < totalPages && (
            <>
              {endPage < totalPages - 1 && (
                <span 
                  className="relative inline-flex items-center px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300"
                  aria-hidden="true"
                >
                  <MoreHorizontal className="h-4 w-4" />
                </span>
              )}
              
              <button
                type="button"
                onClick={() => handlePageChange(totalPages, 'Last page.')}
                onKeyDown={(e) => handleKeyDown(e, totalPages, 'Last page.')}
                className="relative inline-flex items-center px-4 py-2 text-sm font-medium text-gray-500 bg-white border border-gray-300 hover:bg-gray-50 focus:z-10 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                aria-label={`Go to last page, page ${totalPages}`}
                disabled={isLoading}
              >
                {totalPages}
              </button>
            </>
          )}
        </div>

        {/* Next Button */}
        <button
          type="button"
          onClick={() => handlePageChange(currentPage + 1, 'Next page.')}
          onKeyDown={(e) => handleKeyDown(e, currentPage + 1, 'Next page.')}
          disabled={currentPage >= totalPages || isLoading}
          className="relative inline-flex items-center px-2 py-2 text-sm font-medium text-gray-500 bg-white border border-gray-300 rounded-r-md hover:bg-gray-50 focus:z-10 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
          aria-label={`Go to next page, page ${currentPage + 1}`}
          title="Next page"
        >
          <ChevronRight className="h-5 w-5" aria-hidden="true" />
          <span className="sr-only">Next</span>
        </button>
      </div>

      {/* Mobile Pagination Summary */}
      <div className="md:hidden mt-2 text-center">
        <span className="text-sm text-gray-700">
          Page {currentPage} of {totalPages}
        </span>
      </div>

      {/* Jump to Page Input (for large paginations) */}
      {totalPages > 10 && (
        <div className="flex items-center space-x-2 ml-4">
          <label 
            htmlFor={`${paginationId}-jump-to`}
            className="text-sm text-gray-700 whitespace-nowrap"
          >
            Go to page:
          </label>
          <input
            id={`${paginationId}-jump-to`}
            type="number"
            min="1"
            max={totalPages}
            className="w-16 px-2 py-1 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                const target = e.target as HTMLInputElement;
                const page = parseInt(target.value);
                if (page >= 1 && page <= totalPages) {
                  handlePageChange(page, `Jumped to page ${page}.`);
                  target.value = '';
                }
              }
            }}
            disabled={isLoading}
            placeholder={currentPage.toString()}
            aria-describedby={`${paginationId}-jump-help`}
          />
          <div id={`${paginationId}-jump-help`} className="sr-only">
            Enter a page number between 1 and {totalPages}, then press Enter
          </div>
        </div>
      )}
    </nav>
  );
};

export default SearchPagination;