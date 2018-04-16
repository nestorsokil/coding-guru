FROM scratch
ADD deploy /app
ADD deploy/ca-certificates.crt /etc/ssl/certs/
CMD ["app/coding-guru"]