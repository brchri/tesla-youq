#!/bin/bash

# if non-root user specified at runtime with user flag, just execute CMD and exit
if [ $(id -u) -ne 0 ]; then
  exec "$@"
  return
fi

# OUID and OGID are the original user and group ids set during the image
# build and are replaced here at runtime if PGID and PUID are set
if [ -n "$PGID" ] && [ "$PGID" -ne 0 ] && [ "$PGID" -ne "$OGID" ] ; then
  # if group id doesn't already exist, set nonroot gid = $PGID, otherwise just use existing group name for $PGID
  if ! grep ":x:$PGID:" /etc/group 2>&1 >/dev/null; then
    sed -i "s/nonroot:x:$OUID:$OGID:/nonroot:x:$OUID:$PGID:/" /etc/passwd
    sed -i "s/nonroot:x:$OGID:/nonroot:x:$PGID:/" /etc/group
  fi
fi

if [ -n "$PUID" ] && [ "$PUID" -ne 0 ] && [ "$PUID" -ne "$OUID" ]; then
  # if user id doesn't already exist, set nonroot uid = $PUID, otherwise just use existing username for $PUID
  if ! grep ":x:$PUID:" /etc/passwd 2>&1 >/dev/null; then
    sed -i "s/nonroot:x:$OUID:/nonroot:x:$PUID:/" /etc/passwd
  fi
fi

chown $PUID:$PGID /app /app/*

# Use su-exec to execute the command as nonroot user
exec su-exec $PUID:$PGID "$@"
