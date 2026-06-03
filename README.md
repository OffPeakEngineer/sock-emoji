# sock-emoji

A tiny macOS status bar app written in Go. It creates a named pipe at `~/sock` by default and updates the menu bar title whenever text is written into it.

## Build

```bash
go build -o sock-emoji
```

This uses cgo plus a tiny Objective-C bridge for the macOS status bar item, so it needs the Xcode Command Line Tools installed.

## Run

```bash
./sock-emoji
```

Then in another terminal:

```bash
echo "🧦" > ~/sock
```

The emoji or text will appear in the status bar icon area and update each time the pipe receives new content.

Text is shown as a ticker. By default only one Unicode cluster is visible at a time, so `hello` appears as `h`, `e`, `l`, `l`, `o` instead of taking over the menu bar.

Multiple newline-delimited values sent through one write are applied in order:

```bash
printf "🧦\n✨\nOK\n" > ~/sock
```

## Customize pipe path

```bash
./sock-emoji -pipe ~/my-sock
```

## Ticker options

```bash
./sock-emoji -width 1 -delay 200ms
./sock-emoji -width 3 -delay 100ms
```

`-width` controls how many Unicode clusters are visible at once. `-delay` controls the pause between ticker frames.

## Start on login

The macOS-friendly local install is a per-user LaunchAgent, not a daemon. A LaunchAgent runs inside your login session, so it can show a menu bar item. The included installer builds the binary, installs it under `~/Library/Application Support/sock-emoji`, and registers a `launchd` plist under `~/Library/LaunchAgents`.

```bash
chmod +x scripts/install-launch-agent.sh scripts/uninstall-launch-agent.sh
scripts/install-launch-agent.sh
```

The LaunchAgent uses `RunAtLoad` and intentionally does not set `KeepAlive`, so choosing `Quit` from the menu bar item exits the app and `launchd` leaves it stopped. It starts again at your next login, or when you manually run:

```bash
launchctl kickstart -k "gui/$(id -u)/dev.offpeak.sock-emoji"
```

To customize the installed LaunchAgent:

```bash
SOCK_EMOJI_PIPE="$HOME/my-sock" SOCK_EMOJI_WIDTH=3 SOCK_EMOJI_DELAY=100ms scripts/install-launch-agent.sh
```

To uninstall:

```bash
scripts/uninstall-launch-agent.sh
```

## Homebrew

There is a Homebrew formula template at `Formula/sock-emoji.rb`. Until there is a tagged release with a real SHA-256, install the current branch with:

```bash
brew install --HEAD ./Formula/sock-emoji.rb
brew services start sock-emoji
```

Before publishing a stable release in a tap, create a release tarball and replace the placeholder SHA-256:

```bash
git archive --format=tar.gz --prefix=sock-emoji-0.1.0/ -o sock-emoji-0.1.0.tar.gz v0.1.0
shasum -a 256 sock-emoji-0.1.0.tar.gz
```

Then update:

```ruby
url "https://github.com/OffPeakEngineer/sock-emoji/archive/refs/tags/v0.1.0.tar.gz"
sha256 "..."
head "https://github.com/OffPeakEngineer/sock-emoji.git", branch: "main"
```

Once a tap is published, install and start on login with:

```bash
brew tap OffPeakEngineer/tap
brew install OffPeakEngineer/tap/sock-emoji
brew services start sock-emoji
```

The formula's service uses `keep_alive false`, so the menu bar `Quit` item stops the app and Homebrew/launchd will not restart it until you run `brew services start sock-emoji` again or log in again after starting the service.
