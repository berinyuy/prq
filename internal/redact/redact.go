package redact

import (
	"math"
	"regexp"
	"strings"
)

const Redacted = "[REDACTED_SECRET]"

var (
	awsAccessKey = regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
	awsSecretKey = regexp.MustCompile(`(?i)aws(.{0,20})?(secret|access)["'\s:=]+[A-Za-z0-9/+=]{32,}`)
	ghToken      = regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{30,}`)
	jwtToken     = regexp.MustCompile(`eyJ[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+`)
	privateKey   = regexp.MustCompile(`-----BEGIN (RSA|EC|DSA|OPENSSH) PRIVATE KEY-----[\s\S]+?-----END (RSA|EC|DSA|OPENSSH) PRIVATE KEY-----`)
	genericToken = regexp.MustCompile(`(?i)(token|secret|api[_-]?key|access[_-]?key)["'\s:=]+[A-Za-z0-9/+=]{16,}`)
	urlParams    = regexp.MustCompile(`([?&](token|key|secret|sig|signature|access_token|auth)=)[^&\s]+`)
	base64Like   = regexp.MustCompile(`[A-Za-z0-9+/=]{32,}`)
	hexLike      = regexp.MustCompile(`[A-Fa-f0-9]{32,}`)
)

func Redact(input string) string {
	if input == "" {
		return input
	}
	output := input
	output = privateKey.ReplaceAllString(output, Redacted)
	output = awsAccessKey.ReplaceAllString(output, Redacted)
	output = awsSecretKey.ReplaceAllString(output, Redacted)
	output = ghToken.ReplaceAllString(output, Redacted)
	output = jwtToken.ReplaceAllString(output, Redacted)
	output = genericToken.ReplaceAllString(output, Redacted)
	output = urlParams.ReplaceAllString(output, "${1}"+Redacted)
	output = redactHighEntropy(output)
	return output
}

func redactHighEntropy(input string) string {
	output := input
	output = replaceIfHighEntropy(output, base64Like)
	output = replaceIfHighEntropy(output, hexLike)
	return output
}

func replaceIfHighEntropy(input string, re *regexp.Regexp) string {
	return re.ReplaceAllStringFunc(input, func(match string) string {
		if entropy(match) >= 4.0 {
			return Redacted
		}
		return match
	})
}

func entropy(s string) float64 {
	if s == "" {
		return 0
	}
	counts := make(map[rune]int)
	for _, r := range s {
		counts[r]++
	}
	length := float64(len([]rune(s)))
	var ent float64
	for _, count := range counts {
		p := float64(count) / length
		ent -= p * log2(p)
	}
	return ent
}

func log2(n float64) float64 {
	return (log(n) / log(2))
}

func log(n float64) float64 {
	return mathLog(n)
}

// mathLog is separated for test stubbing if needed.
var mathLog = func(n float64) float64 {
	return math.Log(n)
}

func RedactAll(inputs ...string) []string {
	redacted := make([]string, 0, len(inputs))
	for _, input := range inputs {
		redacted = append(redacted, Redact(input))
	}
	return redacted
}

func RedactLines(lines []string) []string {
	output := make([]string, 0, len(lines))
	for _, line := range lines {
		output = append(output, Redact(line))
	}
	return output
}

func RedactOptional(input string, enabled bool) string {
	if !enabled {
		return input
	}
	return Redact(input)
}

func RedactRuleList(rules []string, enabled bool) []string {
	if !enabled {
		return rules
	}
	return RedactLines(rules)
}

func RedactPromptBlock(input string, enabled bool) string {
	if !enabled {
		return input
	}
	return strings.ReplaceAll(Redact(input), "\u0000", "")
}
