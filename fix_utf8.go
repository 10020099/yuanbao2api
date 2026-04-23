package main

import (
	"io/ioutil"
	"strings"
)

func main() {
	content, _ := ioutil.ReadFile("api/anthropic.go")
	text := string(content)

	// 在 hasTools 后添加辅助函数
	old1 := "\thasTools := len(tools) > 0\r\n\r\n\tstreamTimeout := time.NewTimer(120 * time.Second)"
	new1 := `	hasTools := len(tools) > 0

	// UTF-8 安全的字符串切片辅助函数（按字符索引而非字节）
	substringRune := func(s string, start, end int) string {
		runes := []rune(s)
		if start < 0 {
			start = 0
		}
		if end > len(runes) {
			end = len(runes)
		}
		if start >= end {
			return ""
		}
		return string(runes[start:end])
	}

	streamTimeout := time.NewTimer(120 * time.Second)`
	text = strings.Replace(text, old1, new1, 1)

	// 修复第375行
	text = strings.Replace(text, "beforeTag := textBuffer[:startMatch.Index]",
		"beforeTag := substringRune(textBuffer, 0, startMatch.Index)", 1)

	// 修复第376行
	text = strings.Replace(text, "\t\t\t\t\ttextBuffer = textBuffer[startMatch.Index:]",
		"\t\t\t\t\ttextBuffer = substringRune(textBuffer, startMatch.Index, len([]rune(textBuffer)))", 1)

	// 修复第415行
	text = strings.Replace(text, "textBuffer = fullTextStr[endMatch.Index+len(endMatch.Tag):]",
		"textBuffer = substringRune(fullTextStr, endMatch.Index+len(endMatch.Tag), len([]rune(fullTextStr)))", 1)

	// 修复第423行
	text = strings.Replace(text, "beforeNat := textBuffer[:natIdx]",
		"beforeNat := substringRune(textBuffer, 0, natIdx)", 1)

	// 修复第424行
	text = strings.Replace(text, "\t\t\t\t\t\t\ttextBuffer = textBuffer[natIdx:]",
		"\t\t\t\t\t\t\ttextBuffer = substringRune(textBuffer, natIdx, len([]rune(textBuffer)))", 1)

	// 修复第462行
	text = strings.Replace(text, "subText := fullTextStr[fromNatStart:]",
		"subText := substringRune(fullTextStr, fromNatStart, len([]rune(fullTextStr)))", 1)

	// 修复第465行
	text = strings.Replace(text, "textBuffer = fullTextStr[fromNatStart+len(balanced):]",
		"textBuffer = substringRune(fullTextStr, fromNatStart+len(balanced), len([]rune(fullTextStr)))", 1)

	ioutil.WriteFile("api/anthropic.go", []byte(text), 0644)
}
