FROM golang:alpine AS build-go

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o animnya-api.site

FROM golang:alpine AS release

WORKDIR /app

COPY --from=build-go /app/animnya-api.site /app

EXPOSE 9999

CMD ["./animnya-api.site"]

