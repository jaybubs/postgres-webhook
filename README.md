# Postgres Webhook

Keycloak is written in Java. We all hate java. However it stores event logs in postgres. We all tolerate sql.

This project is simply creating a webhook workaround - inject the postgres db with a function to send a json payload to a channel, and a trigger on insert/update/delete. Then the go app listens on the channel for these changes and http posts the payload to wherever we desire.

While the sql scripts are specific to keycloak, i've tried to keep them as generic sounding as possible fo ease of replication elsewhere.

The templating with structs is generic enough that you should only need to construct a new struct with whatever deets you need and have it ready. Additionally since a column might contain json but not be natively json (thanks keycloak), it is being cast into json here.
