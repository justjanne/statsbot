FROM golang

RUN curl https://glide.sh/get | sh

RUN apt-get update && apt-get install -y --no-install-recommends \
imagemagick \
libmagickwand-dev

WORKDIR /go/src/app
COPY . .
RUN glide install
RUN go build -a app .

ENTRYPOINT ["./app"]