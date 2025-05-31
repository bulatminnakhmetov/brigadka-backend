-- Remove email_verified field from users table
ALTER TABLE users 
DROP COLUMN email_verified;