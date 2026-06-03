class SockEmoji < Formula
  desc "Tiny macOS menu bar ticker fed by a named pipe"
  homepage "https://github.com/OffPeakEngineer/sock-emoji"

  # Replace this with the real release tarball SHA-256 when publishing v0.1.0.
  url "https://github.com/OffPeakEngineer/sock-emoji/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "0000000000000000000000000000000000000000000000000000000000000000"

  head "https://github.com/OffPeakEngineer/sock-emoji.git", branch: "main"

  depends_on "go" => :build

  on_macos do
    depends_on :macos
  end

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w")
  end

  service do
    run [
      opt_bin/"sock-emoji",
      "-pipe", "#{Dir.home}/sock",
      "-width", "1",
      "-delay", "200ms"
    ]
    keep_alive false
    log_path var/"log/sock-emoji/stdout.log"
    error_log_path var/"log/sock-emoji/stderr.log"
  end

  test do
    assert_match "Usage of", shell_output("#{bin}/sock-emoji -h 2>&1")
  end
end
