FROM golang as builder
RUN curl https://glide.sh/get | sh

WORKDIR /go/src/app
COPY glide.lock glide.yaml ./
RUN glide install
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a app .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/app/app /app
ENTRYPOINT ["/app"]