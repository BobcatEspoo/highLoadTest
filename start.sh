#!/bin/bash

cd test_run

chmod +x recorder
chmod +x my_linux_app

./recorder

./my_linux_app

sleep 5m

bash saveRecord.sh