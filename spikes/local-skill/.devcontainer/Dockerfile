FROM python:3.13-slim@sha256:e48f59cdba69a3cbd9ed2cf0f348d90d45a8d350d98134cf1070bc106ee9ed90
# Complete CLI system setup before the runtime root filesystem becomes read-only.
RUN groupadd --gid 65532 spike && useradd --uid 65532 --gid 65532 --no-create-home --home-dir /tmp/home spike \
    && mkdir /var/devcontainer \
    && touch /var/devcontainer/.patchEtcEnvironmentMarker /var/devcontainer/.patchEtcProfileMarker
USER spike
