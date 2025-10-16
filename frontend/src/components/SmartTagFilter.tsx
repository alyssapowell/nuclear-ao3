import { useState } from 'react';
import { Switch } from '@headlessui/react';

interface SmartTagFilterProps {
  onFilterChange: (filters: TagFilters) => void;
  className?: string;
}

export interface TagFilters {
  primaryOnly: boolean;
  hideTagSpam: boolean;
  maxRelationshipTags: number;
  minWordThreshold: number;
  hideCollections: boolean;
  showProminence: boolean;
}

export default function SmartTagFilter({ onFilterChange, className = '' }: SmartTagFilterProps) {
  const [filters, setFilters] = useState<TagFilters>({
    primaryOnly: false,
    hideTagSpam: false,
    maxRelationshipTags: 50,
    minWordThreshold: 0,
    hideCollections: false,
    showProminence: true,
  });

  const [showAdvanced, setShowAdvanced] = useState(false);

  const updateFilter = <K extends keyof TagFilters>(key: K, value: TagFilters[K]) => {
    const newFilters = { ...filters, [key]: value };
    setFilters(newFilters);
    onFilterChange(newFilters);
  };

  return (
    <div className={`smart-tag-filter bg-white border border-slate-200 rounded-lg p-4 ${className}`}>
      <h3 className="text-lg font-semibold text-slate-900 mb-4">Smart Tag Filtering</h3>

      {/* Quick Filters */}
      <div className="space-y-3 mb-4">
        <Switch.Group>
          <div className="flex items-center justify-between">
            <Switch.Label className="text-sm font-medium text-slate-700">
              Primary relationships only
              <span className="block text-xs text-slate-500">Hide works tagged with background/side relationships</span>
            </Switch.Label>
            <Switch
              checked={filters.primaryOnly}
              onChange={(value) => updateFilter('primaryOnly', value)}
              className={`${
                filters.primaryOnly ? 'bg-orange-600' : 'bg-slate-200'
              } relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-orange-500 focus:ring-offset-2`}
            >
              <span
                className={`${
                  filters.primaryOnly ? 'translate-x-6' : 'translate-x-1'
                } inline-block h-4 w-4 transform rounded-full bg-white transition-transform`}
              />
            </Switch>
          </div>
        </Switch.Group>

        <Switch.Group>
          <div className="flex items-center justify-between">
            <Switch.Label className="text-sm font-medium text-slate-700">
              Hide tag spam
              <span className="block text-xs text-slate-500">Hide works with excessive tagging (15+ tags)</span>
            </Switch.Label>
            <Switch
              checked={filters.hideTagSpam}
              onChange={(value) => updateFilter('hideTagSpam', value)}
              className={`${
                filters.hideTagSpam ? 'bg-orange-600' : 'bg-slate-200'
              } relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-orange-500 focus:ring-offset-2`}
            >
              <span
                className={`${
                  filters.hideTagSpam ? 'translate-x-6' : 'translate-x-1'
                } inline-block h-4 w-4 transform rounded-full bg-white transition-transform`}
              />
            </Switch>
          </div>
        </Switch.Group>

        <Switch.Group>
          <div className="flex items-center justify-between">
            <Switch.Label className="text-sm font-medium text-slate-700">
              Show tag prominence
              <span className="block text-xs text-slate-500">Display primary/secondary/micro indicators</span>
            </Switch.Label>
            <Switch
              checked={filters.showProminence}
              onChange={(value) => updateFilter('showProminence', value)}
              className={`${
                filters.showProminence ? 'bg-orange-600' : 'bg-slate-200'
              } relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-orange-500 focus:ring-offset-2`}
            >
              <span
                className={`${
                  filters.showProminence ? 'translate-x-6' : 'translate-x-1'
                } inline-block h-4 w-4 transform rounded-full bg-white transition-transform`}
              />
            </Switch>
          </div>
        </Switch.Group>
      </div>

      {/* Advanced Toggle */}
      <div className="border-t border-slate-200 pt-4">
        <button
          onClick={() => setShowAdvanced(!showAdvanced)}
          className="flex items-center text-sm text-slate-600 hover:text-slate-800"
        >
          <svg
            className={`w-4 h-4 mr-2 transform transition-transform ${showAdvanced ? 'rotate-90' : ''}`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
          Advanced Filters
        </button>

        {showAdvanced && (
          <div className="mt-4 space-y-4">
            {/* Max Relationship Tags */}
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-2">
                Maximum relationship tags: {filters.maxRelationshipTags === 50 ? 'No limit' : filters.maxRelationshipTags}
              </label>
              <input
                type="range"
                min="1"
                max="50"
                value={filters.maxRelationshipTags}
                onChange={(e) => updateFilter('maxRelationshipTags', parseInt(e.target.value))}
                className="w-full h-2 bg-slate-200 rounded-lg appearance-none cursor-pointer"
              />
              <div className="flex justify-between text-xs text-slate-500 mt-1">
                <span>1</span>
                <span>10</span>
                <span>No limit</span>
              </div>
            </div>

            {/* Min Word Threshold */}
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-2">
                Minimum words per major tag: {filters.minWordThreshold === 0 ? 'No minimum' : `${filters.minWordThreshold} words`}
              </label>
              <input
                type="range"
                min="0"
                max="5000"
                step="500"
                value={filters.minWordThreshold}
                onChange={(e) => updateFilter('minWordThreshold', parseInt(e.target.value))}
                className="w-full h-2 bg-slate-200 rounded-lg appearance-none cursor-pointer"
              />
              <div className="flex justify-between text-xs text-slate-500 mt-1">
                <span>No min</span>
                <span>1k</span>
                <span>3k</span>
                <span>5k+</span>
              </div>
            </div>

            {/* Hide Collections */}
            <Switch.Group>
              <div className="flex items-center justify-between">
                <Switch.Label className="text-sm font-medium text-slate-700">
                  Hide drabble collections
                  <span className="block text-xs text-slate-500">Hide works with many micro tags (likely collections)</span>
                </Switch.Label>
                <Switch
                  checked={filters.hideCollections}
                  onChange={(value) => updateFilter('hideCollections', value)}
                  className={`${
                    filters.hideCollections ? 'bg-orange-600' : 'bg-slate-200'
                  } relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-orange-500 focus:ring-offset-2`}
                >
                  <span
                    className={`${
                      filters.hideCollections ? 'translate-x-6' : 'translate-x-1'
                    } inline-block h-4 w-4 transform rounded-full bg-white transition-transform`}
                  />
                </Switch>
              </div>
            </Switch.Group>
          </div>
        )}
      </div>

      {/* Filter Summary */}
      {(filters.primaryOnly || filters.hideTagSpam || filters.hideCollections) && (
        <div className="mt-4 p-3 bg-blue-50 border border-blue-200 rounded-lg">
          <h4 className="text-sm font-semibold text-blue-900 mb-1">Active Filters:</h4>
          <div className="text-xs text-blue-800 space-y-1">
            {filters.primaryOnly && <div>• Showing primary relationships only</div>}
            {filters.hideTagSpam && <div>• Hiding works with excessive tags</div>}
            {filters.hideCollections && <div>• Hiding drabble collections</div>}
            {filters.maxRelationshipTags < 50 && <div>• Max {filters.maxRelationshipTags} relationship tags</div>}
            {filters.minWordThreshold > 0 && <div>• Min {filters.minWordThreshold} words per tag</div>}
          </div>
        </div>
      )}

      {/* Reset Button */}
      {(filters.primaryOnly || filters.hideTagSpam || filters.hideCollections || filters.maxRelationshipTags < 50 || filters.minWordThreshold > 0) && (
        <button
          onClick={() => {
            const defaultFilters: TagFilters = {
              primaryOnly: false,
              hideTagSpam: false,
              maxRelationshipTags: 50,
              minWordThreshold: 0,
              hideCollections: false,
              showProminence: true,
            };
            setFilters(defaultFilters);
            onFilterChange(defaultFilters);
          }}
          className="mt-3 text-sm text-slate-600 hover:text-slate-800 underline"
        >
          Reset all filters
        </button>
      )}
    </div>
  );
}