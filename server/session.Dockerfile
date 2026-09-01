# The image every Session runs in.
#
# It gives a Session two things debian:bookworm-slim does not: apt package
# lists, so "apt-get install" works with no download first, and my-shell as
# the default command.
#
# Build it with "make build-session-image". The server never builds or pulls
# this image, so a Session fails to start until the image is there.

# --- build my-shell ---------------------------------------------------------
FROM golang:1.25-bookworm AS build

ARG MY_SHELL_REPO=https://github.com/iAmAdheil/my-shell.git
ARG MY_SHELL_REF=master

WORKDIR /src

RUN git clone --depth 1 --branch "${MY_SHELL_REF}" "${MY_SHELL_REPO}" /src

RUN CGO_ENABLED=0 go build -o /out/my-shell ./app

# --- the Session image ------------------------------------------------------
FROM debian:bookworm-slim

# The lists stay in the image layer. Rebuild the image to refresh them.
RUN apt-get update

COPY --from=build /out/my-shell /usr/local/bin/my-shell

CMD ["/usr/local/bin/my-shell"]
