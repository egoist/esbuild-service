FROM golang:1.14 AS build
WORKDIR /mnt
COPY . .
RUN CGO_ENABLED=0 go build -o ./bin/esbuild-service main.go

FROM node:14.5.0-alpine3.12
WORKDIR /opt
RUN apk add --no-cache ca-certificates
COPY --from=build /mnt/bin/* /usr/bin/
EXPOSE 8080
CMD ["esbuild-service"]
