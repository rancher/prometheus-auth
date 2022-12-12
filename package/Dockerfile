FROM registry.suse.com/bci/bci-micro:15.3
RUN echo 'promauth:x:1000:1000::/home/promauth:/bin/bash' >> /etc/passwd && \
    echo 'promauth:x:1000:' >> /etc/group && \
    mkdir /home/promauth && \
    chown -R promauth:promauth /home/promauth
COPY bin/prometheus-auth /usr/bin/
USER promauth
CMD ["prometheus-auth"]
