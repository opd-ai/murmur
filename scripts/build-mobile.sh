#!/bin/bash
# build-mobile.sh — Build MURMUR for iOS and Android using gomobile
#
# Per TECHNICAL_IMPLEMENTATION.md §12 and ROADMAP.md Priority 3,
# Ebitengine supports mobile platforms. This script wraps gomobile
# for cross-platform mobile builds.
#
# Prerequisites:
#   - Go 1.22+
#   - gomobile: go install golang.org/x/mobile/cmd/gomobile@latest
#   - For Android: Android SDK/NDK, ANDROID_HOME set
#   - For iOS: Xcode (macOS only)
#
# Usage:
#   ./scripts/build-mobile.sh android    # Build Android APK
#   ./scripts/build-mobile.sh ios        # Build iOS framework (macOS only)
#   ./scripts/build-mobile.sh all        # Build both platforms
#
# Output:
#   build/murmur.apk        (Android)
#   build/Murmur.xcframework (iOS)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
BUILD_DIR="${PROJECT_ROOT}/build"
MODULE_PATH="github.com/opd-ai/murmur/cmd/murmur"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_gomobile() {
    if ! command -v gomobile &> /dev/null; then
        log_error "gomobile not found. Install with: go install golang.org/x/mobile/cmd/gomobile@latest"
        exit 1
    fi
    log_info "gomobile found: $(which gomobile)"
}

check_android_sdk() {
    if [[ -z "${ANDROID_HOME:-}" ]]; then
        log_error "ANDROID_HOME not set. Install Android SDK and set ANDROID_HOME."
        log_info "Example: export ANDROID_HOME=~/Android/Sdk"
        return 1
    fi
    if [[ ! -d "${ANDROID_HOME}" ]]; then
        log_error "ANDROID_HOME directory does not exist: ${ANDROID_HOME}"
        return 1
    fi
    log_info "Android SDK found: ${ANDROID_HOME}"
    return 0
}

check_xcode() {
    if [[ "$(uname)" != "Darwin" ]]; then
        log_warn "iOS builds require macOS with Xcode"
        return 1
    fi
    if ! command -v xcodebuild &> /dev/null; then
        log_error "Xcode not found. Install from App Store."
        return 1
    fi
    log_info "Xcode found: $(xcodebuild -version | head -1)"
    return 0
}

init_gomobile() {
    log_info "Initializing gomobile..."
    gomobile init
    log_info "gomobile initialized"
}

build_android() {
    log_info "Building for Android..."
    
    if ! check_android_sdk; then
        log_error "Android SDK not available, skipping Android build"
        return 1
    fi
    
    mkdir -p "${BUILD_DIR}"
    cd "${PROJECT_ROOT}"
    
    # Build APK
    log_info "Running: gomobile build -target=android -o ${BUILD_DIR}/murmur.apk ${MODULE_PATH}"
    gomobile build \
        -target=android \
        -androidapi=21 \
        -o "${BUILD_DIR}/murmur.apk" \
        "${MODULE_PATH}"
    
    if [[ -f "${BUILD_DIR}/murmur.apk" ]]; then
        log_info "Android APK built successfully: ${BUILD_DIR}/murmur.apk"
        log_info "Size: $(du -h "${BUILD_DIR}/murmur.apk" | cut -f1)"
    else
        log_error "Android build failed"
        return 1
    fi
}

build_ios() {
    log_info "Building for iOS..."
    
    if ! check_xcode; then
        log_error "Xcode not available, skipping iOS build"
        return 1
    fi
    
    mkdir -p "${BUILD_DIR}"
    cd "${PROJECT_ROOT}"
    
    # Build xcframework
    log_info "Running: gomobile build -target=ios -o ${BUILD_DIR}/Murmur.xcframework ${MODULE_PATH}"
    gomobile build \
        -target=ios \
        -o "${BUILD_DIR}/Murmur.xcframework" \
        "${MODULE_PATH}"
    
    if [[ -d "${BUILD_DIR}/Murmur.xcframework" ]]; then
        log_info "iOS framework built successfully: ${BUILD_DIR}/Murmur.xcframework"
    else
        log_error "iOS build failed"
        return 1
    fi
}

usage() {
    echo "Usage: $0 <platform>"
    echo ""
    echo "Platforms:"
    echo "  android    Build Android APK"
    echo "  ios        Build iOS framework (macOS only)"
    echo "  all        Build both platforms"
    echo "  check      Check build prerequisites"
    echo ""
    echo "Environment variables:"
    echo "  ANDROID_HOME   Path to Android SDK (required for Android builds)"
    echo ""
    echo "Examples:"
    echo "  $0 android"
    echo "  ANDROID_HOME=~/Android/Sdk $0 android"
}

main() {
    if [[ $# -lt 1 ]]; then
        usage
        exit 1
    fi
    
    local platform="$1"
    
    case "${platform}" in
        check)
            log_info "Checking build prerequisites..."
            check_gomobile
            check_android_sdk || true
            check_xcode || true
            log_info "Prerequisites check complete"
            ;;
        android)
            check_gomobile
            init_gomobile
            build_android
            ;;
        ios)
            check_gomobile
            init_gomobile
            build_ios
            ;;
        all)
            check_gomobile
            init_gomobile
            build_android || log_warn "Android build failed, continuing..."
            build_ios || log_warn "iOS build failed, continuing..."
            log_info "Build complete"
            ;;
        -h|--help|help)
            usage
            ;;
        *)
            log_error "Unknown platform: ${platform}"
            usage
            exit 1
            ;;
    esac
}

main "$@"
