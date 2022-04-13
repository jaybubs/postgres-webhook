FROM golang:1.18.0-buster as build-env

RUN mkdir /app
ADD . /app
WORKDIR /app

RUN go mod download

# CGO_ENABLED=0 is a must for multistage builds
RUN CGO_ENABLED=0 go build -o pgwebhook .

FROM alpine:3.15

COPY --from=build-env /app /

CMD ["/pgwebhook"]
