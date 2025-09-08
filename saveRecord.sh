#!/bin/bash

HOST="kryuchenko.org"
USER="storage"
PASS="storage123"

FILE=$(ls -t recordings_* 2>/dev/null | head -n1)

if [ ! -f "$FILE" ]; then
    echo "Error: File '$FILE' not found!"
    exit 1
fi  

echo "Find file: $FILE"

echo -e "cd files\nput $FILE\nls -l\nexit" | sshpass -p "$PASS" sftp $USER@$HOST

echo "Done!"
