# Wormhole

Минималистичный кроссплатформенный UI для [wormhole-william](https://github.com/psanford/wormhole-william), совместимого с `magic-wormhole`.

Проект состоит из трёх частей:
- `windows/` — приложение для Windows на Go, которое живёт в системном трее
- `android/` — Android-приложение на Kotlin, которое принимает шаринг и показывает уведомления
- `backend/` — relay-сервер на FastAPI, который хранит устройства пользователя и раздаёт сигналы о входящих передачах

Сам файл не ходит через backend. Backend нужен только для доставки сигнала "у меня есть wormhole code, забери файл".

## Как это работает

### Общая схема

```text
Windows app ── register/poll ──┐
                               │
Android app ─ register/FCM ────┼── Backend relay ─── уведомление о входящем файле
                               │
Wormhole sender/receiver ──────┘

Фактическая передача файла:
sender ── magic-wormhole / wormhole-william ── receiver
```

### Что делает backend

Backend не проксирует файл и не участвует в содержимом передачи.

Он делает только служебную работу:
- хранит список устройств пользователя `uid + device_id`
- сохраняет FCM token Android-устройств
- создаёт `pending_codes` для каждого целевого устройства
- шлёт FCM на Android
- отдаёт pending-коды Windows-клиенту через polling
- принимает `/ack`, когда код обработан или отклонён

Основные endpoints:
- `POST /register` — устройство сообщает `uid`, `device_id`, `platform`, `fcm_token`
- `POST /notify` — отправитель сообщает `code` и `filename`
- `GET /poll/{uid}/{device_id}` — Windows опрашивает pending-коды
- `POST /ack` — устройство подтверждает обработку кода

## Поток работы приложения

### 1. Первый запуск и авторизация

#### Android

При первом запуске пользователь входит через Google.

После успешного входа приложение:
- получает Firebase user
- получает FCM token
- генерирует и сохраняет локальный `device_id`
- вызывает `/register` на backend

После этого Android-устройство известно relay-серверу и может получать входящие сигналы.

#### Windows

Windows-приложение при старте:
- поднимает tray icon
- читает локальный конфиг и креды из пользовательской директории
- при необходимости запускает Google OAuth в браузере
- меняет Google OAuth token на Firebase token
- сохраняет сессию на диск
- регистрирует устройство на backend
- запускает polling

Сессия сохраняется локально, поэтому повторный вход нужен только если refresh-token протух или стал невалидным.

### 2. Отправка файла

#### Android → другие устройства

Когда пользователь шарит файл в Android:
- `ShareActivity` получает `ACTION_SEND`
- копирует файл во временный cache
- вызывает `WormholeLib.sendFile(...)`
- как только появляется wormhole code, вызывает `POST /notify`
- backend создаёт `pending_codes` для других устройств этого же `uid`
- Android-устройствам отправляется FCM
- Windows увидит pending-код на следующем poll

Сам файл в этот момент уже готов к получению через wormhole.

#### Windows → другие устройства

Когда пользователь отправляет файл с Windows:
- либо через `Send to -> Wormhole`
- либо через tray menu и встроенный локальный sender UI

Приложение:
- открывает файл
- запускает `wormhole-william`
- получает код
- вызывает `POST /notify`
- показывает toast с кодом и статусом

### 3. Получение файла

#### Android

Android получает FCM data message:
- `FcmService` читает `code`, `filename`, `code_id`
- показывает notification с действиями `Принять` / `Отклонить`

Если пользователь нажимает `Принять`:
- запускается `ReceiveService`
- он вызывает `WormholeLib.receiveFile(...)`
- файл сохраняется в `Downloads`
- backend получает `/ack`

Если пользователь нажимает `Отклонить`:
- код помечается как обработанный через `/ack`

#### Windows

Windows не использует FCM. Вместо этого он опрашивает backend каждые 30 секунд:
- `GET /poll/{uid}/{device_id}`
- если есть pending-коды, показывает toast
- при `Принять` запускает получение файла
- при `Отклонить` вызывает `/ack`

## Что где хранится

### Windows

Windows хранит локальное состояние в пользовательской директории:
- `%APPDATA%\Wormhole\config.json`
- `%APPDATA%\Wormhole\credentials.json`

Там лежат:
- `uid`
- `device_id`
- `relay_url`
- `refresh_token`
- текущий `id_token`

Это позволяет не логиниться заново при каждом запуске.

### Android

Android хранит:
- `device_id` в `SharedPreferences`
- Firebase session внутри Firebase Auth

Полученные файлы сохраняются в `Downloads`.

### Backend

Backend хранит SQLite-базу:
- `devices` — устройства пользователя
- `pending_codes` — ожидающие обработки wormhole-коды

## Почему нужен и backend, и wormhole

`wormhole-william` решает только передачу файла между sender и receiver.
Но сам по себе wormhole не знает:
- на какие устройства пользователя слать сигнал
- как разбудить Android в фоне
- как показать "входящий файл" без ручного ввода кода

Эту часть решает relay:
- связывает устройства одного пользователя
- доставляет служебное уведомление
- даёт Android wake-up через FCM
- даёт Windows pending-queue через polling

## Что обязательно должно быть настроено

### Firebase / Google Cloud

Для Android:
- Firebase project
- Android app `com.wormhole`
- корректный `google-services.json`
- включённый Google sign-in
- добавленные SHA-1 и SHA-256 сертификаты Android app
- runtime permission на notifications на Android 13+

Для Windows:
- Firebase Web API key
- Google OAuth client для Windows
- рабочий browser OAuth flow

Для backend:
- service account JSON для Firebase Admin SDK

### Relay URL

И Android, и Windows должны смотреть в один и тот же backend relay.

Сейчас код по умолчанию использует:
- Android: `android/app/src/main/java/com/wormhole/RelayClient.kt`
- Windows: значение `RelayURL` в локальном config, по умолчанию `https://wormhole.ibotz.fun`

## Быстрый старт

### 1. Поднять backend

```bash
cd backend
pip install -r requirements.txt
export GOOGLE_APPLICATION_CREDENTIALS=serviceAccount.json
uvicorn main:app --host 0.0.0.0 --port 8000
```

Для Railway можно использовать `railway up`.

### 2. Настроить Android

1. Добавить Android app `com.wormhole` в Firebase
2. Включить Google sign-in
3. Добавить SHA-1 и SHA-256
4. Положить актуальный `google-services.json` в `android/app/`
5. Собрать AAR:

```bash
cd native
go mod tidy
bash build.sh
```

6. Собрать APK:

```bash
cd android
./gradlew assembleRelease
```

7. Установить APK
8. Открыть приложение
9. Войти через Google
10. Разрешить уведомления

### 3. Настроить Windows

1. Скопировать `windows/auth.secrets.json.template` в `windows/auth.secrets.json`
2. Заполнить в `windows/auth.secrets.json` реальные значения:
   - `firebase_api_key`
   - `google_client_id`
   - при необходимости `google_client_secret`
3. Запустить `build.bat` — скрипт временно генерирует локальный Go-файл, вшивает значения в бинарник и удаляет временный файл после сборки
4. Собрать приложение:

```bat
cd windows
go mod tidy
build.bat
wormhole-windows-amd64.exe --install
```

5. Запустить `wormhole-windows-amd64.exe`
6. Выполнить вход через браузер
7. Убедиться, что приложение появилось в трее

## Поведение по платформам

### Windows

- работает как tray app
- имеет sender UI в браузере для drag-and-drop
- умеет отправлять через `SendTo`
- получает сигналы через polling, а не FCM
- хранит сессию локально

### Android

- имеет минимальный launcher screen
- умеет принимать файл через share sheet
- просыпается через FCM
- показывает notification для входящего файла
- получает файл в foreground service только в момент передачи

## Структура проекта

```text
backend/   FastAPI relay + SQLite + Firebase Admin
native/    Go binding для wormhole-william, собираемый в Android AAR
windows/   Windows tray app на Go
android/   Android app на Kotlin
```

## Ограничения и типичные проблемы

### Нет уведомлений на Android

Проверьте:
- выдан ли `POST_NOTIFICATIONS`
- зарегистрировалось ли устройство на backend
- актуален ли FCM token
- совпадает ли `uid` на обоих устройствах

### Windows не получает ничего

Проверьте:
- прошёл ли Windows OAuth login
- зарегистрировалось ли Windows-устройство на backend
- не сломан ли relay URL
- не завис ли polling

### Android Google Sign-In выдаёт `DEVELOPER_ERROR (10)`

Обычно это значит:
- нет Web OAuth client ID в конфиге
- не добавлены SHA-1 / SHA-256
- stale `google-services.json`

## Ресурсы в простое

| Компонент | ОЗУ | CPU |
|---|---|---|
| Windows tray app | ~12 МБ | ~0% |
| Android | почти 0 МБ вне активной передачи | ~0% |
| Backend relay | ~20 МБ | ~0% |
