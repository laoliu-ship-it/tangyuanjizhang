FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY server /app/server
RUN chmod +x /app/server
EXPOSE 8080
CMD ["/app/server"]
