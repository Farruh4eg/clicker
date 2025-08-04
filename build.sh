#!/bin/bash

# --- Начало скрипта и настройка окружения ---

# Прекратить выполнение при любой ошибке
set -e

# Получить директорию, в которой находится скрипт (корень проекта)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
cd "$SCRIPT_DIR"

echo "Запуск процесса сборки в $(pwd)..."
echo "ВАЖНО: Убедитесь, что у вас есть необходимые права для выполнения операций сборки."
echo

# --- Установка GOPROXY для надежной загрузки модулей ---
echo "Установка GOPROXY=direct"
export GOPROXY=direct
echo

# --- Диагностика: Проверка наличия GCC ---
echo "ДИАГНОСТИКА: Проверка наличия GCC в PATH..."
if ! command -v gcc &>/dev/null; then
  echo "ПРЕДУПРЕЖДЕНИЕ: gcc не найден. Сборка сервера с CGO может завершиться ошибкой."
else
  command -v gcc
fi

# --- Шаг 1: Генерация кода protobuf ---
echo "[1/5] Запуск buf generate..."
if ! buf generate; then
  echo "ОШИБКА: выполнение buf generate завершилось с ошибкой!"
  exit 1
fi
echo "buf generate успешно завершен."
echo

# --- Шаг 2: Сборка сервера ---
echo "[2/5] Сборка сервера..."
if [ ! -d "cmd/server" ]; then
  echo "ОШИБКА: директория 'cmd/server' не найдена."
  exit 1
fi
(
  cd cmd/server
  echo "Включение CGO для сборки сервера (CGO_ENABLED=1) и использование тега cgo"
  export CGO_ENABLED=1
  echo "Сборка сервера с флагами (-v -x)..."
  if ! go build -v -x -tags cgo -ldflags="-extldflags=-static -s -w" -o ../../server .; then
    echo "ОШИБКА: Сборка сервера завершилась с ошибкой!"
    exit 1
  fi
)
echo "Сборка сервера успешно завершена. Исполняемый файл 'server' находится в корне проекта."
echo

# --- Шаг 3: Сборка клиента ---
echo "[3/5] Сборка клиента..."
if [ ! -d "cmd/client" ]; then
  echo "ОШИБКА: директория 'cmd/client' не найдена."
  exit 1
fi
(
  cd cmd/client
  if ! go build -o ../../client -ldflags="-s -w" .; then
    echo "ОШИБКА: Сборка клиента завершилась с ошибкой!"
    exit 1
  fi
)
echo "Сборка клиента успешно завершена. Исполняемый файл 'client' находится в корне проекта."
echo

echo "Процесс сборки успешно завершен!"
echo "Финальные исполняемые файлы (server, client) находятся в $(pwd)"

# Очистка переменных окружения не требуется так же, как в `setlocal`/`endlocal`,
# поскольку изменения `export` действуют только в текущем сеансе оболочки и его дочерних процессах.
