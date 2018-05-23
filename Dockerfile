FROM golang

RUN curl https://glide.sh/get | sh

WORKDIR /go/src/app
COPY . .
RUN glide install
RUN go build -a app .

ENTRYPOINT ["./app"]