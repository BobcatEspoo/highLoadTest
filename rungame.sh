#!/bin/bash

set -e

# Проверяем, установлен ли golang
if ! command -v go &> /dev/null
then
    echo "Go не установлен. Начинаем установку..."

    # Получаем URL последнего релиза Go для linux amd64
    GO_URL=$(curl -s https://go.dev/dl/ | grep linux-amd64.tar.gz | head -1 | grep -o 'https://.*/go[0-9.]*.linux-amd64.tar.gz')

    echo "Скачиваем $GO_URL"
    wget -q "$GO_URL" -O /tmp/go.tar.gz

    # Удаляем старую версию (если есть)
    sudo rm -rf /usr/local/go

    # Распаковываем
    sudo tar -C /usr/local -xzf /tmp/go.tar.gz

    # Добавляем Go в PATH если не добавлено
    if ! grep -q 'export PATH=$PATH:/usr/local/go/bin' ~/.profile; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
    fi

    # Обновляем PATH для текущей сессии
    export PATH=$PATH:/usr/local/go/bin

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
