#!/bin/bash
sudo apt install sshpass

chromeLinuxPath="/usr/local/bin/google-chrome"
url="https://x.la/cgs/1754888695/play"

args=(
  "--password-store=basic"
  "--no-first-run"
  "--disable-features=AccountConsistency"
  "--disable-signin-promo"
  "$url"
)

"$chromeLinuxPath" "${args[@]}" &

if [ $? -eq 0 ]; then
  echo "started Successful!"
else
  echo "Error while starting chrome"
  exit 1
fi
