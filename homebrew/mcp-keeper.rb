class McpKeeper < Formula
  desc "Lightweight stdio proxy for filtering MCP server tool lists"
  homepage "https://github.com/chy168/mcp-keeper"
  version "0.0.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/chy168/mcp-keeper/releases/download/v#{version}/mcp-keeper-darwin-arm64"
      sha256 "b14e44aa57acdc4c39261ed8e44bed460ad4220e427593c0cd2169e0b7fdb504"

      def install
        bin.install "mcp-keeper-darwin-arm64" => "mcp-keeper"
      end
    else
      url "https://github.com/chy168/mcp-keeper/releases/download/v#{version}/mcp-keeper-darwin-amd64"
      sha256 "01a3f138f8f4e2ad1e662c38d9fbc49f52c715abfa693fa0ab36f1b709e91ae9"

      def install
        bin.install "mcp-keeper-darwin-amd64" => "mcp-keeper"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/chy168/mcp-keeper/releases/download/v#{version}/mcp-keeper-linux-arm64"
      sha256 "e944a2adb234df121518a11dc45f2b7bf8d41fa1e077210eaa28d84fc683d7fe"

      def install
        bin.install "mcp-keeper-linux-arm64" => "mcp-keeper"
      end
    else
      url "https://github.com/chy168/mcp-keeper/releases/download/v#{version}/mcp-keeper-linux-amd64"
      sha256 "bdebdf91877db2bccb36b9ce6de20b567b0203e4f4257beed3d1688571265d6d"

      def install
        bin.install "mcp-keeper-linux-amd64" => "mcp-keeper"
      end
    end
  end

  test do
    assert_match "Usage:", shell_output("#{bin}/mcp-keeper 2>&1", 1)
  end
end
