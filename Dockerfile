FROM golang:1.19-alpine AS build

WORKDIR /app
# ADD go.sum .
# ADD go.mod .
# ADD data.db .
# ADD website .
# ADD main.go .
# COPY . .

ADD website ./website
COPY . .

RUN go mod download

RUN apk upgrade
RUN apk add gcc musl-dev
RUN apk add sqlite
RUN CGO_ENABLED=1 GOOS=linux go build -o http-server -a -ldflags '-linkmode external -extldflags "-static"' .
RUN apk del gcc musl-dev
# FROM scratch
# COPY --from=builder /app /app
ENV HTTP_PORT="8080"
EXPOSE 8080

# CMD ["ash"]
ENTRYPOINT ["/app/http-server"]
