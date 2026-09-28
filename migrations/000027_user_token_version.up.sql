-- token_version backs "log out everywhere": every issued JWT carries the
-- token_version that was current when it was minted, and every authenticated
-- request checks it against the user's current value. Bumping this column
-- immediately invalidates every token issued before the bump, without
-- needing a server-side session store or token blocklist.
ALTER TABLE users ADD COLUMN token_version INT NOT NULL DEFAULT 1;
