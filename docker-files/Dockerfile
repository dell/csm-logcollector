# Dockerfile to build csm-logcollector image
# based on standard UBI image
FROM registry.access.redhat.com/ubi8/ubi-minimal@sha256:574f201d7ed185a9932c91cef5d397f5298dff9df08bc2ebb266c6d1e6284cd1

RUN microdnf install yum
RUN yum update -y
RUN yum install vi -y
RUN yum install wget -y
RUN wget https://dl.google.com/go/go1.17.2.linux-amd64.tar.gz
RUN yum install tar -y
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
