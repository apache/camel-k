FROM alpine:3.6

RUN adduser -D camel-k-operator
USER camel-k-operator

ADD tmp/_output/bin/camel-k-operator /usr/local/bin/camel-k-operator
