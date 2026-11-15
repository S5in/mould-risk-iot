-- Users who can sign in to the API.
-- Each user belongs to exactly one organisation, and the API takes the
-- organisation from here (through the login token), never from the request.
CREATE TABLE IF NOT EXISTS users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        uuid NOT NULL REFERENCES organisations(id),
    email         text NOT NULL,
    password_hash text NOT NULL,  -- bcrypt hash, never the password itself
    role          text NOT NULL DEFAULT 'viewer' CHECK (role IN ('admin', 'viewer')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_email_has_at CHECK (position('@' in email) > 1)
);

-- Emails are unique ignoring upper/lower case: A@x.com and a@x.com are the same user.
CREATE UNIQUE INDEX IF NOT EXISTS users_email_lower_idx ON users (lower(email));
CREATE INDEX IF NOT EXISTS users_org_id_idx ON users (org_id);