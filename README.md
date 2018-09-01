# Coding Guru

Coding Guru is a Telegram Bot that answers all of your coding questions.

App can be built and started using docker:

`docker build -t guru:latest && docker run --name guru-instance -e TELEGRAM_KEY_CODING_GURU_BOT=${apikey} guru:latest`

Env vars:
- `TELEGRAM_KEY_CODING_GURU_BOT` - telegram API key
- `DEV_MODE_CODING_GURU_BOT` - dev mode (use polling instead of web hook) 
- `WEB_HOOK_HOST_CODING_GURU_BOT` - host string for web hook
- `WEB_HOOK_LISTEN_PORT_CODING_GURU_BOT` - listen port for web hook
- `USE_TLS_ENCRYPTION_CODING_GURU_BOT` - use TLS encryption for web hook
- `TLS_CERT_FILE_CODING_GURU_BOT` - TLS cert file
- `TLS_KEY_FILE_CODING_GURU_BOT` - TLS cert key file

Bot is available here: https://t.me/codinggurubot