FROM fabric8/s2i-java:3.0-java8

ADD build/_maven_output /tmp/artifacts/m2

USER 0
RUN chgrp -R 0 /tmp/artifacts/m2 \
 && chmod -R g=u /tmp/artifacts/m2

USER 1000

ADD build/_output/bin/camel-k /usr/local/bin/camel-k
