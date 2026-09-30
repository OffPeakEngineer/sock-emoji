# sock-emoji

A tiny macOS menu-bar / Windows notification-area app written in Go. Send newline-delimited UTF-8 through a named pipe to update its emoji. No notification popups or application window. Linux is not yet supported.

## Windows

Build with Go 1.23 or newer; no C compiler is needed:

```powershell
go build -ldflags=-H=windowsgui -o sock-emoji.exe .
Start-Process .\sock-emoji.exe -WindowStyle Hidden
```

The GUI build runs without a console window. For console diagnostics, use `go run .` instead. Look for the initial sock icon in the notification area beside the clock (Windows may put it in the hidden-icons overflow). Right-click the icon and choose **Quit** to exit.

Send an update from PowerShell:

```powershell
echo '🧦' | .\sock-emoji.exe
echo '✅ Build finished' | .\sock-emoji.exe
```

With piped or file-redirected stdin, the executable runs as a short-lived **sender**: it connects to the running tray instance, streams the input, and exits. It does not create another icon or start a tray instance automatically. If no instance is running, it exits with an error. Long-running producers can send successive newline-delimited updates without closing their output. Use `-pipe` on the sender to select a custom endpoint.

PowerShell 7 sends UTF-8 by default. In Windows PowerShell 5.1, set this once per session (or in your profile) before piping emoji to an executable:

```powershell
$OutputEncoding = [System.Text.UTF8Encoding]::new($false)
```

Otherwise PowerShell 5.1 replaces emoji with question marks before the app receives them. See [PowerShell character encoding](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_character_encoding).

Normal launches without redirected stdin still start the tray app. If your launcher redirects stdin, force tray mode with `-listen`:

```powershell
Start-Process .\sock-emoji.exe -WindowStyle Hidden -ArgumentList '-listen'
```

The existing helper also works without changing PowerShell's output encoding:

```powershell
.\scripts\send-emoji.ps1 '🧦'
.\scripts\send-emoji.ps1 '✅ Build finished'
```

The default endpoint is `\\.\pipe\sock-emoji`. The helper connects with a two-second timeout and writes UTF-8 without a BOM; it works with Windows PowerShell 5.1 and PowerShell 7. Other programs can connect to the same local named pipe and write UTF-8 lines. Avoid shell redirection: Windows named pipes are not Unix FIFOs, and Windows PowerShell's default file encoding is unsuitable.

Windows shows the **first Unicode cluster** from each message as a stationary icon; the message also appears in its tooltip, truncated to the Windows limit. DirectWrite/Direct2D render Segoe UI Emoji in **full color on a transparent background**, including supported skin-tone and joined sequences. No additional font installation is needed. Glyph coverage and composite appearance depend on the installed Windows font version; unsupported symbols may appear as boxes. Ordinary text and symbols without color glyphs use white.

The pipe accepts only your Windows account and rejects remote clients. Multiple scripts can connect independently, including while another writer remains connected. Blank lines are ignored; rapid updates coalesce to the latest pending status. A final line without a newline is accepted when the writer disconnects. Messages must be smaller than 1 MiB; NUL-containing lines are ignored. Starting a second app on the same pipe fails rather than replacing the first.

To choose another pipe:

```powershell
Start-Process .\sock-emoji.exe -WindowStyle Hidden -ArgumentList '-pipe', '\\.\pipe\my-status'
.\scripts\send-emoji.ps1 '✨' -PipeName my-status
echo '✨' | .\sock-emoji.exe -pipe '\\.\pipe\my-status'
```

For startup at login, place a shortcut to the built executable in the folder opened by `shell:startup` in the Windows Run dialog. The app also restores its tray icon when Explorer restarts.

## macOS

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
# Or forward stdin through a short-lived sender process:
echo "🧦" | ./sock-emoji
```

The emoji or text will appear in the status bar icon area and update each time the pipe receives new content.

Text is shown as a ticker. By default only one Unicode cluster is visible at a time, so `hello` appears as `h`, `e`, `l`, `l`, `o` instead of taking over the menu bar.

Multiple newline-delimited values may be sent through one write. Rapid updates coalesce to the latest pending status:

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

The macOS-friendly local install is via a per-user LaunchAgent, not a daemon. A LaunchAgent runs inside your login session, so it can show a menu bar item, all without needing root. The included installer builds the binary, installs it under `~/Library/Application Support/sock-emoji`, and registers a `launchd` plist under `~/Library/LaunchAgents`.

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
