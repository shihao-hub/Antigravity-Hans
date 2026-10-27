package main

import "testing"

func TestPsQuote(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"含空格与中文", "C:/Users/Admin/Desktop/Antigravity 中文.lnk", "'C:/Users/Admin/Desktop/Antigravity 中文.lnk'"},
		{"含美元符号（双引号串会被当变量展开）", "C:/Users/a$b/x.lnk", "'C:/Users/a$b/x.lnk'"},
		{"含反引号（双引号串会被当转义符）", "C:/Users/a`b/x.lnk", "'C:/Users/a`b/x.lnk'"},
		{"含单引号（需翻倍）", "C:/it's here/x.lnk", "'C:/it''s here/x.lnk'"},
		{"真实 Windows 路径形状", `C:\Users\Admin\Desktop\Antigravity 中文.lnk`, `'C:\Users\Admin\Desktop\Antigravity 中文.lnk'`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := psQuote(tc.in); got != tc.want {
				t.Fatalf("psQuote(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
