FROM migrate/migrate:v4.18.1 AS migrations

COPY migrations /migrations
ENTRYPOINT ["migrate", "-path", "/migrations"]
