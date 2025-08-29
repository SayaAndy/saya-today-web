CREATE TABLE IF NOT EXISTS blog_likes (
    page_ref VARCHAR(32) NOT NULL,
    user_id VARCHAR(32) NOT NULL,
    PRIMARY KEY (page_ref, user_id)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS blog_views (
    page_ref VARCHAR(32) NOT NULL,
    user_id VARCHAR(32) NOT NULL,
    PRIMARY KEY (page_ref, user_id)
) WITHOUT ROWID;
