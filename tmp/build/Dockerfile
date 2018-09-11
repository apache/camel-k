FROM fabric8/s2i-java:2.1

#RUN adduser -D camel-k-operator
#USER camel-k-operator

ADD tmp/_maven_output /tmp/artifacts/m2

ADD tmp/_output/bin/camel-k-operator /usr/local/bin/camel-k-operator

USER 0
RUN chgrp -R 0 /tmp/artifacts/m2 \
 && chmod -R g=u /tmp/artifacts/m2

USER 1000
