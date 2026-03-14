class McpKeeper < Formula
  desc "Lightweight stdio proxy for filtering MCP server tool lists"
  homepage "https://github.com/chy168/mcp-keeper"
  version "0.1.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/chy168/mcp-keeper/releases/download/v#{version}/mcp-keeper-darwin-arm64"
      sha256 "REPLACE_WITH_SHA256_DARWIN_ARM64"

      def install
        bin.install "mcp-keeper-darwin-arm64" => "mcp-keeper"
      end
    else
      url "https://github.com/chy168/mcp-keeper/releases/download/v#{version}/mcp-keeper-darwin-amd64"
      sha256 "REPLACE_WITH_SHA256_DARWIN_AMD64"

      def install
        bin.install "mcp-keeper-darwin-amd64" => "mcp-keeper"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/chy168/mcp-keeper/releases/download/v#{version}/mcp-keeper-linux-arm64"
      sha256 "REPLACE_WITH_SHA256_LINUX_ARM64"

      def install
        bin.install "mcp-keeper-linux-arm64" => "mcp-keeper"
      end
    else
      url "https://github.com/chy168/mcp-keeper/releases/download/v#{version}/mcp-keeper-linux-amd64"
      sha256 "REPLACE_WITH_SHA256_LINUX_AMD64"

      def install
        bin.install "mcp-keeper-linux-amd64" => "mcp-keeper"
      end
    end
  end

  test do
    system "#{bin}/mcp-keeper", "--help"
  end
end
