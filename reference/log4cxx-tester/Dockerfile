FROM java:7-jre

RUN apt-get update && apt-get -y install build-essential libapr1-dev libaprutil1-dev

# Use Ubuntu's Log4cxx instead of debain for the SocketAppender patch - https://bugs.launchpad.net/ubuntu/+source/log4cxx/+bug/785134
ADD http://mirrors.kernel.org/ubuntu/pool/universe/l/log4cxx/liblog4cxx10_0.10.0-1.2ubuntu2_amd64.deb /tmp/liblog4cxx.deb
ADD http://mirrors.kernel.org/ubuntu/pool/universe/l/log4cxx/liblog4cxx10-dev_0.10.0-1.2ubuntu2_amd64.deb /tmp/liblog4cxx-dev.deb

RUN dpkg -i /tmp/liblog4cxx.deb
RUN dpkg -i /tmp/liblog4cxx-dev.deb

VOLUME /build

WORKDIR /build

CMD make
