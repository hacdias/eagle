create table activitypub_links (
  entry_id text not null,
  object_id text not null
);

create unique index activitypub_links_unique on activitypub_links (entry_id, object_id);

create table activitypub_followers (
  iri text primary key,
  inbox text not null
);
