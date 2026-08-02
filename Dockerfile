FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/clank ./cmd/clank

FROM alpine:3.23
RUN addgroup -S clank && adduser -S -G clank clank && mkdir /data && chown clank:clank /data
COPY --from=build /out/clank /usr/local/bin/clank
USER clank
ENV CLANKSPACE_DATA_DIR=/data
EXPOSE 8080
ENTRYPOINT ["clank"]
CMD ["serve"]
