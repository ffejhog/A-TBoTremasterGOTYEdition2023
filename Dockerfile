FROM alpine:latest
WORKDIR /app

RUN apk add --no-cache libc6-compat

COPY build/ .

RUN chmod +x ./DBot

ENTRYPOINT ["./DBot"]