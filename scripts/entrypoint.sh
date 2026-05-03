#!/bin/bash
# SSH forced-command script for ZhatRoom
# Called from authorized_keys: command="/opt/zhatroom/entrypoint.sh <username>",restrict ...
#
# Validates the user exists in the database, then launches the chat client.

set -e

USERNAME="$1"

if [ -z "$USERNAME" ]; then
    echo "Usage: entrypoint.sh <username>"
    exit 1
fi

# Validate user exists in PostgreSQL
EXISTS=$(psql -h 127.0.0.1 -U postgres -d zhat_db -t -A \
    -c "SELECT COUNT(*) FROM users WHERE id = '$USERNAME';" 2>/dev/null)

if [ "$EXISTS" != "1" ]; then
    echo "Access denied: user '$USERNAME' not registered."
    echo "Contact the server admin to get an account."
    exit 1
fi

exec /opt/zhatroom/bin/client --id "$USERNAME" --usr "$USERNAME" --socket /tmp/zhatroom.sock
