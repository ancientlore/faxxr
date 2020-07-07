FROM golang:1.14.4 as builder
WORKDIR /go/src/faxxr
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go install
WORKDIR /go/foo
RUN echo "root:x:0:0:user:/home:/bin/bash" > passwd && echo "nobody:x:65534:65534:user:/home:/bin/bash" >> passwd
RUN echo "root:x:0:" > group && echo "nobody:x:65534:" >> group

FROM gcr.io/distroless/static:latest
COPY --from=builder /go/foo/group /etc/group
COPY --from=builder /go/foo/passwd /etc/passwd

COPY --from=builder /go/bin/faxxr /usr/bin/faxxr
COPY --from=builder /go/src/faxxr/media /faxxr/media
COPY --from=builder --chown=nobody:nobody /go/src/faxxr/tmp /faxxr/tmp

EXPOSE 9000/tcp
USER nobody:nobody
WORKDIR /faxxr
ENTRYPOINT ["/usr/bin/faxxr"]
