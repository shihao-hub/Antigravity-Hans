package main

import "strings"

// psQuote 把字符串转换成 PowerShell 单引号字面量（内部单引号翻倍转义）。
//
// 不能直接用 Go 的 %q：PowerShell 的转义符是反引号而不是反斜杠，
// 且双引号字符串会展开 $ 变量与反引号转义——路径里一旦含这些字符，
// 生成的快捷方式就会指向错误的位置。单引号字面量不做任何展开，最安全。
func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
