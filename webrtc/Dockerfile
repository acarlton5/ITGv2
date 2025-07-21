FROM golang:1.14 as builder

WORKDIR /go/src/app
COPY . .
ENV GO111MODULE=on
RUN go mod download
RUN go build -o itg-webrtc .


FROM debian:buster-slim
COPY --from=builder /go/src/app/itg-webrtc /usr/local/bin/
EXPOSE 8080

#CMD ["itg-webrtc --addr=XXX.XXX.XXX.XXX", "run"]
# defaults to localhost:8080, then up to docker compose to bind ports
CMD ["itg-webrtc", "--addr=localhost"]
