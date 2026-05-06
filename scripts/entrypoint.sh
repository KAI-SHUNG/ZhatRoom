#!/bin/bash
# SSH forced-command script for ZhatRoom
# Called from authorized_keys: command="/opt/zhatroom/entrypoint.sh <uid> <username>",restrict ...
#
# Validates the user exists in the database, then launches the chat client.

set -e

USER_ID="$1"
USERNAME="$2"

if [ -z "$USER_ID" ] || [ -z "$USERNAME" ]; then
    echo "Usage: entrypoint.sh <uid> <username>"
    exit 1
fi

# Validate user exists in PostgreSQL
EXISTS=$(psql -h 127.0.0.1 -U postgres -d zhat_db -t -A \
    -c "SELECT COUNT(*) FROM users WHERE id = '$USER_ID';" 2>/dev/null)

if [ "$EXISTS" != "1" ]; then
    echo "Access denied: user not registered."
    echo "Contact the server admin to get an account."
    exit 1
fi

exec /opt/zhatroom/bin/client --id "$USER_ID" --usr "$USERNAME" --socket /tmp/zhatroom.sock
