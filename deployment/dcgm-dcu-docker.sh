#!/bin/bash
mkdir -p /etc/vdev

docker run --name dcgm-dcu -d --privileged \
  --device=/dev/kfd \
  --device=/dev/mkfd \
  --device=/dev/dri \
  -v /etc/vdev:/etc/vdev \
  -v /etc/hostname:/etc/hostname \
  -v /etc/vdev:/etc/vdev \
  -v /opt/hyhal:/opt/hyhal \
  -v /home/chengdm/config:/home/dcgm/config \
  -p 16081:16081 \
  -e LD_LIBRARY_PATH="/home/dcgm/lib/driver6.3.x" \
  dcgm-dcu:v2.0.0
