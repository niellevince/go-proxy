#!/usr/bin/env bash
set -euo pipefail

if [ "$(id -u)" -eq 0 ]; then
  run() { "$@"; }
else
  run() { sudo "$@"; }
fi

if [ -n "${SUDO_USER:-}" ] && [ "${SUDO_USER}" != "root" ]; then
  target_user="${SUDO_USER}"
else
  target_user="${USER}"
fi

if [ ! -r /etc/os-release ]; then
  echo "This script supports Ubuntu only." >&2
  exit 1
fi
# shellcheck disable=SC1091
. /etc/os-release
if [ "${ID:-}" != "ubuntu" ]; then
  echo "This script supports Ubuntu only. Found: ${ID:-unknown}." >&2
  exit 1
fi

codename="${UBUNTU_CODENAME:-${VERSION_CODENAME:-}}"
if [ -z "${codename}" ]; then
  echo "Could not read the Ubuntu codename from /etc/os-release." >&2
  exit 1
fi

export DEBIAN_FRONTEND=noninteractive

run apt-get update
run apt-get install -y ca-certificates curl
run install -m 0755 -d /etc/apt/keyrings
run curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
run chmod a+r /etc/apt/keyrings/docker.asc

run tee /etc/apt/sources.list.d/docker.sources >/dev/null <<EOF
Types: deb
URIs: https://download.docker.com/linux/ubuntu
Suites: ${codename}
Components: stable
Architectures: $(dpkg --print-architecture)
Signed-By: /etc/apt/keyrings/docker.asc
EOF

run apt-get update
run apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
run systemctl enable --now docker

if ! getent group docker >/dev/null; then
  run groupadd docker
fi

if [ "${target_user}" = "root" ]; then
  echo "Docker is installed. Log in as a normal user and run: sudo usermod -aG docker \$USER" >&2
  exit 0
fi

run usermod -aG docker "${target_user}"

echo
echo "Docker is installed. ${target_user} can run docker without sudo after a new login."
echo "Log out and back in, then run: docker ps"
