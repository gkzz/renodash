FROM ubuntu:24.04@sha256:2198e4c65076e9cb0b3d695219df667c0e3b0072da64ca3c2648aca17dcecaf5
# Complete CLI system setup before the runtime root filesystem becomes read-only.
RUN groupadd --gid 65532 spike && useradd --uid 65532 --gid 65532 --no-create-home --home-dir /tmp/home spike \
    && mkdir /var/devcontainer \
    && touch /var/devcontainer/.patchEtcEnvironmentMarker /var/devcontainer/.patchEtcProfileMarker
USER spike
