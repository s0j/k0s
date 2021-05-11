ARG BUILDIMAGE=golang:1.13-alpine
FROM $BUILDIMAGE AS build

ARG VERSION

RUN apk add build-base git

RUN cd / && git clone -b v$VERSION --depth=1 https://github.com/etcd-io/etcd.git
WORKDIR /etcd
RUN go version
RUN go mod vendor
RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o etcd

FROM scratch
COPY --from=build /etcd/etcd /bin/etcd
CMD ["/bin/etcd"]
