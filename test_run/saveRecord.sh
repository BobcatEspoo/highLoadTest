#!/bin/bash

HOST="kryuchenko.org"
USER="storage"
PASS="storage123"
mkdir -p ~/.ssh
chmod 700 ~/.ssh
ssh-keyscan -H $HOST >> ~/.ssh/known_hosts
chmod 644 ~/.ssh/known_hosts
FILE=$(ls -t recordings_* 2>/dev/null | head -n1)

if [ ! -f "$FILE" ]; then
    echo "Error: File '$FILE' not found!"
    exit 1
fi

echo "Find file: $FILE"

echo -e "cd files\nput $FILE\nls -l\nexit" | sshpass -p "$PASS" sftp -o StrictHostKeyChecking=no $USER@$HOST


echo "Done!"