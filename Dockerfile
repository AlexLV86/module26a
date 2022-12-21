FROM golang
RUN mkdir -p /go/src/pipeline
WORKDIR /go/src/pipeline
ADD pipeline.go .
ADD go.mod .
RUN go install .

FROM alpine
LABEL version="1.0.0"
LABEL maintainer="Alexander Likhanov<canlik@mail.ru>"
WORKDIR /root/
COPY --from=0 /go/bin/pipeline .
ENTRYPOINT ./pipeline
