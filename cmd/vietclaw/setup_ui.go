package main

import (
	"fmt"
	"os"
	"strings"
)

const (
	uiReset = "\033[0m"
	uiBold  = "\033[1m"
	uiDim   = "\033[2m"
	uiMark  = "\033[7m" // reverse video for cursor line
)

func uiTitle(title string) {
	fmt.Println()
	fmt.Printf("%s%s%s\n", uiBold, title, uiReset)
	fmt.Printf("%s%s%s\n", uiDim, strings.Repeat("─", 42), uiReset)
}

func uiHint(text string) {
	fmt.Printf("%s%s%s\n", uiDim, text, uiReset)
}

func uiOk(text string) {
	fmt.Printf("  ok  %s\n", text)
}

func uiWarn(text string) {
	fmt.Printf("  !   %s\n", text)
}

func uiSelectPrompt(label string) {
	fmt.Printf("%s%s%s\n", uiBold, label, uiReset)
	fmt.Printf("%s↑/↓ chọn · Enter xác nhận%s\n", uiDim, uiReset)
}

func promptSingleSelectClean(label string, options []string) (int, error) {
	if len(options) == 0 {
		return -1, nil
	}
	if len(options) == 1 {
		return 0, nil
	}

	cleanup, err := setTerminalRaw()
	if err != nil {
		fmt.Println(options[0])
		return 0, nil
	}
	defer cleanup()

	uiSelectPrompt(label)

	cursor := 0
	printed := 0
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	draw := func() {
		if printed > 0 {
			fmt.Print(strings.Repeat("\033[F\033[K", printed))
		}
		printed = 0
		for i, opt := range options {
			marker := "  "
			if i == cursor {
				marker = "> "
			}
			line := fmt.Sprintf("%s %s", marker, opt)
			if i == cursor {
				fmt.Printf("%s%s%s\n", uiMark, line, uiReset)
			} else {
				fmt.Println(line)
			}
			printed++
		}
	}

	draw()

	for {
		key, err := readKey()
		if err != nil {
			return cursor, nil
		}
		switch key {
		case "up":
			cursor--
			if cursor < 0 {
				cursor = len(options) - 1
			}
			draw()
		case "down":
			cursor++
			if cursor >= len(options) {
				cursor = 0
			}
			draw()
		case "enter":
			fmt.Printf("%s  → %s%s\n", uiDim, options[cursor], uiReset)
			return cursor, nil
		case "ctrlc", "escape":
			fmt.Println()
			os.Exit(0)
		}
	}
}

func pickModelClean(models []string, defaultModel, label string) string {
	if len(models) == 0 {
		return defaultModel
	}
	start := 0
	for i, m := range models {
		if m == defaultModel {
			start = i
			break
		}
	}

	cleanup, err := setTerminalRaw()
	if err != nil {
		return defaultModel
	}
	defer cleanup()

	uiSelectPrompt(label)

	cursor := start
	printed := 0
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	draw := func() {
		if printed > 0 {
			fmt.Print(strings.Repeat("\033[F\033[K", printed))
		}
		printed = 0

		winStart := cursor - 4
		if winStart < 0 {
			winStart = 0
		}
		winEnd := winStart + 8
		if winEnd > len(models) {
			winEnd = len(models)
			winStart = winEnd - 8
			if winStart < 0 {
				winStart = 0
			}
		}

		if winStart > 0 {
			fmt.Printf("%s  … %d trên%s\n", uiDim, winStart, uiReset)
			printed++
		}
		for i := winStart; i < winEnd; i++ {
			marker := "  "
			if i == cursor {
				marker = "> "
			}
			line := fmt.Sprintf("%s %s", marker, models[i])
			if i == cursor {
				fmt.Printf("%s%s%s\n", uiMark, line, uiReset)
			} else {
				fmt.Println(line)
			}
			printed++
		}
		if winEnd < len(models) {
			fmt.Printf("%s  … %d dưới%s\n", uiDim, len(models)-winEnd, uiReset)
			printed++
		}
	}

	draw()

	for {
		key, err := readKey()
		if err != nil {
			return models[cursor]
		}
		switch key {
		case "up":
			if cursor > 0 {
				cursor--
			}
			draw()
		case "down":
			if cursor < len(models)-1 {
				cursor++
			}
			draw()
		case "enter":
			fmt.Printf("%s  → %s%s\n", uiDim, models[cursor], uiReset)
			return models[cursor]
		case "ctrlc", "escape":
			fmt.Println()
			os.Exit(0)
		}
	}
}
