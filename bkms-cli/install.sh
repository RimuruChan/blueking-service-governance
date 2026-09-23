#!/bin/sh
# TencentBlueKing is pleased to support the open source community by making
# 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
# Copyright (C) Tencent. All rights reserved.
# Licensed under the MIT License (the "License"); you may not use this file except
# in compliance with the License. You may obtain a copy of the License at
#
#  http://opensource.org/licenses/MIT
#
# Unless required by applicable law or agreed to in writing, software distributed under
# the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
# either express or implied. See the License for the specific language governing permissions and
# limitations under the License.
#
# We undertake not to change the open source license (MIT license) applicable
# to the current version of the project delivered to anyone in the future.

# Run with sh, including through curl/wget | sh.
set -eu

readonly REPOSITORY='TencentBlueKing/blueking-service-governance'
readonly LATEST_URL="https://raw.githubusercontent.com/$REPOSITORY/main/bkms-cli/latest.txt"
readonly DOWNLOAD_URL_TEMPLATE="https://github.com/$REPOSITORY/releases/download/bkms-cli%2Fv{version}/{archive}"

# Internal distributions may set both defaults here. Public distribution leaves them empty.
readonly DEFAULT_UPDATE_LATEST_URL=''
readonly DEFAULT_UPDATE_DOWNLOAD_URL_TEMPLATE=''
readonly CR=$(printf '\r')

fail() {
    printf 'Error: %s\n' "$*" >&2
    exit 1
}

usage() {
    printf '%s\n' 'Install bkms-cli from GitHub Releases (macOS/Linux, amd64/arm64).

Usage: sh install.sh [options]
  --version VERSION    Install a specific version; default: latest stable release.
  --install-dir DIR    Installation directory; default: $HOME/.local/bin.
  --bkms-base-url URL  Replace the service URL; omit to keep the existing value.
  --update-latest-url URL             Custom version file URL.
  --update-download-url-template URL  Custom URL using {version} and {archive}.
  -h, --help                          Show this help.

Custom update URLs must be supplied together. Arguments override BKMS_CLI_UPDATE_*
environment variables and the defaults above. Installation replaces the saved update source.

Requires curl or wget, tar, basic Unix commands, and one of
sha256sum, shasum or openssl. No jq, Python, Node.js or Go is needed.'
}

# Helpers run in subshells so their working variables cannot change main's state.
first_available() (
    for tool do
        if command -v "$tool" >/dev/null 2>&1; then
            printf '%s\n' "$tool"
            return 0
        fi
    done
    return 1
)

detect_platform() (
    system=$(uname -s)
    machine=$(uname -m)
    case "$system" in
        Darwin) system=darwin ;;
        Linux) system=linux ;;
        *) fail "Unsupported OS: $system. Use install.ps1 on Windows." ;;
    esac
    case "$machine" in
        x86_64|amd64) machine=amd64 ;;
        aarch64|arm64) machine=arm64 ;;
        *) fail "Unsupported architecture: $machine" ;;
    esac
    printf '%s_%s\n' "$system" "$machine"
)

# download DOWNLOADER URL DESTINATION
download() (
    downloader=$1
    url=$2
    destination=$3
    case "$downloader" in
        curl)
            curl --fail --location --silent --show-error --retry 2 \
                --connect-timeout 10 --max-time 300 --output "$destination" "$url" ||
                fail "Download failed: $url"
            ;;
        wget)
            wget --quiet --timeout=30 --tries=3 --output-document="$destination" "$url" ||
                fail "Download failed: $url"
            ;;
    esac
)

# resolve_version REQUESTED_VERSION DOWNLOADER WORK_DIR LATEST_URL
resolve_version() (
    requested=$1
    downloader=$2
    work_dir=$3

    if [ -n "$requested" ]; then
        resolved=${requested#bkms-cli/}
        resolved=${resolved#v}
    else
        download "$downloader" "$4" "$work_dir/latest.txt"
        resolved=
        while IFS= read -r line || [ -n "$line" ]; do
            [ -z "$resolved" ] || fail "latest.txt must contain a single version."
            resolved=${line%"$CR"}
        done < "$work_dir/latest.txt"
        case "$resolved" in
            ''|*[!0-9.]*) fail "Invalid latest.txt; specify --version X.Y.Z." ;;
        esac
    fi
    case "$resolved" in
        ''|*[!0-9A-Za-z.+-]*) fail "Invalid version: $resolved" ;;
    esac
    printf '%s\n' "$resolved" |
        LC_ALL=C grep -Eq '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$' ||
        fail "Invalid version: $resolved"
    printf '%s\n' "$resolved"
)

# download_url TEMPLATE VERSION ARCHIVE -- literal substitution, never evaluated as shell code.
download_url() (
    remaining=$1
    while :; do
        case "$remaining" in
            *'{'*)
                prefix=${remaining%%\{*}
                case "$prefix" in *'}'*) fail "Invalid download URL template." ;; esac
                printf '%s' "$prefix"
                remaining=${remaining#*\{}
                key=${remaining%%\}*}
                [ "$key" != "$remaining" ] || fail "Unclosed URL placeholder."
                remaining=${remaining#*\}}
                case "$key" in
                    version) printf '%s' "$2" ;;
                    archive) printf '%s' "$3" ;;
                    *) fail "Unknown URL placeholder: {$key}" ;;
                esac
                ;;
            *'}'*) fail "Invalid download URL template." ;;
            *) printf '%s\n' "$remaining"; break ;;
        esac
    done
)

# Check both addresses before making any network request.
validate_update_source() (
    case "$1" in *'{'*|*'}'*) fail "The latest-version URL cannot contain placeholders." ;; esac
    case "$2" in *'{archive}'*) ;; *) fail "Download URL template must contain {archive}." ;; esac
    rendered=$(download_url "$2" 1.0.0 checksums.txt)
    for address in "$1" "$rendered"; do
        case "$address" in
            http://?*|https://?*) ;;
            *) fail "Update URLs must use HTTP or HTTPS." ;;
        esac
        case "$address" in *[[:space:]]*|*'#'*) fail "Update URLs cannot contain whitespace or fragments." ;; esac
    done
)

# verify_checksum CHECKSUM_TOOL ARCHIVE CHECKSUM_FILE
verify_checksum() (
    checksum_tool=$1
    archive=$2
    checksum_file=$3

    expected=
    while read -r hash name extra || [ -n "$hash$name$extra" ]; do
        name=${name%"$CR"}
        [ "${name#\*}" = "${archive##*/}" ] || continue
        [ -z "$expected" ] && [ -z "$extra" ] ||
            fail "Duplicate or malformed checksum for ${archive##*/}."
        expected=$hash
    done < "$checksum_file"
    case "$expected" in
        ''|*[!0-9a-fA-F]*) fail "Missing or invalid SHA-256 checksum for ${archive##*/}." ;;
    esac
    [ "${#expected}" -eq 64 ] || fail "Invalid SHA-256 checksum length."

    # Hash stdin so filenames containing spaces or backslashes do not affect the output.
    case "$checksum_tool" in
        sha256sum) output=$(sha256sum < "$archive"); actual=${output%% *} ;;
        shasum) output=$(shasum -a 256 < "$archive"); actual=${output%% *} ;;
        openssl) output=$(openssl dgst -sha256 < "$archive"); actual=${output##* } ;;
    esac
    printf '%s\n' "$actual" | LC_ALL=C grep -Fqix "$expected" ||
        fail "SHA-256 mismatch; existing installation was not changed."
)

# install_archive ARCHIVE WORK_DIR DESTINATION
install_archive() (
    archive=$1
    work_dir=$2
    destination=$3
    binary=$work_dir/bkms-cli

    # Extract only the executable; never extract arbitrary paths from the archive.
    tar -xzf "$archive" -C "$work_dir" bkms-cli
    [ -f "$binary" ] && [ ! -L "$binary" ] ||
        fail "Archive does not contain a regular bkms-cli executable."
    chmod 755 "$binary"
    "$binary" version || fail "Downloaded binary could not run; existing installation was not changed."

    [ ! -d "$destination" ] || fail "$destination is a directory."
    # main creates WORK_DIR beside DESTINATION, allowing atomic replacement.
    mv -f "$binary" "$destination"
)

print_path_hint() (
    install_dir=$1
    case ":${PATH:-}:" in
        *:"$install_dir":*) ;;
        *) printf 'Add %s to PATH in your shell configuration.\n' "$install_dir" ;;
    esac
    current=$(command -v bkms-cli || true)
    if [ -n "$current" ] && [ "$current" != "$install_dir/bkms-cli" ]; then
        printf 'PATH currently selects %s; adjust PATH to use this installation.\n' "$current"
    fi
)

main() (
    requested_version=
    install_dir=
    base_url=
    latest_url=${BKMS_CLI_UPDATE_LATEST_URL:-$DEFAULT_UPDATE_LATEST_URL}
    download_template=${BKMS_CLI_UPDATE_DOWNLOAD_URL_TEMPLATE:-$DEFAULT_UPDATE_DOWNLOAD_URL_TEMPLATE}

    # Parse options before checking tools, so --help works without download tools.
    while [ "$#" -gt 0 ]; do
        case "$1" in
            -h|--help) usage; return 0 ;;
            --version|--install-dir|--bkms-base-url|--update-latest-url|--update-download-url-template)
                [ "$#" -ge 2 ] && [ -n "$2" ] || fail "$1 requires a value."
                case "$2" in --*) fail "$1 requires a value." ;; esac
                case "$1" in
                    --version) requested_version=$2 ;;
                    --install-dir) install_dir=$2 ;;
                    --bkms-base-url) base_url=$2 ;;
                    --update-latest-url) latest_url=$2 ;;
                    --update-download-url-template) download_template=$2 ;;
                esac
                shift 2
                ;;
            *) fail "Unknown argument: $1. See --help." ;;
        esac
    done

    if [ -n "$latest_url$download_template" ]; then
        [ -n "$latest_url" ] && [ -n "$download_template" ] ||
            fail "Set both update URLs together."
    else
        latest_url=$LATEST_URL
        download_template=$DOWNLOAD_URL_TEMPLATE
    fi
    validate_update_source "$latest_url" "$download_template"

    for tool in uname mkdir mktemp grep tar chmod mv rm; do
        command -v "$tool" >/dev/null 2>&1 || fail "Required system tool not found: $tool"
    done
    downloader=$(first_available curl wget) || fail "Install curl or wget first."
    checksum_tool=$(first_available sha256sum shasum openssl) ||
        fail "Install sha256sum, shasum or openssl first."
    platform=$(detect_platform)

    if [ -z "$install_dir" ]; then
        [ -n "${HOME:-}" ] || fail 'HOME is unset; use --install-dir.'
        install_dir=$HOME/.local/bin
    fi
    case "$install_dir" in
        /*) ;;
        *) install_dir=$PWD/$install_dir ;;
    esac
    mkdir -p "$install_dir"
    install_dir=$(CDPATH= cd "$install_dir" && pwd -P)
    destination=$install_dir/bkms-cli
    [ ! -d "$destination" ] || fail "$destination is a directory."

    # One workspace owns all temporary files and is removed on success or failure.
    work_dir=$(mktemp -d "$install_dir/.bkms-cli.XXXXXXXX")
    trap 'rm -rf "$work_dir"' 0
    trap 'exit 1' HUP INT TERM

    release_version=$(resolve_version "$requested_version" "$downloader" "$work_dir" "$latest_url")
    archive_name="bkms-cli_${release_version}_${platform}.tar.gz"
    archive=$work_dir/$archive_name
    checksums=$work_dir/checksums.txt

    printf 'Installing bkms-cli %s (%s)...\n' "$release_version" "$platform"
    archive_url=$(download_url "$download_template" "$release_version" "$archive_name")
    checksums_url=$(download_url "$download_template" "$release_version" checksums.txt)
    download "$downloader" "$archive_url" "$archive"
    download "$downloader" "$checksums_url" "$checksums"
    verify_checksum "$checksum_tool" "$archive" "$checksums"
    install_archive "$archive" "$work_dir" "$destination"

    set -- config set --update-latest-url "$latest_url" --update-download-url-template "$download_template"
    [ -z "$base_url" ] || set -- "$@" --bkms-base-url "$base_url"
    "$destination" "$@" || fail "Installed bkms-cli, but saving endpoint configuration failed."
    printf 'Installed to %s\n' "$destination"
    print_path_hint "$install_dir"
)

main "$@"
