FROM golang as builder
WORKDIR /go/src/faxxr
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go install

FROM gcr.io/distroless/base

COPY --from=builder /go/bin/faxxr /root/faxxr
COPY --from=builder /go/src/faxxr/media /root/media
COPY --from=builder /go/src/faxxr/tmp /root/tmp

EXPOSE 9000/tcp
WORKDIR /root

ENTRYPOINT ["/root/faxxr"]
