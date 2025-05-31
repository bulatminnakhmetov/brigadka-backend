-- Add email_verified field to users table
ALTER TABLE users 
ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE;