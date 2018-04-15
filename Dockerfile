FROM scratch
ADD deploy /app
ADD deploy/ca-certificates.crt /etc/ssl/certs/
CMD ["app/coding-guru"]

# docker run --name codeguru -e TELEGRAM_KEY_CODING_GURU_BOT=${apikey} -e DEV_MODE_CODING_GURU_BOT=true coding-guru:latest