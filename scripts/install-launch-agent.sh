#!/bin/sh
set -eu

label="dev.offpeak.sock-emoji"
repo_dir="$(cd "$(dirname "$0")/.." && pwd)"
install_dir="$HOME/Library/Application Support/sock-emoji"
agent_dir="$HOME/Library/LaunchAgents"
log_dir="$HOME/Library/Logs/sock-emoji"
binary="$install_dir/sock-emoji"
plist="$agent_dir/$label.plist"

pipe_path="${SOCK_EMOJI_PIPE:-$HOME/sock}"
width="${SOCK_EMOJI_WIDTH:-1}"
delay="${SOCK_EMOJI_DELAY:-200ms}"

mkdir -p "$install_dir" "$agent_dir" "$log_dir"

cd "$repo_dir"
go build -o "$binary"

cat > "$plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>$label</string>

  <key>ProgramArguments</key>
  <array>
    <string>$binary</string>
    <string>-pipe</string>
    <string>$pipe_path</string>
    <string>-width</string>
    <string>$width</string>
    <string>-delay</string>
    <string>$delay</string>
  </array>

  <key>RunAtLoad</key>
  <true/>

  <key>StandardOutPath</key>
  <string>$log_dir/stdout.log</string>

  <key>StandardErrorPath</key>
  <string>$log_dir/stderr.log</string>
</dict>
</plist>
EOF

launchctl bootout "gui/$(id -u)" "$plist" 2>/dev/null || true
launchctl bootstrap "gui/$(id -u)" "$plist"
launchctl kickstart -k "gui/$(id -u)/$label"

echo "Installed $label"
echo "Binary: $binary"
echo "LaunchAgent: $plist"
echo "Pipe: $pipe_path"
