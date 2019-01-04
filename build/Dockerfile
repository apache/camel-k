FROM fabric8/s2i-java:2.3

ADD build/_maven_output /tmp/artifacts/m2

USER 0
RUN chgrp -R 0 /tmp/artifacts/m2 \
 && chmod -R g=u /tmp/artifacts/m2

USER 1000

ADD build/_output/bin/camel-k /usr/local/bin/camel-k
