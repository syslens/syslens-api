-- PostgreSQL Setup Script for SysLens
-- Creates the database user and database.
-- Execute this script as a PostgreSQL superuser (e.g., 'postgres').

-- 1. Create the database user
-- IMPORTANT: Replace 'a_very_secure_password_here' with a strong, unique password!
CREATE USER syslens_user WITH PASSWORD 'a_very_secure_password_here';

-- Grant login permission (usually default)
ALTER USER syslens_user WITH LOGIN;


-- 2. Create the database
CREATE DATABASE syslens OWNER syslens_user;


-- 3. Grant privileges to the user on the database
GRANT ALL PRIVILEGES ON DATABASE syslens TO syslens_user;


-- Optional: Connect to the new database and verify permissions (if running interactively)
-- \c syslens syslens_user
-- \dt  -- Should show no tables initially, but confirms connection and basic rights


-- End of setup script  