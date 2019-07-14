# oauth2er/oauth2-proxy
# https://github.com/rdeusser/oauth2-proxy
FROM golang:1.10 AS builder

LABEL maintainer="oauth2@bnf.net"

RUN mkdir -p ${GOPATH}/src/github.com/rdeusser/oauth2-proxy
WORKDIR ${GOPATH}/src/github.com/rdeusser/oauth2-proxy

COPY . .

# RUN go-wrapper download  # "go get -d -v ./..."
# RUN ./do.sh build    # see `do.sh` for oauth2 build details
# RUN go-wrapper install # "go install -v ./..."

RUN ./do.sh goget
RUN ./do.sh gobuildstatic # see `do.sh` for oauth2-proxy build details
RUN ./do.sh install

FROM scratch
LABEL maintainer="oauth2@bnf.net"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY templates/ templates/
# see note for /static in main.go
COPY static /static
COPY --from=builder /go/bin/oauth2-proxy /oauth2-proxy
EXPOSE 9090
ENTRYPOINT ["/oauth2-proxy"]
