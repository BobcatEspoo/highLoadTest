#!/bin/bash
sudo apt install sshpass
sudo apt install expect
echo "Y"
cd test_run

chmod +x recorder
chmod +x my_linux_app

./my_linux_app

./recorder

bash saveRecord.sh