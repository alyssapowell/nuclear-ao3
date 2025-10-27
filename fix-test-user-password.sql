-- Fix the test user password hash
-- This updates the test user to have the correct hash for "password123"

UPDATE users 
SET password_hash = '$2a$10$8gOj1DJjiSl5JuloUEDuzOUzg00tkq8c9V6.VQVK7laoeUycdNR9e'
WHERE email = 'test@nuclear-ao3.com';

-- Also update all other test users to have the same working password
UPDATE users 
SET password_hash = '$2a$10$8gOj1DJjiSl5JuloUEDuzOUzg00tkq8c9V6.VQVK7laoeUycdNR9e'
WHERE email IN ('admin@nuclear-ao3.com', 'author2@nuclear-ao3.com', 'tags@nuclear-ao3.com');

-- Verify the update
SELECT email, username, 
       CASE 
         WHEN password_hash = '$2a$10$8gOj1DJjiSl5JuloUEDuzOUzg00tkq8c9V6.VQVK7laoeUycdNR9e' 
         THEN '✅ Correct hash' 
         ELSE '❌ Wrong hash' 
       END as password_status
FROM users 
WHERE email LIKE '%@nuclear-ao3.com';