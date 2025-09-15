#!/bin/bash
sudo apt install -y google-chrome-stable
# sudo apt install xvfb ffmpeg -y
# Xvfb :99 -screen 0 1920x1080x24 &
# export DISPLAY=:99
# DISPLAY=:99 xclock

# Use the Linux binary we built
chmod +x ./highLoadTest
./highLoadTest

# ./recorder -name=$1

# bash saveRecord.sh