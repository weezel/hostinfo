#!/usr/bin/env bash

# XXX This is only to be used with Unix socket listener

set -euo pipefail

# Ensure that this directory only has www-data:hostinfo-username
# access rights and 770 permissions.
readonly _socket_path="/var/www/htdocs/sockets/hostinfo.sock"


echo "Starting the binary..."
./hostinfo -u "${_socket_path}" &
BINARY_PID=$!

echo "Waiting for the file to be created..."
while ! ls "${_socket_path}" >/dev/null 2>&1; do
        sleep 1.0
done

echo "Sending the process to the background..."
kill -STOP $BINARY_PID

echo "Changing file ownership..."
sudo chown www-data:weezel "${_socket_path}"

echo "Returning the binary to the foreground..."
kill -CONT $BINARY_PID
wait $BINARY_PID
