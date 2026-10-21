#!/bin/bash
# start.sh - 启动业务进程（单例锁 + 重复检查 + PID 记录）

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/utils.sh"

readonly SELF_PATH="${BKMS_MONITOR_PATH}/start.sh"
readonly LOG_FILE="${BKMS_MONITOR_LOGS_PATH}/startScript.log"
readonly FILE_MISSING=10001
readonly START_PROCESS_FAIL=10002
readonly STARTUP_WAIT_TIME=3

# 文件锁单例检查
check_singleton() {
    # 使用文件描述符 99 打开脚本文件
    exec 99<>"${SELF_PATH}"

    # 尝试获取文件锁（非阻塞模式）
    if flock -n 99; then
        log_success "Acquired singleton lock successfully."
    else
        log_error "Failed to acquire lock. Another start.sh is already running."
        exit 99
    fi
}

# 检查进程是否已经在运行
# 返回: 0=未运行, 1=已运行
is_process_already_running() {
    local count=$(get_process_count)
    log_info "Current process count: ${count}"

    if [ ${count} -gt 0 ]; then
        log_warn "Process is already running. Skipping startup."
        return 1
    fi

    return 0
}

# 创建必要的目录结构
prepare_directories() {
    # 为空检查
    if [ -z "${TRPC_BIN_PATH_TEMPLATE}" ]; then
      log_fatal "TRPC_BIN_PATH_TEMPLATE is empty."
    fi
}

# 验证可执行文件是否存在
validate_executable() {
    if [ ! -f "${TRPC_BIN_PATH}/${SERVER_NAME}" ]; then
        log_error "Executable file not found: ${TRPC_BIN_PATH}/${SERVER_NAME}"
        exit ${FILE_MISSING}
    fi

    # 设置可执行权限
    chmod a+x ${TRPC_BIN_PATH}/${SERVER_NAME}
    log_info "Executable file validated: ${TRPC_BIN_PATH}/${SERVER_NAME}"
}

# 启动业务进程
start_process() {
    log_info "Starting process: ${SERVER_NAME}"

    # 设置 core dump 文件大小无限制（便于调试）
    ulimit -c unlimited

    # 子 Shell 启动，关闭 fd 99 避免继承文件锁
    (
        # 关闭文件描述符 99，防止子进程继承文件锁
        exec 99>&-

        # 启动进程（后台运行，输出重定向到标准输出/错误）
        # {{.BKMS_CUSTOM_START_SCRIPT}} 是 Go 模板渲染后的，用户 app 启动命令，
        #  exec 模式为 appModel 中的 command、args 以空格拼接后的字符串，shell 模式为 trpc-start.sh 的路径。
        nohup {{.BKMS_CUSTOM_START_SCRIPT}} > /proc/self/fd/1 2> /proc/self/fd/2 &
    )

    # 等待进程启动完成
    sleep ${STARTUP_WAIT_TIME}
}

# 验证进程启动结果
verify_startup() {
    local count=$(get_process_count)
    log_info "Process count after startup: ${count}"

    if [ ${count} -lt 1 ]; then
        log_error "Process failed to start."
        exit ${START_PROCESS_FAIL}
    fi

    # 记录进程 PID 到配置文件
    local pid=$(get_process_pid)
    echo "${pid}" > "${PID_CONF}"
    log_success "Process started successfully. PID: ${pid}"
}

main() {
    # 1. 获取实际的二进制文件名（处理大小写和星号问题）
    get_actual_server_name
    
    # 2. 准备环境
    prepare_directories

    # 3. 清除停止标志，恢复 monitor.sh 的托管进程检查
    clear_stop_flag

    # 4. 单例检查
    check_singleton

    # 5. 检查进程是否已经在运行
    if ! is_process_already_running; then
        exit 0
    fi

    # 6. 验证可执行文件
    validate_executable

    # 7. 启动进程
    start_process

    # 8. 验证启动结果
    verify_startup

    exit 0
}

# 执行主函数
main
