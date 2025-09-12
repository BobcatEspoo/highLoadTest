#!/bin/bash
sudo apt install sshpass
# sudo apt install xvfb ffmpeg -y
# Xvfb :99 -screen 0 1920x1080x24 &
# export DISPLAY=:99
# DISPLAY=:99 xclock
cd test_run

# chmod +x recorder
chmod +x ./start_game
file ./start_game
sudo uname -m
./start_game

# ./recorder -name=$1

# bash saveRecord.sh