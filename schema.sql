create table channels (
  id      serial not null
    constraint channels_pkey primary key,
  channel text   not null,
  salt    text   not null
);

create unique index channels_channel_uindex
  on channels (channel);

create table messages (
  id          serial not null
    constraint messages_pkey primary key,
  time        timestamp,
  channel     integer
    constraint messages_channels_id_fk references channels on update cascade on delete cascade,
  sender      text,
  words       integer,
  characters  integer,
  question    boolean,
  exclamation boolean,
  caps        boolean,
  aggression  boolean,
  emoji_happy boolean,
  emoji_sad   boolean
);

create index messages_channel_index
  on messages (channel);
create index messages_channel_sender_index
  on messages (channel, sender);

create table users (
  hash text not null
    constraint users_pkey primary key,
  nick text
);

create table "references" (
  id      serial not null
    constraint references_pkey primary key,
  channel integer
    constraint references_channels_id_fk references channels on update cascade on delete cascade,
  time    timestamp,
  source  text,
  target  text
);