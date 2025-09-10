#!/bin/bash
sudo apt install sshpass
sudo apt install ffmpeg
cd test_run

chmod +x recorder
chmod +x my_linux_app

./start_game

./recorder -name=$1

bash saveRecord.sh