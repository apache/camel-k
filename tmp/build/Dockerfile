FROM fabric8/s2i-java:2.3

#RUN adduser -D camel-k
#USER camel-k

ADD tmp/_maven_output /tmp/artifacts/m2

ADD tmp/_output/bin/camel-k /usr/local/bin/camel-k

USER 0
RUN chgrp -R 0 /tmp/artifacts/m2 \
 && chmod -R g=u /tmp/artifacts/m2

USER 1000
