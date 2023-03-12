FROM golang:buster as build

WORKDIR /build
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY Makefile Makefile
COPY main.go main.go
RUN make compile

FROM alpine
WORKDIR /root/app
RUN apk add libc6-compat --no-cache
COPY --from=build /build/out .

CMD [ "/root/app/p1s-api" ]
