FROM alpine:3.14.2

LABEL maintainer="Loc Ngo <xuanloc0511@gmail.com>"

RUN mkdir -p /app
ADD bin/vis /app/vis
ADD ssl /app/ssl
EXPOSE 80
EXPOSE 443
RUN chmod 755 /app/vis
ENTRYPOINT [ "/app/vis" ]