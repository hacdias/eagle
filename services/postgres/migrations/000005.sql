create table emojis (
	entry_id text not null references entries(id) on delete cascade,
	emoji text not null
);

create unique index entry_emoji on emojis (entry_id, emoji);
create index entry_emojis on emojis (emoji);
