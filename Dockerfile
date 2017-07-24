FROM centos:7

MAINTAINER Elijah Zupancic <elijah@zupancic.name>

ENV SIGSERVER_VERSION 1.0.2

# Metadata for Docker containers: http://label-schema.org/
LABEL org.label-schema.name="ClamAV Private Mirror" \
      org.label-schema.description="ClamAV Private Mirror and Updater" \
      org.label-schema.url="https://github.com/dekobon/clamav-mirror" \
      org.label-schema.vcs-url="org.label-schema.vcs-ref" \
      org.label-schema.schema-version="1.0"

RUN yum install -y epel-release && \
    yum update -y && \
    yum install -y clamav && \
    curl --retry 7 --fail -Lso /tmp/sigserver.gz "https://github.com/dekobon/clamav-mirror/releases/download/$SIGSERVER_VERSION/sigserver-$SIGSERVER_VERSION-linux-amd64.gz" && \
    echo 'f5ff94a9cd18e278ae38adeab3db8db2479ce35457a35ce27d2b70746f6743ed  /tmp/sigserver.gz' | sha256sum -c && \
    gunzip /tmp/sigserver.gz && \
    mv /tmp/sigserver /usr/local/bin/ && \
    chmod +x /usr/local/bin/sigserver && \
    mkdir -p /var/clamav/data && \
    yum clean all

EXPOSE 80

CMD [ "/usr/local/bin/sigserver" ]
