FROM ubuntu:20.04

WORKDIR /tfsuit

RUN apt update -y \
  && apt install -y jq wget

# Install gsht
RUN wget https://github.com/NekoOs/gsht.sh/releases/download/nightly/gsht; \
  mv gsht /usr/local/bin/gsht; \
  chmod a+x /usr/local/bin/gsht;

COPY src .

# Transpile tfsuit
RUN gsht --input tfsuit.sh  --output tfsuit; \
  mv tfsuit /usr/local/bin/tfsuit; \
  chmod a+x /usr/local/bin/tfsuit

COPY entrypoint.sh /entrypoint.sh

RUN chmod a+x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"] 
