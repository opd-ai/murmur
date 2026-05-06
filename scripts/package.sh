#!/usr/bin/env bash
# MURMUR Cross-Platform Release Packaging Script
# Creates distributable packages for all target platforms

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BUILD_DIR="$PROJECT_ROOT/bin"
DIST_DIR="$PROJECT_ROOT/dist"
VERSION="${VERSION:-$(cat "$PROJECT_ROOT/VERSION" 2>/dev/null || echo "0.0.0-dev")}"
COMMIT="${COMMIT:-$(git -C "$PROJECT_ROOT" rev-parse --short HEAD 2>/dev/null || echo "unknown")}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[package]${NC} $*"
}

warn() {
    echo -e "${YELLOW}[package]${NC} $*"
}

error() {
    echo -e "${RED}[package]${NC} $*" >&2
}

# Cleanup function
cleanup() {
    log "Cleaning previous builds..."
    rm -rf "$BUILD_DIR" "$DIST_DIR"
    mkdir -p "$BUILD_DIR" "$DIST_DIR"
}

# Build binary for a specific platform
build_platform() {
    local goos=$1
    local goarch=$2
    local output_name="murmur-${goos}-${goarch}"
    
    if [ "$goos" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    log "Building $goos/$goarch..."
    
    cd "$PROJECT_ROOT"
    GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=1 \
        go build -v \
        -ldflags="-s -w -X main.Version=$VERSION -X main.Commit=$COMMIT" \
        -o "$BUILD_DIR/$output_name" \
        ./cmd/murmur
    
    if [ $? -eq 0 ]; then
        log "✅ Built $output_name ($(du -h "$BUILD_DIR/$output_name" | cut -f1))"
    else
        error "❌ Failed to build $output_name"
        return 1
    fi
}

# Create tarball package
package_tarball() {
    local goos=$1
    local goarch=$2
    local binary_name="murmur-${goos}-${goarch}"
    local package_name="murmur-${VERSION}-${goos}-${goarch}"
    local package_dir="$DIST_DIR/${package_name}"
    
    if [ "$goos" = "windows" ]; then
        binary_name="${binary_name}.exe"
    fi
    
    log "Creating package for $goos/$goarch..."
    
    # Create package directory structure
    mkdir -p "$package_dir"
    
    # Copy binary
    cp "$BUILD_DIR/$binary_name" "$package_dir/"
    
    # Copy documentation
    cp "$PROJECT_ROOT/README.md" "$package_dir/"
    cp "$PROJECT_ROOT/LICENSE" "$package_dir/"
    cp "$PROJECT_ROOT/CHANGELOG.md" "$package_dir/" 2>/dev/null || true
    
    # Create VERSION file
    echo "$VERSION (commit $COMMIT)" > "$package_dir/VERSION.txt"
    
    # Create platform-specific archive
    cd "$DIST_DIR"
    if [ "$goos" = "windows" ]; then
        # Use zip for Windows
        log "Creating ${package_name}.zip..."
        zip -qr "${package_name}.zip" "${package_name}"
        rm -rf "${package_name}"
        log "✅ Created ${package_name}.zip ($(du -h "${package_name}.zip" | cut -f1))"
    else
        # Use tar.gz for Unix-like systems
        log "Creating ${package_name}.tar.gz..."
        tar czf "${package_name}.tar.gz" "${package_name}"
        rm -rf "${package_name}"
        log "✅ Created ${package_name}.tar.gz ($(du -h "${package_name}.tar.gz" | cut -f1))"
    fi
}

# Generate checksums
generate_checksums() {
    log "Generating checksums..."
    cd "$DIST_DIR"
    
    # SHA256 checksums
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum *.{tar.gz,zip} 2>/dev/null > checksums-sha256.txt || true
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 *.{tar.gz,zip} 2>/dev/null > checksums-sha256.txt || true
    else
        warn "sha256sum not found, skipping checksum generation"
        return
    fi
    
    log "✅ Generated checksums-sha256.txt"
}

# Generate release notes template
generate_release_notes() {
    log "Generating release notes template..."
    
    cat > "$DIST_DIR/RELEASE_NOTES.md" << EOF
# MURMUR $VERSION

Release Date: $(date +%Y-%m-%d)
Commit: $COMMIT

## Installation

### Linux (amd64)
\`\`\`bash
tar xzf murmur-${VERSION}-linux-amd64.tar.gz
cd murmur-${VERSION}-linux-amd64
./murmur-linux-amd64
\`\`\`

### macOS (Apple Silicon)
\`\`\`bash
tar xzf murmur-${VERSION}-darwin-arm64.tar.gz
cd murmur-${VERSION}-darwin-arm64
./murmur-darwin-arm64
\`\`\`

### Windows (amd64)
\`\`\`powershell
Expand-Archive murmur-${VERSION}-windows-amd64.zip
cd murmur-${VERSION}-windows-amd64
.\\murmur-windows-amd64.exe
\`\`\`

## Changes

See CHANGELOG.md for full release notes.

## Verification

Verify package integrity:
\`\`\`bash
sha256sum -c checksums-sha256.txt
\`\`\`

## Support

- Documentation: https://github.com/opd-ai/murmur
- Issues: https://github.com/opd-ai/murmur/issues
EOF
    
    log "✅ Generated RELEASE_NOTES.md"
}

# Main execution
main() {
    log "MURMUR Release Packaging Script"
    log "Version: $VERSION"
    log "Commit: $COMMIT"
    echo
    
    # Parse arguments
    PLATFORMS="${1:-all}"
    
    if [ "$PLATFORMS" = "help" ] || [ "$PLATFORMS" = "--help" ] || [ "$PLATFORMS" = "-h" ]; then
        cat << EOF
Usage: $0 [PLATFORM]

PLATFORM options:
  all           - Build all platforms (default)
  linux-amd64   - Linux x86_64
  linux-arm64   - Linux ARM64
  darwin-amd64  - macOS Intel
  darwin-arm64  - macOS Apple Silicon
  windows-amd64 - Windows x86_64

Environment variables:
  VERSION       - Override version (default: from VERSION file or "0.0.0-dev")
  COMMIT        - Override commit hash (default: git rev-parse --short HEAD)

Examples:
  $0                    # Build all platforms
  $0 linux-amd64        # Build only Linux amd64
  VERSION=1.0.0 $0      # Build all with version 1.0.0
EOF
        exit 0
    fi
    
    # Cleanup old builds
    cleanup
    
    # Build platforms
    case "$PLATFORMS" in
        all)
            build_platform linux amd64
            build_platform linux arm64
            build_platform darwin amd64
            build_platform darwin arm64
            build_platform windows amd64
            
            package_tarball linux amd64
            package_tarball linux arm64
            package_tarball darwin amd64
            package_tarball darwin arm64
            package_tarball windows amd64
            ;;
        linux-amd64)
            build_platform linux amd64 && package_tarball linux amd64
            ;;
        linux-arm64)
            build_platform linux arm64 && package_tarball linux arm64
            ;;
        darwin-amd64)
            build_platform darwin amd64 && package_tarball darwin amd64
            ;;
        darwin-arm64)
            build_platform darwin arm64 && package_tarball darwin arm64
            ;;
        windows-amd64)
            build_platform windows amd64 && package_tarball windows amd64
            ;;
        *)
            error "Unknown platform: $PLATFORMS"
            error "Run '$0 help' for usage information"
            exit 1
            ;;
    esac
    
    # Generate checksums and release notes
    generate_checksums
    generate_release_notes
    
    echo
    log "✅ Packaging complete!"
    log "Artifacts in: $DIST_DIR"
    ls -lh "$DIST_DIR"
}

main "$@"
