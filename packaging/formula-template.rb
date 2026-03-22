class Tortuga < Formula
  desc "DESCRIPTION"
  homepage "https://REPO_URL"
  license "MIT"
  version "FORMULA_VERSION"

  depends_on "git"

  on_macos do
    on_intel do
      url "https://REPO_URL/releases/download/vFORMULA_VERSION/tortuga-FORMULA_VERSION_darwin_amd64.tar.gz"
      sha256 "SHA256_DARWIN_AMD64"
    end

    on_arm do
      url "https://REPO_URL/releases/download/vFORMULA_VERSION/tortuga-FORMULA_VERSION_darwin_arm64.tar.gz"
      sha256 "SHA256_DARWIN_ARM64"
    end
  end

  def install
    bin.install "tt"
  end

  test do
    system "#{bin}/tt", "--version"
  end
end
