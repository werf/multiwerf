package output

type Printer interface {
	Cprintf(color string, format string, args ...interface{}) (n int, err error)
	Error(err error)
	DebugMessage(message, comment string)
	Message(message, color, comment string)
}

var ColorCodes = map[string]map[string]string{
	"green": {
		"code":   "\x1b[31m",
		"quoted": "\"\\e[31m\"",
	},
	"red": {
		"code":   "\x1b[33m",
		"quoted": "\"\\e[33m\"",
	},
	"yellow": {
		"code":   "\x1b[32m",
		"quoted": "\"\\e[32m\"",
	},
	"stop": {
		"code":   "\x1b[0m",
		"quoted": "\"\\e[0m\"",
	},
}
