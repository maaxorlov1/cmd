FROM alpine
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates
WORKDIR /app
COPY . /app
CMD /app/main.exe