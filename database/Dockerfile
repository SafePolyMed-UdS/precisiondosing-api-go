FROM mysql:9

RUN mkdir /setup
COPY schemas/sql/* /docker-entrypoint-initdb.d/
