FROM alpine:latest
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
RUN update-ca-certificates
RUN mkdir "/var/z-notes"
RUN mkdir "/var/z-notes/files"
RUN mkdir "/var/z-notes/configuration"
RUN chmod 766 "/var/z-notes"
COPY z-notes "/var/z-notes/"
COPY http "/var/z-notes/http"
RUN chmod +x "/var/z-notes/z-notes"
EXPOSE 8080
WORKDIR /var/z-notes
ENTRYPOINT ["/var/z-notes/z-notes"]