FROM alpine:latest
RUN apk --no-cache add ca-certificates
RUN mkdir /app
WORKDIR /app
ADD build/ .
EXPOSE 9090 8080
CMD ["./srv_core"]