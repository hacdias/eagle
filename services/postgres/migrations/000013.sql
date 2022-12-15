alter table entries
drop column audience,
drop column visibility,
add column isUnlisted boolean not null default false;
