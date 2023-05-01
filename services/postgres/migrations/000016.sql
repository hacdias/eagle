create table guestbook_entries (
  id serial primary key,
  name text,
  website text,
  content text not null,
  date timestamp with time zone not null
);
