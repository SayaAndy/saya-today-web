CREATE TABLE IF NOT EXISTS user_email_table (
    user_id VARCHAR(32) NOT NULL PRIMARY KEY,
    email VARCHAR(64) NOT NULL,
    lang VARCHAR(2) NOT NULL
) WITHOUT ROWID;

CREATE UNIQUE INDEX user_email_table_email_uindex
ON user_email_table(email);

CREATE TABLE IF NOT EXISTS subscription_user_to_tags_table (
    user_id VARCHAR(32) NOT NULL PRIMARY KEY,
    tags TEXT NOT NULL
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS subscription_tag_to_users_table (
    tag VARCHAR(32) NOT NULL PRIMARY KEY,
    user_ids TEXT NOT NULL
) WITHOUT ROWID;
