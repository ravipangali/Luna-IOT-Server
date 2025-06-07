package colors

import (
	"fmt"
	"time"
)

// ANSI color codes
const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
	Italic = "\033[3m"

	// Regular colors
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Gray    = "\033[90m"

	// Bright colors
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
	BrightWhite   = "\033[97m"

	// Background colors
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
)

// PrintInfo prints informational messages with cyan color
func PrintInfo(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %sℹ%s  %s%s%s\n",
		Gray, timestamp, Reset,
		Cyan, Reset,
		BrightCyan, fmt.Sprintf(format, args...), Reset)
}

// PrintSuccess prints success messages with green color
func PrintSuccess(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %s✅%s %s%s%s\n",
		Gray, timestamp, Reset,
		Green, Reset,
		BrightGreen, fmt.Sprintf(format, args...), Reset)
}

// PrintWarning prints warning messages with yellow color
func PrintWarning(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %s⚠️ %s %s%s%s\n",
		Gray, timestamp, Reset,
		Yellow, Reset,
		BrightYellow, fmt.Sprintf(format, args...), Reset)
}

// PrintError prints error messages with red color
func PrintError(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %s❌%s %s%s%s\n",
		Gray, timestamp, Reset,
		Red, Reset,
		BrightRed, fmt.Sprintf(format, args...), Reset)
}

// PrintHeader prints header messages with bold styling
func PrintHeader(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	border := "═══════════════════════════════════════════════════════════════"
	fmt.Printf("\n%s%s╔%s╗%s\n", BrightBlue, Bold, border[:len(message)+2], Reset)
	fmt.Printf("%s%s║ %s ║%s\n", BrightBlue, Bold, message, Reset)
	fmt.Printf("%s%s╚%s╝%s\n\n", BrightBlue, Bold, border[:len(message)+2], Reset)
}

// PrintSubHeader prints sub-header messages
func PrintSubHeader(format string, args ...interface{}) {
	fmt.Printf("%s%s▶ %s%s\n", BrightMagenta, Bold, fmt.Sprintf(format, args...), Reset)
}

// PrintServer prints server-related messages
func PrintServer(icon, format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %s%s%s %s%s%s\n",
		Gray, timestamp, Reset,
		BrightBlue, icon, Reset,
		White, fmt.Sprintf(format, args...), Reset)
}

// PrintConnection prints connection-related messages
func PrintConnection(icon, format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %s%s%s %s%s%s\n",
		Gray, timestamp, Reset,
		BrightGreen, icon, Reset,
		White, fmt.Sprintf(format, args...), Reset)
}

// PrintData prints data-related messages
func PrintData(icon, format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %s%s%s %s%s%s\n",
		Gray, timestamp, Reset,
		Cyan, icon, Reset,
		BrightWhite, fmt.Sprintf(format, args...), Reset)
}

// PrintControl prints control-related messages
func PrintControl(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %s⚡%s %s%s%s\n",
		Gray, timestamp, Reset,
		BrightYellow, Reset,
		Yellow, fmt.Sprintf(format, args...), Reset)
}

// PrintDebug prints debug messages with gray color
func PrintDebug(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s] 🔍 %s%s\n",
		Gray, timestamp, fmt.Sprintf(format, args...), Reset)
}

// PrintBanner prints an attractive application banner
func PrintBanner() {
	banner := `
%s%s
██╗     ██╗   ██╗███╗   ██╗ █████╗     ██╗ ██████╗ ████████╗
██║     ██║   ██║████╗  ██║██╔══██╗    ██║██╔═══██╗╚══██╔══╝
██║     ██║   ██║██╔██╗ ██║███████║    ██║██║   ██║   ██║   
██║     ██║   ██║██║╚██╗██║██╔══██║    ██║██║   ██║   ██║   
███████╗╚██████╔╝██║ ╚████║██║  ██║    ██║╚██████╔╝   ██║   
╚══════╝ ╚═════╝ ╚═╝  ╚═══╝╚═╝  ╚═╝    ╚═╝ ╚═════╝    ╚═╝   
%s
          %s🌟 Smart IoT Device Management & Tracking Server 🌟%s
%s`
	fmt.Printf(banner, BrightCyan, Bold, Reset, BrightYellow, Reset, Reset)
}

// PrintEndpoint prints API endpoint information
func PrintEndpoint(method, path, description string) {
	var methodColor string
	switch method {
	case "GET":
		methodColor = BrightGreen
	case "POST":
		methodColor = BrightBlue
	case "PUT":
		methodColor = BrightYellow
	case "DELETE":
		methodColor = BrightRed
	default:
		methodColor = White
	}

	fmt.Printf("  %s%-6s%s %s%-30s%s %s%s%s\n",
		methodColor, method, Reset,
		Cyan, path, Reset,
		Gray, description, Reset)
}

// PrintShutdown prints shutdown message
func PrintShutdown() {
	fmt.Printf("\n%s%s🛑 Luna IoT Server Shutdown Initiated...%s\n", BrightRed, Bold, Reset)
	fmt.Printf("%s%s⏳ Gracefully closing all connections...%s\n", Yellow, Bold, Reset)
	fmt.Printf("%s%s👋 Thank you for using Luna IoT Server!%s\n\n", BrightBlue, Bold, Reset)
}

// PrintStats prints statistics in a formatted way
func PrintStats(label string, value interface{}) {
	fmt.Printf("%s📊 %-20s:%s %s%v%s\n",
		Cyan, label, Reset,
		BrightWhite, value, Reset)
}
