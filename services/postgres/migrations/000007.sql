drop index entry_tag;
drop index entry_tags;

drop index entry_emoji;
drop index entry_emojis;

drop table tags;
drop table emojis;

create table taxonomies (
  entry_id text not null references entries(id) on delete cascade,
  taxonomy text not null,
  term text not null
);

create index taxonomies_terms on taxonomies (taxonomy, term);
create unique index taxonomies_entries on taxonomies (entry_id, taxonomy, term);
