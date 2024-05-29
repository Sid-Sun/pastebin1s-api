FROM golang:1.22-bullseye as build

WORKDIR /build
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY Makefile Makefile
COPY main.go main.go
RUN make compile

FROM alpine
WORKDIR /root/app
COPY --from=build /build/out .

CMD [ "/root/app/p1s-api" ]
