FROM ubi8:latest

ARG OCP_VERSION
ENV OCP_VERSION ${OCP_VERSION:-3.10}

ADD bin/sign-image bin/scan-image /usr/local/bin/
USER 0 
# The curl install of JQ is required in order to bypass requiring the EPEL repository. The URL can be mirrored in disconnected environments.
COPY ./etc-pki-entitlement /etc/pki/entitlement
RUN rm /etc/rhsm-host && \
    yum repolist > /dev/null && \
    yum module enable -y container-tools:1.0 && \
    yum module install -y container-tools:1.0 && \
    curl -o jq -L https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 && \
    chmod +x ./jq && \
    cp jq /usr/bin && \
    curl -o oc.tar.gz https://mirror.openshift.com/pub/openshift-v4/clients/ocp/latest/openshift-client-linux-4.3.1.tar.gz && \
    tar -xvf oc.tar.gz && \
    chmod +x ./oc && \
    cp oc /usr/bin && \
 #   sed 's/#mount_program/mount_program/' /etc/containers/storage.conf -i cp /etc/containers/storage.conf /etc/containers/storage.conf.orig
    sed -i 's/driver = "overlay"/driver = "vfs"/' /etc/containers/storage.conf && \
    cp /usr/share/containers/libpod.conf /etc/containers/ && \
    sed -i 's/cgroup_manager = "systemd"/cgroup_manager = "cgroupfs"/' /etc/containers/libpod.conf && \
    echo 'events_logger = "file"' >> /etc/containers/libpod.conf && \
    sed -i "s/'registry.redhat.io'/'registry.access.redhat.com', 'registry.redhat.io'/" /etc/containers/registries.conf
#USER 1001