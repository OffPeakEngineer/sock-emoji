#!/bin/sh
set -eu

label="dev.offpeak.sock-emoji"
install_dir="$HOME/Library/Application Support/sock-emoji"
plist="$HOME/Library/LaunchAgents/$label.plist"

launchctl bootout "gui/$(id -u)" "$plist" 2>/dev/null || true

rm -f "$plist"
rm -f "$install_dir/sock-emoji"
rmdir "$install_dir" 2>/dev/null || true

echo "Uninstalled $label"
