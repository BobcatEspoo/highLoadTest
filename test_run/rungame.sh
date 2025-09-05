#!/bin/bash

HOST="kryuchenko.org"
USER="storage"
PASS="storage123"

echo -e "cd files\nput photo.bmp\nls -l\nexit" | sshpass -p "$PASS" sftp $USER@$HOST

echo "Готово!"
