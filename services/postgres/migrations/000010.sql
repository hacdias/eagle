create table queue (
  id serial primary key,
  queue text not null,
  data text not null,
  attempt int default 1,
  created_at timestamp not null default now(),
  scheduled_at timestamp not null
);