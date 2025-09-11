#!/bin/bash
sudo apt install sshpass
sudo apt install -y ffmpeg
Xvfb :99 -screen 0 1920x1080x24 &
export DISPLAY=:99
cd test_run

chmod +x recorder
chmod +x my_linux_app

./start_game

./recorder -name=$1

bash saveRecord.sh