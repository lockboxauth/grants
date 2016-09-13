FROM busybox

COPY ./grantsd /grantsd
COPY ./ca-certificates.crt /etc/ssl/certs/

EXPOSE 4002

CMD ["/grantsd"]
