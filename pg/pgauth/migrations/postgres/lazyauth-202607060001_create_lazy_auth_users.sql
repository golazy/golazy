-- +lazy Up
CREATE TABLE lazy_auth_users (
	id TEXT PRIMARY KEY,
	email TEXT NOT NULL,
	password_hash TEXT NOT NULL,
	data JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX lazy_auth_users_email_lower_idx
	ON lazy_auth_users (lower(email));

-- +lazy Down
DROP TABLE lazy_auth_users;
