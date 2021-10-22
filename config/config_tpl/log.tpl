#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

log:
  system: # 链日志配置
    log_level_default: {log_level}       # 默认日志级别
    log_levels:
      core: {log_level}                  # 查看commit block落快信息关键字，需将core改为info级别及以下
      net: {log_level}
      vm: {log_level}                    # 合约中的日志，需将vm改为debug级别
      storage: {log_level}               # sql模式查看sql语句，需将storage改为debug级别
    file_path: ../log/system.log
    max_age: 365                  # 日志最长保存时间，单位：天
    rotation_time: 1              # 日志滚动时间，单位：小时
    log_in_console: false         # 是否展示日志到终端，仅限于调试使用
    show_color: true              # 是否打印颜色日志
#    stack_trace_level: ERROR      # 设置 stack trace 日志级别，当日志信息大于等于此级别时将会输出 stack trace, 置空或无效级别将不会打开 stack trace
  brief:
    log_level_default: {log_level}
    file_path: ../log/brief.log
    max_age: 365                  # 日志最长保存时间，单位：天
    rotation_time: 1              # 日志滚动时间，单位：小时
    log_in_console: false         # 是否展示日志到终端，仅限于调试使用
    show_color: true              # 是否打印颜色日志
#    stack_trace_level: ERROR      # 设置 stack trace 日志级别，当日志信息大于等于此级别时将会输出 stack trace, 置空或无效级别将不会打开 stack trace
  event:
    log_level_default: {log_level}
    file_path: ../log/event.log
    max_age: 365                  # 日志最长保存时间，单位：天
    rotation_time: 1              # 日志滚动间隔，单位：小时
    log_in_console: false         # 是否展示日志到终端，仅限于调试使用
    show_color: true              # 是否打印颜色日志
#    stack_trace_level: ERROR      # 设置 stack trace 日志级别，当日志信息大于等于此级别时将会输出 stack trace, 置空或无效级别将不会打开 stack trace
