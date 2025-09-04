#!/bin/bash

set -e

# Проверяем, установлен ли golang
if ! command -v go &> /dev/null
then
    echo "Go не установлен. Начинаем установку..."

    rm -rf /usr/local/go && tar -C /usr/local -xzf go1.24.4.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    go version
    echo "Go установлен."
else
    echo "Go уже установлен."
fi

# Компилируем main.go
echo "Компилируем main.go..."
go build -o main main.go

# Делаем бинарник исполняемым и запускаем
chmod +x ./main
echo "Запускаем ./main..."
./main
