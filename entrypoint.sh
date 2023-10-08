#!/bin/bash

# OUID and OGID are the original user and group ids set during the image
# build and are replaced here at runtime if PGID and PUID are set
if [ -n "$PGID" ] && [ "$PGID" -ne 0 ]; then
  sed -i "s/nonroot:x:$OUID:$OGID:/nonroot:x:$OUID:$PGID:/" /etc/passwd
  sed -i "s/nonroot:x:$OGID:/nonroot:x:$PGID:/" /etc/group
fi

if [ -n "$PUID" ] && [ "$PUID" -ne 0 ]; then
  sed -i "s/nonroot:x:$OUID:/nonroot:x:$PUID:/" /etc/passwd
fi

chown nonroot: /app /app/*

# Use su-exec to execute the command as nonroot user
exec su-exec nonroot "$@"
