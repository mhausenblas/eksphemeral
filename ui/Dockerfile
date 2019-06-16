FROM amazonlinux:2018.03
MAINTAINER hausenbl@amazon.com

COPY css/* /app/css/
COPY img/* /app/img/
COPY js/* /app/js/
COPY *.html /app/
WORKDIR /app
RUN chown -R 1001:1 /app
USER 1001
COPY proxy .
EXPOSE 8080
CMD ["/app/proxy"]
