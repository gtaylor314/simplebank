#!/bin/sh
# start.sh will be run by /bin/sh since we are using an alpine-3.16 image - shebang (#!) required as it tells the kernel
# how to run start.sh

# script will exit immediately if any command returns a non-zero status
set -e 

echo "run db migration"
# call migrate binary with the path to the migration files, the database URL, verbose logging, and up command (migrate up)
# use $DB_SOURCE for the database URL - it will pull from the compose.yaml file
/app/migrate -path /app/migration -database "$DB_SOURCE" -verbose up

echo "start the app"
# $@ = take all parameters passed to script and run it which should be /app/main from the Dockerfile
# the CMD in the Dockerfile is passed to the Entrypoint in the Dockerfile
exec "$@"
