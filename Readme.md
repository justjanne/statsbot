# KStats

## Setup

1. Create a PostgreSQL user and database for the statsbot
2. Apply the schema (see `schema.sql` to the database)
3. Add channels to the channels table
4. Start the bot

## Configuration

The bot takes several environment variables for configuration.  
Required variables are marked with *

| Name                     | Example                                        | Description                                            |
| ------------------------ | ---------------------------------------------- | ------------------------------------------------------ |
|`KSTATS_IRC_SERVER`*      | `irc.rizon.net`                                | Irc Servername                                         |
|`KSTATS_IRC_PORT`*        | `6697`                                         | Irc Port                                               |
|`KSTATS_IRC_SECURE`       | `true`                                         | If TLS should be enabled for this connection           |
|`KSTATS_IRC_NICK`         |`kstats`                                        | Nickname visible on IRC                                |
|`KSTATS_IRC_IDENT`        |`kstats`                                        | Ident visible on IRC                                   |
|`KSTATS_IRC_REALNAME`     |`KStats kuschku.de Statistics Bot`              | Realname visible on IRC                                |
|`KSTATS_IRC_SASL_ENABLED` |`true`                                          | If SASL is enabled                                     |
|`KSTATS_IRC_SASL_ACCOUNT` |`kstats`                                        | SASL Username                                          |
|`KSTATS_IRC_SASL_PASSWORD`|`caa5db269fc39`                                 | SASL Password                                          |
|`KSTATS_DATABASE_TYPE`*   |`postgres`                                      | Database driver (only postgres is supported currently) |
|`KSTATS_DATABASE_URL`*    |`postgresql://kstats:hunter2@localhost/statsbot`| Database URL                                           |
