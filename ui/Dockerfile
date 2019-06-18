FROM amazonlinux:2018.03
MAINTAINER hausenbl@amazon.com

# install jq, the AWS CLI, and EKSphemeral
RUN yum update -y  && \
    yum install unzip jq git -y  && \
    curl -sL https://bootstrap.pypa.io/get-pip.py -o get-pip.py && \
    python get-pip.py && \
    pip install awscli --upgrade && \
    curl -sL http://get.eksphemeral.info/install.sh | bash

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
