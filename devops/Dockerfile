FROM alpine:latest
RUN apk --no-cache add ca-certificates
RUN mkdir /app
WORKDIR /app
ADD build/ .
EXPOSE 8090 9090
CMD ["./srv_core"]