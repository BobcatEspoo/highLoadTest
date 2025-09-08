#!/bin/bash

cd test_run

chmod +x recorder
chmod +x my_linux_app

./my_linux_app

./recorder

sleep 5m

bash saveRecord.sh