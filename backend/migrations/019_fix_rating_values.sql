-- Migration 019: Fix rating values to use snake_case instead of display strings
-- This is a better API design pattern

-- First, update any existing data to use snake_case values
UPDATE works SET rating = 'not_rated' WHERE rating = 'Not Rated';
UPDATE works SET rating = 'general' WHERE rating = 'General Audiences';
UPDATE works SET rating = 'teen' WHERE rating = 'Teen And Up Audiences';
UPDATE works SET rating = 'mature' WHERE rating = 'Mature';
UPDATE works SET rating = 'explicit' WHERE rating = 'Explicit';

-- Drop the old constraint
ALTER TABLE works DROP CONSTRAINT rating_values;

-- Add the new constraint with snake_case values
ALTER TABLE works ADD CONSTRAINT rating_values 
CHECK (rating IN ('not_rated', 'general', 'teen', 'mature', 'explicit'));

-- Update any other tables that might reference ratings
-- (Add similar updates for other tables if they exist)