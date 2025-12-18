#!/bin/bash
# Chrome launcher wrapper with fullscreen and zoom support

CHROME_PATH="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"

# Launch Chrome with fullscreen and force device scale factor (zoom)
exec "$CHROME_PATH" \
  --start-fullscreen \
  --force-device-scale-factor=0.8 \
  "$@"
