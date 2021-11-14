drop table if exists sections;
drop table if exists tags;
drop table if exists entries;

create table entries (
  id text primary key,
  content text not null,
  isDraft boolean not null,
  isDeleted boolean not null,
  isPrivate boolean not null,
  date TIMESTAMP WITH TIME ZONE,
  properties json,
  ts tsvector generated always as (to_tsvector('english', content)) STORED
);

create index ts_idx on entries using gin (ts);

create table tags (
	entry_id text not null references entries(id) on delete cascade,
	tag text not null
);

create unique index entry_tag on tags (entry_id, tag);
create index entry_tags on tags (tag);

create table sections (
	entry_id text not null references entries(id) on delete cascade,
	section text not null
);

create unique index entry_section on sections (entry_id, section);
create index entry_sections on sections (section);
