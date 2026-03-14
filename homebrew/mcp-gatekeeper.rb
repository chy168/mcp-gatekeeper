class McpGatekeeper < Formula
  desc "Lightweight stdio proxy for filtering MCP server tool lists"
  homepage "https://github.com/chy168/mcp-gatekeeper"
  version "__VERSION__"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/chy168/mcp-gatekeeper/releases/download/v__VERSION__/mcp-gatekeeper-darwin-arm64"
      sha256 "__SHA_DARWIN_ARM64__"

      def install
        bin.install "mcp-gatekeeper-darwin-arm64" => "mcp-gatekeeper"
      end
    else
      url "https://github.com/chy168/mcp-gatekeeper/releases/download/v__VERSION__/mcp-gatekeeper-darwin-amd64"
      sha256 "__SHA_DARWIN_AMD64__"

      def install
        bin.install "mcp-gatekeeper-darwin-amd64" => "mcp-gatekeeper"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/chy168/mcp-gatekeeper/releases/download/v__VERSION__/mcp-gatekeeper-linux-arm64"
      sha256 "__SHA_LINUX_ARM64__"

      def install
        bin.install "mcp-gatekeeper-linux-arm64" => "mcp-gatekeeper"
      end
    else
      url "https://github.com/chy168/mcp-gatekeeper/releases/download/v__VERSION__/mcp-gatekeeper-linux-amd64"
      sha256 "__SHA_LINUX_AMD64__"

      def install
        bin.install "mcp-gatekeeper-linux-amd64" => "mcp-gatekeeper"
      end
    end
  end

  test do
    assert_match "Usage:", shell_output("#{bin}/mcp-gatekeeper 2>&1", 1)
  end
end
