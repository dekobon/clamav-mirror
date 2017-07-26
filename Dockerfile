FROM centos:7

MAINTAINER Elijah Zupancic <elijah@zupancic.name>

ENV SIGSERVER_VERSION 1.0.4
ENV SIGSERVER_SHA256SUM 7c3438aa29a7693550df579bcdf2b1925584c703d2b7b56726728b2995b39cf7
ENV VERBOSE true
ENV DATA_FILE_PATH /var/clamav/data
ENV DIFF_THRESHOLD 100
ENV DOWNLOAD_MIRROR_URL http://database.clamav.net
ENV DNS_DB_DOMAIN current.cvd.clamav.net
ENV SIGSERVER_PORT 80
ENV UPDATE_HOURLY_INTERVAL 4

# Metadata for Docker containers: http://label-schema.org/
LABEL org.label-schema.name="ClamAV Private Mirror" \
      org.label-schema.description="ClamAV Private Mirror and Updater" \
      org.label-schema.url="https://github.com/dekobon/clamav-mirror" \
      org.label-schema.vcs-url="org.label-schema.vcs-ref" \
      org.label-schema.schema-version="1.0"

RUN yum install -y epel-release && \
    yum update -y && \
    yum install -y clamav && \
    yum clean all

RUN curl --retry 7 --fail -Lso /tmp/sigserver.gz "https://github.com/dekobon/clamav-mirror/releases/download/$SIGSERVER_VERSION/sigserver-$SIGSERVER_VERSION-linux-amd64.gz" && \
    echo "$SIGSERVER_SHA256SUM  /tmp/sigserver.gz" | sha256sum -c && \
    gunzip /tmp/sigserver.gz && \
    mv /tmp/sigserver /usr/local/bin/ && \
    chmod +x /usr/local/bin/sigserver && \
    mkdir -p /var/clamav/data

EXPOSE 80

CMD [ "/usr/local/bin/sigserver" ]
