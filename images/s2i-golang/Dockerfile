FROM registry.access.redhat.com/rhel8/go-toolset:1.11.6

MAINTAINER Andrew Block <ablock@redhat.com>

LABEL io.k8s.description="Platform for building and running Go applications" \
      io.k8s.display-name="Go Source-To-Image Builder" \
      io.openshift.tags="builder,go" \
      io.openshift.s2i.scripts-url="image:///usr/local/s2i" \
      io.openshift.s2i.destination="/tmp" 

ENV HOME=/opt/app-root \
    GOPATH=/opt/app-root/go \
    GOBIN=/opt/app-root/go/bin \
    PATH=$PATH:/opt/app-root/go/bin \
    STI_SCRIPTS_PATH=/usr/local/s2i

COPY s2i $STI_SCRIPTS_PATH

RUN mkdir -p ${GOPATH}/{bin,src} && \
    chown -R 1001:0 ${HOME} && \
    chmod -R g+rwX ${HOME}

WORKDIR ${HOME}

USER 1001

CMD ["/usr/local/s2i/usage"]
