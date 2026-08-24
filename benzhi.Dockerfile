FROM golang:1.26.2
WORKDIR /app
ENV GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY . .
RUN go build -mod=vendor -o /usr/local/bin/stericycle ./cmd/stericycle
EXPOSE 21212
CMD ["/usr/local/bin/stericycle"]
