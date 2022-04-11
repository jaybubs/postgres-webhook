# Postgres Webhook

Keycloak is written in Java. We all hate java. However it stores event logs in postgres. We all tolerate sql.

This project is simply creating a webhook workaround - inject the postgres db with a function to send a json payload to a channel, and a trigger on insert/update/delete. Then the go app listens on the channel for these changes and http posts the payload to wherever we desire.
