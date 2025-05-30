FROM debian:buster

# Install common packages
RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -Rf /var/lib/apt/lists/*

# Deploy component
RUN mkdir /app
COPY build/selector /app

CMD [ "/app/selector" ]
