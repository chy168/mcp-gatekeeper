class McpGatekeeper < Formula
  desc "Lightweight stdio proxy for filtering MCP server tool lists"
  homepage "https://github.com/chy168/mcp-gatekeeper"
  version "__VERSION__"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/chy168/mcp-gatekeeper/releases/download/v__VERSION__/mcp-gatekeeper-darwin-arm64"
      sha256 "__SHA_DARWIN_ARM64__"

      resource "mcp-gatekeeper-secret" do
        url "https://github.com/chy168/mcp-gatekeeper/releases/download/v__VERSION__/mcp-gatekeeper-secret-darwin-arm64"
        sha256 "__SHA_SECRET_DARWIN_ARM64__"
      end

      def install
        bin.install "mcp-gatekeeper-darwin-arm64" => "mcp-gatekeeper"
        resource("mcp-gatekeeper-secret").stage { bin.install "mcp-gatekeeper-secret-darwin-arm64" => "mcp-gatekeeper-secret" }
      end
    else
      url "https://github.com/chy168/mcp-gatekeeper/releases/download/v__VERSION__/mcp-gatekeeper-darwin-amd64"
      sha256 "__SHA_DARWIN_AMD64__"

      resource "mcp-gatekeeper-secret" do
        url "https://github.com/chy168/mcp-gatekeeper/releases/download/v__VERSION__/mcp-gatekeeper-secret-darwin-amd64"
        sha256 "__SHA_SECRET_DARWIN_AMD64__"
      end

      def install
        bin.install "mcp-gatekeeper-darwin-amd64" => "mcp-gatekeeper"
        resource("mcp-gatekeeper-secret").stage { bin.install "mcp-gatekeeper-secret-darwin-amd64" => "mcp-gatekeeper-secret" }
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/chy168/mcp-gatekeeper/releases/download/v__VERSION__/mcp-gatekeeper-linux-arm64"
      sha256 "__SHA_LINUX_ARM64__"

      resource "mcp-gatekeeper-secret" do
        url "https://github.com/chy168/mcp-gatekeeper/releases/download/v__VERSION__/mcp-gatekeeper-secret-linux-arm64"
        sha256 "__SHA_SECRET_LINUX_ARM64__"
      end

      def install
        bin.install "mcp-gatekeeper-linux-arm64" => "mcp-gatekeeper"
        resource("mcp-gatekeeper-secret").stage { bin.install "mcp-gatekeeper-secret-linux-arm64" => "mcp-gatekeeper-secret" }
      end
    else
      url "https://github.com/chy168/mcp-gatekeeper/releases/download/v__VERSION__/mcp-gatekeeper-linux-amd64"
      sha256 "__SHA_LINUX_AMD64__"

      resource "mcp-gatekeeper-secret" do
        url "https://github.com/chy168/mcp-gatekeeper/releases/download/v__VERSION__/mcp-gatekeeper-secret-linux-amd64"
        sha256 "__SHA_SECRET_LINUX_AMD64__"
      end

      def install
        bin.install "mcp-gatekeeper-linux-amd64" => "mcp-gatekeeper"
        resource("mcp-gatekeeper-secret").stage { bin.install "mcp-gatekeeper-secret-linux-amd64" => "mcp-gatekeeper-secret" }
      end
    end
  end

  test do
    assert_match "Usage:", shell_output("#{bin}/mcp-gatekeeper 2>&1", 1)
    assert_match "Usage:", shell_output("#{bin}/mcp-gatekeeper-secret 2>&1", 1)
  end
end
