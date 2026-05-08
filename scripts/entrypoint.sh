#!/bin/bash
# ZhatRoom SSH entrypoint
# Two ways to run:
#   1. Via authorized_keys command= (primary): entrypoint.sh <uid> <username>
#   2. Via shell fallback (chat user's shell is set to this script)
#
# When the client exits, the shell exits, SSH disconnects immediately.

# Prevent Ctrl+C / Ctrl+Z escape — exit the session cleanly
trap 'exit 0' INT TERM
trap '' TSTP

USER_ID="$1"
USERNAME="$2"

if [ -z "$USER_ID" ] || [ -z "$USERNAME" ]; then
    echo "Access denied."
    exit 1
fi

# Validate user exists in PostgreSQL (via Docker container)
EXISTS=$(docker exec zhat_db psql -U postgres -d zhat_db -t -A \
    -c "SELECT COUNT(*) FROM users WHERE id = '$USER_ID';" 2>/dev/null)

if [ "$EXISTS" != "1" ]; then
    echo "Access denied: user not registered."
    echo "Contact the server admin to get an account."
    exit 1
fi

exec /opt/zhatroom/bin/client --id "$USER_ID" --usr "$USERNAME" --socket /tmp/zhatroom.sock
