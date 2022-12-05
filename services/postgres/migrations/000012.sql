alter table entries drop column properties;
alter table entries rename column date to published_at;
alter table entries rename column updated to updated_at;
