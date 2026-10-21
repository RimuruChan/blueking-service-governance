#!/usr/bin/env bats

load '../../helpers/base.bash'

teardown() {
    if [ -n "${app_pid:-}" ]; then
        pkill -P "${app_pid}" 2>/dev/null || true
    fi
    kill ${app_pid:-} ${wrapper_pid:-} 2>/dev/null || true
}

@test "trpc get_process_pid picks the binary rather than the sh wrapper running trpc-start.sh" {
    local bin_dir="${BATS_TEST_TMPDIR}/bin"
    local pid_file="${BATS_TEST_TMPDIR}/app.pid"
    local start_script="${BATS_TEST_TMPDIR}/trpc-start.sh"

    # 模拟业务二进制：记录自身 PID 后常驻
    mkdir -p "${bin_dir}"
    printf '#!/bin/sh\necho $$ > %s\nsleep 30\n' "${pid_file}" > "${bin_dir}/demo-app"
    chmod +x "${bin_dir}/demo-app"
    # 模拟 ShCommand 模式生成的 trpc-start.sh：业务命令后还有其他语句，包装 shell 会一直存活
    printf '#!/bin/sh\n%s\necho "server exited"\n' "${bin_dir}/demo-app" > "${start_script}"
    chmod +x "${start_script}"

    "${start_script}" 3>&- &
    wrapper_pid=$!
    for _ in $(seq 50); do
        [ -s "${pid_file}" ] && break
        sleep 0.1
    done
    [ -s "${pid_file}" ] || { echo "demo-app did not start: ${pid_file} is empty" >&2; return 1; }
    app_pid=$(cat "${pid_file}")

    # shellcheck disable=SC2016
    run env SCRIPT_PATH="$(assets_workspace)/trpc/utils.sh" BIN_DIR="${bin_dir}" bash -c '
        source "${SCRIPT_PATH}"
        TRPC_BIN_PATH="${BIN_DIR}"
        SERVER_NAME="demo-app"
        echo "$(get_process_count) $(get_process_pid)"
    '

    assert_run_status_equals 0 "sh_wrapper" || return 1
    assert_run_output_equals "1 ${app_pid}" "sh_wrapper" || return 1
}
