# Dockerfile to build csm-logcollector image
# based on standard UBI image
FROM registry.access.redhat.com/ubi8/ubi:latest

RUN yum update -y
RUN yum install wget -y
RUN wget https://dl.google.com/go/go1.17.2.linux-amd64.tar.gz
RUN tar -xzf go1.17.2.linux-amd64.tar.gz
RUN mv go /usr/local
RUN echo "export PATH=$PATH:/usr/local/go/bin" > /etc/profile.d/ubuntu_path.sh

LABEL vendor="Dell Inc." \
      name="csm-log collector" \
      summary="CSI log collection application" \
      description="CSI log collection application" \
      version="1.0.0" \
      license="Apache-2.0"

COPY "build/" /root/csm-logcollector/
EXPOSE 22