alter table activitypub_followers
add column name text,
add column handle text not null;
