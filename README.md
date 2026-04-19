# Wormhole — минималистичный UI для magic-wormhole

Кроссплатформенный интерфейс для протокола [wormhole-william](https://github.com/psanford/wormhole-william) (совместим с magic-wormhole).

## Архитектура

```
Windows app (Go)  ←──FCM/polling──→  Backend relay (FastAPI)  ←──FCM──→  Android app (Kotlin)
     │                                         │
     └──── wormhole-william (Go library) ──────┘
                  (прямой P2P-перенос файла)
```

- **Windows**: ~12 МБ ОЗУ в простое, только иконка в трее
- **Android**: 0 МБ в простое (FCM будит приложение при необходимости)
- **Протокол**: end-to-end зашифрованный P2P (wormhole-william)

---

## Быстрый старт

### 1. Firebase

1. Создайте проект в [Firebase Console](https://console.firebase.google.com/)
2. Включите **Authentication → Google**
3. Добавьте Android-приложение (`com.wormhole`), скачайте `google-services.json` → `android/app/`
4. Создайте сервис-аккаунт: **Project settings → Service accounts → Generate new private key** → сохраните как `backend/serviceAccount.json`

### 2. Backend

```bash
cd backend
pip install -r requirements.txt
export GOOGLE_APPLICATION_CREDENTIALS=serviceAccount.json
uvicorn main:app --host 0.0.0.0 --port 8000
```

Для деплоя на Railway: `railway up` (railway.toml уже настроен).

Замените `RELAY_URL` в:
- `windows/relay.go` → константа `RELAY_URL` (пока `cfg.RelayURL`)
- `android/app/src/main/java/com/wormhole/RelayClient.kt` → константа `RELAY_URL`

### 3. Windows

1. Замените константы в `windows/auth.go`:
   - `firebaseAPIKey` — из Firebase Console → Project settings → Web API key
   - `googleClientID` — из Google Cloud Console → OAuth 2.0 client (Desktop app)
2. Постройте и установите:
   ```bat
   cd windows
   go mod tidy
   build.bat
   wormhole-windows-amd64.exe --install
   ```
3. Запустите `wormhole-windows-amd64.exe` — войдёт браузер для авторизации Google, затем появится иконка в трее.

#### Использование (Windows)
- **Отправить**: правой кнопкой на файле → **Отправить → Wormhole**, либо иконка в трее → **Отправить файл…**
- **Получить**: всплывёт уведомление с кнопками **Принять** / **Отклонить**

### 4. Android

1. Скопируйте `android/app/google-services.json.template` → `android/app/google-services.json` и заполните реальными значениями из Firebase.
2. Постройте нативную библиотеку:
   ```bash
   cd native
   go mod tidy
   bash build.sh   # → android/app/libs/wormhole.aar
   ```
3. Постройте APK:
   ```bash
   cd android
   ./gradlew assembleRelease
   ```
4. Установите APK, откройте приложение → войдите через Google.

#### Использование (Android)
- **Отправить**: в любом приложении → **Поделиться** → **Wormhole**
- **Получить**: появится уведомление **Принять / Отклонить**; после приёма файл сохраняется в **Загрузки**

---

## Структура проекта

```
backend/      FastAPI relay-сервер (уведомления через FCM)
native/       Go-библиотека wormhole-william для gomobile (Android AAR)
windows/      Go приложение для Windows (системный трей, без окон)
android/      Kotlin Android-приложение
```

---

## Ресурсы в простое

| Компонент | ОЗУ | CPU |
|---|---|---|
| Windows (трей) | ~12 МБ | ~0% |
| Android | 0 МБ | 0% |
| Backend | ~20 МБ | ~0% |
