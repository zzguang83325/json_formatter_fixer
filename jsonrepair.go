package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ================================
// CONSTANTS (from const.go)
// ================================

const (
	codeBackslash               = 0x5c // "\"
	codeSlash                   = 0x2f // "/"
	codeAsterisk                = 0x2a // "*"
	codeOpeningBrace            = 0x7b // "{"
	codeClosingBrace            = 0x7d // "}"
	codeOpeningBracket          = 0x5b // "["
	codeClosingBracket          = 0x5d // "]"
	codeOpenParenthesis         = 0x28 // "("
	codeCloseParenthesis        = 0x29 // ")"
	codeSpace                   = 0x20 // " "
	codeNewline                 = 0xa  // "\n"
	codeTab                     = 0x9  // "\t"
	codeReturn                  = 0xd  // "\r"
	codeBackspace               = 0x08 // "\b"
	codeFormFeed                = 0x0c // "\f"
	codeDoubleQuote             = 0x22 // "
	codePlus                    = 0x2b // "+"
	codeMinus                   = 0x2d // "-"
	codeQuote                   = 0x27 // "'"
	codeZero                    = 0x30 // "0"
	codeNine                    = 0x39 // "9"
	codeComma                   = 0x2c // ","
	codeDot                     = 0x2e // "." (dot, period)
	codeColon                   = 0x3a // ":"
	codeEqual                   = 0x3d // "="
	codeSemicolon               = 0x3b // ";"
	codeUppercaseA              = 0x41 // "A"
	codeLowercaseA              = 0x61 // "a"
	codeUppercaseE              = 0x45 // "E"
	codeLowercaseE              = 0x65 // "e"
	codeUppercaseF              = 0x46 // "F"
	codeLowercaseF              = 0x66 // "f"
	codeNonBreakingSpace        = 0xa0
	codeEnQuad                  = 0x2000
	codeHairSpace               = 0x200a
	codeNarrowNoBreakSpace      = 0x202f
	codeMediumMathematicalSpace = 0x205f
	codeIdeographicSpace        = 0x3000
	codeOghamSpaceMark          = 0x1680
	codeDoubleQuoteLeft         = 0x201c // “
	codeDoubleQuoteRight        = 0x201d // ”
	codeQuoteLeft               = 0x2018 // ‘
	codeQuoteRight              = 0x2019 // ’
	codeGraveAccent             = 0x60   // `
	codeAcuteAccent             = 0xb4   // ´
)

// ================================
// VARIABLES (from const.go, errors.go, utils.go)
// ================================

// Define control and escape character mappings according to JSON standard (RFC 8259)
var controlCharacters = map[rune]string{
	codeBackspace: `\b`,
	codeFormFeed:  `\f`,
	codeNewline:   `\n`,
	codeReturn:    `\r`,
	codeTab:       `\t`,
}

// JSON standard escape characters - these MUST be escaped or CAN be escaped in JSON strings
var escapeCharacters = map[rune]string{
	'"':  "\"", // MUST be escaped
	'\\': "\\", // MUST be escaped
	'/':  "/",  // CAN be escaped (optional)
	'b':  "\b", // Backspace control character
	'f':  "\f", // Form feed control character
	'n':  "\n", // Newline control character
	'r':  "\r", // Carriage return control character
	't':  "\t", // Tab control character
	'u':  "u",  // Unicode escape sequence
}

// Predefined error variables for use with errors.Is()
var (
	ErrUnexpectedEnd       = errors.New("unexpected end of json string")
	ErrObjectKeyExpected   = errors.New("object key expected")
	ErrColonExpected       = errors.New("colon expected")
	ErrInvalidCharacter    = errors.New("invalid character")
	ErrUnexpectedCharacter = errors.New("unexpected character")
	ErrInvalidUnicode      = errors.New("invalid unicode character")
)

// URL-related regular expressions and functions
var regexURLStart = regexp.MustCompile(`^(https?|ftp|mailto|file|data|irc)://`)
var regexURLChar = regexp.MustCompile(`^[A-Za-z0-9\-._~:/?#@!$&'()*+;=]$`)

// Regular expression cache for improved performance
var (
	driveLetterRe        = regexp.MustCompile(`^[A-Za-z]:\\`)
	containsDriveRe      = regexp.MustCompile(`[A-Za-z]:\\`)
	base64Re             = regexp.MustCompile(`^[A-Za-z0-9+/=]{20,}$`)
	fileExtensionRe      = regexp.MustCompile(`(?i)\.[a-z0-9]{2,5}(\?|$|\\|"|/)`)
	unicodeEscapeRe      = regexp.MustCompile(`\\u[0-9a-fA-F]{4}`)
	urlEncodingRe        = regexp.MustCompile(`%[0-9a-fA-F]{2}`)
	regexStartOfValue    = regexp.MustCompile(`^[{[\w-]$`)
	trailingWhitespaceRe = regexp.MustCompile(`(?:\\[ntrfb]| )+$`)
	stringTrimRe         = regexp.MustCompile(`^(?:\\[ntrfb]| )+|(?:\\[ntrfb]| )+$`)
)

// windowsPathPatterns contains common Windows directory patterns for path detection.
var windowsPathPatterns = []string{
	"program files", "system32", "windows\\", "programdata",
	"users\\", "documents", "desktop", "downloads", "music", "pictures", "videos", "appdata", "roaming", "public",
	"temp\\", "fonts", "startup", "sendto", "recent", "nethood", "cookies", "cache", "history", "favorites", "templates",
}

// unixPathPatterns contains common Unix/macOS directory patterns for path detection.
var unixPathPatterns = []string{
	"/bin/", "/etc/", "/var/", "/usr/", "/opt/", "/home/", "/tmp/", "/lib/", "/lib64/",
	"/proc/", "/dev/", "/sys/", "/run/", "/srv/", "/mnt/", "/media/", "/boot/", "/snap/",
	"/usr/share/", "/usr/local/", "/usr/src/", "/var/log/", "/var/lib/", "/var/cache/", "/var/spool/",
	"/Applications/", "/Library/", "/System/", "/Users/",
}

// commonFileExtensions contains file extensions commonly found in file paths.
var commonFileExtensions = []string{
	".config", ".cfg", ".ini", ".conf", ".properties", ".toml",
	".json", ".xml", ".yml", ".yaml", ".csv", ".tsv",
	".backup", ".bak", ".old", ".tmp", ".temp", ".swp", ".~",
	".log", ".out", ".err", ".debug", ".trace",
	".db", ".sqlite", ".sqlite3", ".mdb",
	".txt", ".md", ".readme", ".doc", ".docx", ".pdf",
	".zip", ".tar", ".gz", ".rar", ".7z", ".bz2", ".xz",
	".js", ".ts", ".py", ".go", ".java", ".cpp", ".c", ".h", ".cs", ".php", ".rb", ".rs",
	".mp3", ".mp4", ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".ico",
	".dat", ".bin", ".raw", ".dump",
}

// ================================
// TYPES (from errors.go)
// ================================

// Error represents a structured JSON repair error.
type Error struct {
	Message  string
	Position int
	Err      error // optional underlying error
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s at position %d: %v", e.Message, e.Position, e.Err)
	}
	return fmt.Sprintf("%s at position %d", e.Message, e.Position)
}

// Unwrap allows Error to support errors.Is / errors.As
func (e *Error) Unwrap() error {
	return e.Err
}

// ================================
// PUBLIC API
// ================================

// JSONRepair attempts to repair the given JSON string and returns the repaired version.
func JSONRepair(text string, trimWhitespace bool) (string, error) {
	if len(text) == 0 {
		return "", newUnexpectedEndError(0)
	}

	runes := []rune(text)
	i := 0
	var output strings.Builder

	parseMarkdownCodeBlock(&runes, &i, []string{"```", "[```", "{```"}, &output, trimWhitespace)

	success, err := parseValue(&runes, &i, &output, trimWhitespace)
	if err != nil {
		return "", err
	}
	if !success {
		return "", newUnexpectedEndError(len(runes))
	}

	parseMarkdownCodeBlock(&runes, &i, []string{"```", "```]", "```}"}, &output, trimWhitespace)

	return output.String(), nil
}

// ================================
// PARSING FUNCTIONS
// ================================

func parseValue(text *[]rune, i *int, output *strings.Builder, trimWhitespace bool) (bool, error) {
	parseWhitespaceAndSkipComments(text, i, output, true)

	iBeforeObj := *i
	oBeforeObj := output.Len()
	if processedObj, err := parseObject(text, i, output, trimWhitespace); err != nil {
		return false, err
	} else if processedObj {
		parseWhitespaceAndSkipComments(text, i, output, true)
		return true, nil
	}
	*i = iBeforeObj
	outputStr := output.String()
	if len(outputStr) > oBeforeObj {
		output.Reset()
		output.WriteString(outputStr[:oBeforeObj])
	}

	iBeforeMongo := *i
	oBeforeMongo := output.Len()
	if *i < len(*text) && isFunctionNameCharStart((*text)[*i]) {
		j := *i
		for j < len(*text) && isFunctionNameChar((*text)[j]) {
			j++
		}
		name := string((*text)[*i:j])
		if name == "ObjectId" || name == "NumberLong" || name == "NumberInt" || name == "ISODate" || name == "BinData" {
			k := j
			for k < len(*text) && isWhitespace((*text)[k]) {
				k++
			}
			if k >= len(*text) || (*text)[k] != codeOpenParenthesis {
				*i = k
				var innerValue strings.Builder
				if parseUnquotedString(text, i, &innerValue, trimWhitespace) {
					fmt.Fprintf(output, `"%s"`, name)
					val := innerValue.String()
					if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
						output.WriteString("(")
						output.WriteString(val)
						output.WriteString(")")
					} else {
						fmt.Fprintf(output, `("%s")`, val)
					}
					return true, nil
				}
			}
		}
	}
	*i = iBeforeMongo
	output.Reset()
	output.WriteString(outputStr[:oBeforeMongo])

	processed, err := parseArray(text, i, output, trimWhitespace)
	if err != nil {
		return false, err
	}
	if !processed {
		stringProcessed, err := parseString(text, i, output, false, -1, trimWhitespace)
		if err != nil {
			return false, err
		}
		processed = stringProcessed
		if !processed {
			processed = parseNumber(text, i, output) ||
				parseKeywords(text, i, output) ||
				parseUnquotedString(text, i, output, trimWhitespace) ||
				parseRegex(text, i, output)
		}
	}
	parseWhitespaceAndSkipComments(text, i, output, true)

	return processed, nil
}

func parseWhitespaceAndSkipComments(text *[]rune, i *int, output *strings.Builder, skipNewline bool) bool {
	start := *i
	parseWhitespace(text, i, output, skipNewline)
	for {
		changed := parseComment(text, i)
		if changed {
			changed = parseWhitespace(text, i, output, skipNewline)
		}
		if !changed {
			break
		}
	}
	return *i > start
}

func parseWhitespace(text *[]rune, i *int, output *strings.Builder, skipNewline bool) bool {
	start := *i
	whitespace := strings.Builder{}
	isW := isWhitespace
	if !skipNewline {
		isW = isWhitespaceExceptNewline
	}
	for *i < len(*text) && (isW((*text)[*i]) || isSpecialWhitespace((*text)[*i])) {
		if !isSpecialWhitespace((*text)[*i]) {
			whitespace.WriteRune((*text)[*i])
		} else {
			whitespace.WriteRune(' ') // repair special whitespace
		}
		*i++
	}
	if whitespace.Len() > 0 {
		output.WriteString(whitespace.String())
		return true
	}
	return *i > start
}

func parseComment(text *[]rune, i *int) bool {
	if *i+1 < len(*text) {
		if (*text)[*i] == codeSlash && (*text)[*i+1] == codeAsterisk {
			for *i < len(*text) && !atEndOfBlockComment(text, i) {
				*i++
			}
			if *i+2 <= len(*text) {
				*i += 2
			}
			return true
		} else if (*text)[*i] == codeSlash && (*text)[*i+1] == codeSlash {
			if *i > 0 && (*text)[*i-1] == codeColon {
				j := *i - 2
				for j >= 0 && (isLetter((*text)[j])) {
					j--
				}
				protocol := string((*text)[j+1 : *i-1])
				if protocol == "http" || protocol == "https" || protocol == "ftp" {
					return false
				}
			}
			for *i < len(*text) && (*text)[*i] != codeNewline && (*text)[*i] != codeReturn {
				*i++
			}
			return true
		}
	}
	return false
}

func lookAheadForColon(text *[]rune, i int) bool {
	j := i
	if j < len(*text) && ((*text)[j] == codeNewline || (*text)[j] == codeReturn) {
		j++
	}
	for j < len(*text) {
		if isWhitespace((*text)[j]) || isSpecialWhitespace((*text)[j]) {
			j++
			continue
		}
		if (*text)[j] == codeSlash && j+1 < len(*text) {
			if (*text)[j+1] == codeSlash {
				j += 2
				for j < len(*text) && (*text)[j] != codeNewline && (*text)[j] != codeReturn {
					j++
				}
				continue
			}
			if (*text)[j+1] == codeAsterisk {
				j += 2
				for j+1 < len(*text) && !((*text)[j] == codeAsterisk && (*text)[j+1] == codeSlash) {
					j++
				}
				if j+1 < len(*text) {
					j += 2
				}
				continue
			}
		}
		break
	}
	hasKey := false
	for j < len(*text) && !isDelimiter((*text)[j]) && !isQuote((*text)[j]) {
		if (*text)[j] == codeColon || (*text)[j] == codeEqual {
			return hasKey
		}
		if !isWhitespace((*text)[j]) {
			hasKey = true
		}
		j++
	}
	if j < len(*text) && isQuote((*text)[j]) {
		j++
		for j < len(*text) && !isQuote((*text)[j]) {
			j++
		}
		if j < len(*text) && isQuote((*text)[j]) {
			j++
			for j < len(*text) && isWhitespace((*text)[j]) {
				j++
			}
			if j < len(*text) && (*text)[j] == codeColon {
				return true
			}
		}
	}
	return false
}

func parseCharacter(text *[]rune, i *int, output *strings.Builder, code rune) bool {
	if *i < len(*text) && (*text)[*i] == code {
		output.WriteRune((*text)[*i])
		*i++
		return true
	}
	return false
}

func skipCharacter(text *[]rune, i *int, code rune) bool {
	if *i < len(*text) && (*text)[*i] == code {
		*i++
		return true
	}
	return false
}

func skipEscapeCharacter(text *[]rune, i *int) bool {
	return skipCharacter(text, i, codeBackslash)
}

func skipEllipsis(text *[]rune, i *int, output *strings.Builder) bool {
	parseWhitespaceAndSkipComments(text, i, output, true)
	if *i+2 < len(*text) &&
		(*text)[*i] == codeDot &&
		(*text)[*i+1] == codeDot &&
		(*text)[*i+2] == codeDot {
		*i += 3
		parseWhitespaceAndSkipComments(text, i, output, true)
		skipCharacter(text, i, codeComma)
		return true
	}
	return false
}

func parseObject(text *[]rune, i *int, output *strings.Builder, trimWhitespace bool) (bool, error) {
	if *i >= len(*text) {
		return false, nil
	}
	if (*text)[*i] == codeOpeningBrace {
		output.WriteRune((*text)[*i])
		*i++
	} else {
		iBefore := *i
		var tempOutput strings.Builder
		parseWhitespaceAndSkipComments(text, i, &tempOutput, true)
		stringProcessed, _ := parseString(text, i, &tempOutput, false, -1, trimWhitespace)
		processedKey := stringProcessed || parseUnquotedStringWithMode(text, i, &tempOutput, true, trimWhitespace)
		if processedKey {
			key := strings.Trim(strings.TrimSpace(tempOutput.String()), "\"")
			// Strictness check for braceless objects:
			// If it's an unquoted key, don't allow it if it looks like HTML or a long sentence
			if !stringProcessed {
				if key == "" || key[0] == '<' || strings.Count(key, " ") > 1 {
					*i = iBefore
					return false, nil
				}
			}
			if (key == "http" || key == "https" || key == "ftp") &&
				*i < len(*text) && (*text)[*i] == codeColon &&
				*i+2 < len(*text) && (*text)[*i+1] == codeSlash && (*text)[*i+2] == codeSlash {
				*i = iBefore
				return false, nil
			}
			parseWhitespaceAndSkipComments(text, i, &tempOutput, true)
			if *i < len(*text) && ((*text)[*i] == codeColon || (*text)[*i] == codeEqual) {
				output.WriteRune('{')
				*i = iBefore
			} else {
				*i = iBefore
				return false, nil
			}
		} else {
			*i = iBefore
			return false, nil
		}
	}
	parseWhitespaceAndSkipComments(text, i, output, true)
	if skipCharacter(text, i, codeComma) {
		parseWhitespaceAndSkipComments(text, i, output, true)
	}
	initial := true
	for {
		parseWhitespaceAndSkipComments(text, i, output, true)
		if *i >= len(*text) || (*text)[*i] == codeClosingBrace {
			break
		}

		if !initial {
			iBefore := *i
			oBefore := output.Len()
			processedComma := parseCharacter(text, i, output, codeComma)
			if processedComma {
				parseWhitespaceAndSkipComments(text, i, output, true)
				for skipCharacter(text, i, codeComma) {
					parseWhitespaceAndSkipComments(text, i, output, true)
				}
				temp := output.String()
				if strings.HasSuffix(temp, ",") {
					temp = temp[:len(temp)-1]
					temp = insertBeforeLastWhitespace(temp, ",")
					if idx := strings.LastIndex(temp, "\n"); idx != -1 {
						j := idx + 1
						for j < len(temp) && (temp[j] == ' ' || temp[j] == '\t') {
							j++
						}
						if j == len(temp) {
							temp = temp[:idx+1]
						}
					}
					output.Reset()
					output.WriteString(temp)
				}
			} else {
				// Check if we're at the start of a new key without a comma
				isNewKey := false
				j := *i
				parseWhitespaceAndSkipComments(text, &j, &strings.Builder{}, true)
				if j < len(*text) {
					char := (*text)[j]
					if isQuote(char) || isLetter(char) {
						isNewKey = true
					}
				}

				if isNewKey {
					outputStr := insertBeforeLastWhitespace(output.String(), ",")
					output.Reset()
					output.WriteString(outputStr)
				} else {
					*i = iBefore
					tempStr := output.String()[:oBefore]
					output.Reset()
					output.WriteString(insertBeforeLastWhitespace(tempStr, ","))
				}
			}
		} else {
			initial = false
		}
		skipEllipsis(text, i, output)
		iKeyStart := *i
		var keyOutput strings.Builder
		stringProcessed, err := parseString(text, i, &keyOutput, false, -1, trimWhitespace)
		if err != nil {
			return false, err
		}
		processedKey := stringProcessed || parseUnquotedStringWithMode(text, i, &keyOutput, true, trimWhitespace)
		if !processedKey {
			// Check if we have a stray comma before a closing brace
			outputStr := output.String()
			lastCommaIdx := strings.LastIndex(outputStr, ",")
			if lastCommaIdx != -1 {
				contentAfterLastComma := outputStr[lastCommaIdx+1:]
				if strings.TrimSpace(contentAfterLastComma) == "" {
					output.Reset()
					output.WriteString(outputStr[:lastCommaIdx] + contentAfterLastComma)
				}
			}

			// If we just skipped a comma, we might be at the end of the object
			j := *i
			parseWhitespaceAndSkipComments(text, &j, &strings.Builder{}, true)
			if j < len(*text) && (*text)[j] == codeClosingBrace {
				*i = j
				break
			}
		}

		if processedKey {
			key := keyOutput.String()
			if trimWhitespace {
				// Remove quotes, trim, then re-add quotes
				if strings.HasPrefix(key, "\"") && strings.HasSuffix(key, "\"") {
					inner := key[1 : len(key)-1]
					inner = stringTrimRe.ReplaceAllString(inner, "")
					key = "\"" + inner + "\""
				} else {
					key = stringTrimRe.ReplaceAllString(key, "")
				}
			}
			keyTrimmed := strings.Trim(strings.TrimSpace(key), "\"")
			if (keyTrimmed == "http" || keyTrimmed == "https" || keyTrimmed == "ftp") &&
				*i < len(*text) && (*text)[*i] == codeColon &&
				*i+2 < len(*text) && (*text)[*i+1] == codeSlash && (*text)[*i+2] == codeSlash {
				*i = iKeyStart
				return false, nil
			}
			output.WriteString(key)
		}
		if !processedKey {
			if *i >= len(*text) ||
				(*text)[*i] == codeClosingBrace ||
				(*text)[*i] == codeOpeningBrace ||
				(*text)[*i] == codeClosingBracket ||
				(*text)[*i] == codeOpeningBracket ||
				(*text)[*i] == 0 {
				outputStr := stripLastOccurrence(output.String(), ",", false)
				output.Reset()
				output.WriteString(outputStr)
			} else {
				return false, newObjectKeyExpectedError(*i)
			}
			break
		}
		parseWhitespaceAndSkipComments(text, i, output, true)
		for parseComment(text, i) {
			parseWhitespaceAndSkipComments(text, i, output, true)
		}
		parseWhitespaceAndSkipComments(text, i, output, true)
		iBeforeColon := *i
		processedColon := parseCharacter(text, i, output, codeColon)
		if !processedColon {
			// Check if we have a colon after some whitespace
			j := *i
			parseWhitespaceAndSkipComments(text, &j, &strings.Builder{}, true)
			for parseComment(text, &j) {
				parseWhitespaceAndSkipComments(text, &j, &strings.Builder{}, true)
			}
			if j < len(*text) && (*text)[j] == codeColon {
				*i = j
				processedColon = parseCharacter(text, i, output, codeColon)
			} else {
				// Special case: "name" "value" (missing colon)
				// Look ahead to see if there's a value starting
				k := iBeforeColon
				parseWhitespaceAndSkipComments(text, &k, &strings.Builder{}, true)
				for parseComment(text, &k) {
					parseWhitespaceAndSkipComments(text, &k, &strings.Builder{}, true)
				}
				if k < len(*text) && (isQuote((*text)[k]) || isLetter((*text)[k]) || isDigit((*text)[k]) || (*text)[k] == codeOpeningBrace || (*text)[k] == codeOpeningBracket) {
					outputStr := insertBeforeLastWhitespace(output.String(), ":")
					output.Reset()
					output.WriteString(outputStr)
					processedColon = true
					*i = k
				}
			}
		}
		if processedColon {
			parseWhitespaceAndSkipComments(text, i, output, true)
			for skipCharacter(text, i, codeColon) {
				parseWhitespaceAndSkipComments(text, i, output, true)
			}
		}
		if !processedColon && skipCharacter(text, i, codeEqual) {
			outputStr := insertBeforeLastWhitespace(output.String(), ":")
			output.Reset()
			output.WriteString(outputStr)
			processedColon = true
		}
		truncatedText := *i >= len(*text)
		if !processedColon {
			if truncatedText {
				outputStr := insertBeforeLastWhitespace(output.String(), ":")
				output.Reset()
				output.WriteString(outputStr)
				processedColon = true
			} else {
				return false, newColonExpectedError(*i)
			}
		}
		for parseComment(text, i) {
		}
		parseWhitespaceAndSkipComments(text, i, output, true)
		processedValue, err := parseValue(text, i, output, trimWhitespace)
		if err != nil {
			return false, err
		}
		if !processedValue {
			if processedColon || truncatedText {
				output.WriteString("null")
			} else {
				return false, nil
			}
		}
		for parseComment(text, i) {
		}
		parseWhitespaceAndSkipComments(text, i, output, true)
	}
	if *i < len(*text) && (*text)[*i] == codeClosingBrace {
		output.WriteRune((*text)[*i])
		*i++
	} else {
		outputStr := insertBeforeLastWhitespace(output.String(), "}")
		output.Reset()
		output.WriteString(outputStr)
	}
	return true, nil
}

func parseArray(text *[]rune, i *int, output *strings.Builder, trimWhitespace bool) (bool, error) {
	if *i >= len(*text) {
		return false, nil
	}
	if (*text)[*i] == codeOpeningBracket {
		output.WriteRune((*text)[*i])
		*i++
		parseWhitespaceAndSkipComments(text, i, output, true)
		initial := true
		for *i < len(*text) && (*text)[*i] != codeClosingBracket {
			if !initial {
				iBefore := *i
				oBefore := output.Len()
				parseWhitespaceAndSkipComments(text, i, output, true)

				// Before checking for comma, check if we're at the start of a new value
				// without a comma (missing comma)
				isNewValue := false
				if *i < len(*text) {
					char := (*text)[*i]
					// If it's a quote, brace, bracket, number, or start of unquoted string
					if isQuote(char) || char == codeOpeningBrace || char == codeOpeningBracket ||
						isDigit(char) || char == codeMinus || char == codePlus ||
						isLetter(char) {
						isNewValue = true
					}
				}

				processedComma := parseCharacter(text, i, output, codeComma)
				if !processedComma {
					if isNewValue {
						// Missing comma between array elements
						outputStr := insertBeforeLastWhitespace(output.String(), ",")
						output.Reset()
						output.WriteString(outputStr)
					} else {
						*i = iBefore
						tempStr := output.String()
						output.Reset()
						output.WriteString(tempStr[:oBefore])
						outputStr := insertBeforeLastWhitespace(output.String(), ",")
						output.Reset()
						output.WriteString(outputStr)
					}
				} else {
					for {
						iBeforeExtra := *i
						oBeforeExtra := output.Len()
						parseWhitespaceAndSkipComments(text, i, output, true)
						if parseCharacter(text, i, output, codeComma) {
							outputStr := output.String()
							lastCommaIdx := strings.LastIndex(outputStr, ",")
							if lastCommaIdx != -1 {
								j := *i
								parseWhitespaceAndSkipComments(text, &j, &strings.Builder{}, true)
								if j < len(*text) && (*text)[j] == codeClosingBracket {
									newOutput := outputStr[:lastCommaIdx]
									output.Reset()
									output.WriteString(newOutput)
								} else {
									newOutput := outputStr[:lastCommaIdx] + "null" + outputStr[lastCommaIdx:]
									output.Reset()
									output.WriteString(newOutput)
								}
							}
						} else {
							*i = iBeforeExtra
							currentOutput := output.String()
							output.Reset()
							output.WriteString(currentOutput[:oBeforeExtra])
							break
						}
					}
				}
			} else {
				initial = false
				if skipCharacter(text, i, codeComma) {
					output.WriteString("null,")
				}
			}
			parseWhitespaceAndSkipComments(text, i, output, true)
			skipEllipsis(text, i, output)

			// Before parsing value, check if this is actually a key for an outer object
			if !initial {
				j := *i
				parseWhitespaceAndSkipComments(text, &j, &strings.Builder{}, true)
				if j < len(*text) {
					iTemp := j
					var keyTemp strings.Builder
					stringProcessed, _ := parseString(text, &iTemp, &keyTemp, false, -1, trimWhitespace)
					processedKey := stringProcessed || parseUnquotedStringWithMode(text, &iTemp, &keyTemp, true, trimWhitespace)

					isOuterElement := false
					if processedKey {
						parseWhitespaceAndSkipComments(text, &iTemp, &strings.Builder{}, true)
						if iTemp < len(*text) && ((*text)[iTemp] == codeColon || (*text)[iTemp] == codeEqual) {
							isOuterElement = true
						}
					} else if (*text)[j] == codeClosingBrace {
						isOuterElement = true
					}

					if isOuterElement {
						outputStr := output.String()
						lastCommaIdx := strings.LastIndex(outputStr, ",")
						if lastCommaIdx != -1 {
							contentAfterLastComma := outputStr[lastCommaIdx+1:]
							if strings.TrimSpace(contentAfterLastComma) == "" {
								output.Reset()
								output.WriteString(outputStr[:lastCommaIdx] + contentAfterLastComma)
							}
						}
						break
					}
				}
			}

			processedValue, err := parseValue(text, i, output, trimWhitespace)
			if err != nil {
				return false, err
			}
			if !processedValue {
				outputStr := output.String()
				lastCommaIdx := strings.LastIndex(outputStr, ",")
				if lastCommaIdx != -1 {
					contentAfterLastComma := outputStr[lastCommaIdx+1:]
					if strings.TrimSpace(contentAfterLastComma) == "" {
						output.Reset()
						output.WriteString(outputStr[:lastCommaIdx] + contentAfterLastComma)
					}
				}
				break
			}
		}
		if *i < len(*text) && (*text)[*i] == codeClosingBracket {
			output.WriteRune((*text)[*i])
			*i++
		} else {
			outputStr := insertBeforeLastWhitespace(output.String(), "]")
			output.Reset()
			output.WriteString(outputStr)
		}
		return true, nil
	}
	return false, nil
}

func parseNewlineDelimitedJSON(text *[]rune, i *int, output *strings.Builder, trimWhitespace bool) {
	initial := true
	processedValue := true
	for processedValue {
		if !initial {
			output.WriteRune('\n')
		} else {
			initial = false
		}
		var lineOutput strings.Builder
		var err error
		processedValue, err = parseValue(text, i, &lineOutput, trimWhitespace)
		if err != nil {
			break
		}
		if processedValue {
			output.WriteString(lineOutput.String())
		}
		for *i < len(*text) && (*text)[*i] != codeNewline && (*text)[*i] != codeReturn {
			*i++
		}
		for *i < len(*text) && ((*text)[*i] == codeNewline || (*text)[*i] == codeReturn) {
			*i++
		}
	}
}

func parseString(text *[]rune, i *int, output *strings.Builder, stopAtDelimiter bool, stopAtIndex int, trimWhitespace bool) (bool, error) {
	if *i >= len(*text) {
		return false, nil
	}
	char := (*text)[*i]
	if isQuote(char) {
		isEndQuote := isDoubleQuote
		if isSingleQuote(char) {
			isEndQuote = isSingleQuote
		} else if isDoubleQuoteLike(char) {
			isEndQuote = isDoubleQuoteLike
		} else if isSingleQuoteLike(char) {
			isEndQuote = isSingleQuoteLike
		}

		if *i+1 < len(*text) {
			otherQuotes := []func(rune) bool{isDoubleQuote, isSingleQuote, isDoubleQuoteLike, isSingleQuoteLike}
			bestK := -1
			var bestQuoteFunc func(rune) bool
			for _, quoteFunc := range otherQuotes {
				k := *i + 1
				for k < len(*text) && !quoteFunc((*text)[k]) && (*text)[k] != codeNewline && (*text)[k] != codeReturn {
					k++
				}
				if k < len(*text) && quoteFunc((*text)[k]) {
					nextIdx := k + 1
					for nextIdx < len(*text) && isWhitespace((*text)[nextIdx]) {
						nextIdx++
					}
					if nextIdx < len(*text) && ((*text)[nextIdx] == codeColon || (*text)[nextIdx] == codeEqual) {
						if bestK == -1 || k < bestK {
							bestK = k
							bestQuoteFunc = quoteFunc
						}
					}
				}
			}
			if bestK != -1 {
				isEndQuote = bestQuoteFunc
			}
		}

		*i++
		isFilePath := analyzePotentialFilePath(text, *i-1)
		var str strings.Builder
		for *i < len(*text) {
			currentChar := (*text)[*i]
			if isQuote(currentChar) {
				// Potential end quote. Check if it's followed by a delimiter.
				isRealEndQuote := false
				isOfficialEndQuote := isEndQuote(currentChar)

				j := *i + 1
				// Skip whitespace and comments
				parseWhitespaceAndSkipComments(text, &j, &strings.Builder{}, true)

				if j >= len(*text) {
					isRealEndQuote = true
				} else {
					nextChar := (*text)[j]
					if nextChar == codeComma || nextChar == codeClosingBrace || nextChar == codeClosingBracket ||
						nextChar == codeColon || nextChar == codeEqual || nextChar == codePlus {
						isRealEndQuote = true
					} else if isQuote(nextChar) || isLetter(nextChar) || isDigit(nextChar) {
						// Special case: "Basketball" "Swimming" (missing comma between array elements)
						// Or "name" "value" (missing colon between key and value)
						// If the next character is a quote, letter, or digit, it's likely a missing delimiter.
						isRealEndQuote = true
					}
				}

				if isRealEndQuote && !isOfficialEndQuote {
					// Mismatched quote.
					// Check if the official end quote exists later on the same line.
					for k := *i + 1; k < len(*text) && (*text)[k] != codeNewline && (*text)[k] != codeReturn; k++ {
						if isEndQuote((*text)[k]) {
							isRealEndQuote = false
							break
						}
					}
				}

				if isRealEndQuote {
					*i++
					if parseConcatenatedString(text, i, output, trimWhitespace) {
						str.WriteString(output.String())
						output.Reset()
						continue
					}

					finalStr := str.String()
					if isNumericString(finalStr) {
						output.WriteString(finalStr)
					} else {
						fmt.Fprintf(output, `"%s"`, finalStr)
					}
					return true, nil
				} else {
					// Not a real end quote, escape it and continue
					if currentChar == '"' {
						str.WriteString("\\\"")
					} else {
						str.WriteRune(currentChar)
					}
					*i++
					continue
				}
			} else if (*text)[*i] == codeBackslash {
				if isFilePath {
					str.WriteString("\\\\")
					*i++
					continue
				}
				*i++
				if *i < len(*text) {
					char := (*text)[*i]
					if _, ok := escapeCharacters[char]; ok {
						str.WriteString("\\")
						str.WriteRune(char)
						if char == 'u' {
							// Check if we have 4 hex digits
							hexCount := 0
							for j := 0; j < 4 && *i+1 < len(*text) && isHex((*text)[*i+1]); j++ {
								*i++
								str.WriteRune((*text)[*i])
								hexCount++
							}
							// If not 4 hex digits, we should probably escape the backslash and treat u as normal char
							if hexCount < 4 {
								currentStr := str.String()
								// The last character in str is 'u', and before it is '\'.
								// We need to change "\u..." to "\\u..."
								// Actually, it's easier to just handle it here.
								// If hexCount is 0, we have "\u", we want "\\u"
								// If hexCount is 2, we have "\u26", we want "\\u26"

								// Remove the last 'u' and '\' from str
								tempStr := currentStr[:len(currentStr)-(1+hexCount+1)]
								str.Reset()
								str.WriteString(tempStr)
								str.WriteString("\\\\u")
								// Re-append the hex digits that were already consumed
								str.WriteString(currentStr[len(currentStr)-hexCount:])
							}
						}
						*i++
					} else {
						// Not a standard escape character, treat as literal backslash
						str.WriteString("\\\\")
						// Don't consume the next character, let the loop handle it
					}
				} else {
					str.WriteString("\\\\")
				}
			} else {
				char := (*text)[*i]
				if !isFilePath && (char == codeComma || char == codeClosingBrace || char == codeClosingBracket) {
					foundEndQuote := false
					// Check if this delimiter is followed by a valid next item
					if char == codeClosingBrace || char == codeClosingBracket {
						// For closing delimiters, we check if there's an end quote later on the same line
						for k := *i + 1; k < len(*text) && (*text)[k] != codeNewline && (*text)[k] != codeReturn; k++ {
							if isEndQuote((*text)[k]) {
								foundEndQuote = true
								break
							}
						}
					} else if char == codeComma {
						// For comma, we check if it's followed by a valid key:value or value
						nextIdx := *i + 1
						for nextIdx < len(*text) && isWhitespace((*text)[nextIdx]) {
							nextIdx++
						}
						if nextIdx < len(*text) {
							nextChar := (*text)[nextIdx]
							if isQuote(nextChar) {
								// Look ahead for the end of this potential next item
								foundColonAfterQuote := false
								for k := nextIdx + 1; k < len(*text) && (*text)[k] != codeNewline && (*text)[k] != codeReturn; k++ {
									if isQuote((*text)[k]) {
										// Found another quote, check if it's followed by a colon
										n := k + 1
										for n < len(*text) && isWhitespace((*text)[n]) {
											n++
										}
										if n < len(*text) && ((*text)[n] == codeColon || (*text)[n] == codeEqual) {
											foundColonAfterQuote = true
										}
										break
									}
								}
								if foundColonAfterQuote {
									// The quote at nextIdx is the start of a new key
									// So the current delimiter IS a delimiter
									foundEndQuote = false
								} else {
									// The quote at nextIdx is probably the end quote of the current string
									// Or just a quote inside the string.
									// If it's the end quote, then the current delimiter is NOT a delimiter.
									foundEndQuote = true
								}
							} else if isStartOfValue(nextChar) {
								// In an array, a comma followed by a value is likely a delimiter
								// But we need to be careful not to break strings like "a,b"
								// So we check if there's an end quote for the CURRENT string later
								hasEndQuoteLater := false
								for k := *i + 1; k < len(*text) && (*text)[k] != codeNewline && (*text)[k] != codeReturn; k++ {
									if isEndQuote((*text)[k]) {
										hasEndQuoteLater = true
										break
									}
								}
								if hasEndQuoteLater {
									foundEndQuote = true
								} else {
									foundEndQuote = false
								}
							} else {
								// Unquoted key?
								hasColon := false
								for k := nextIdx; k < len(*text) && (*text)[k] != codeNewline && (*text)[k] != codeReturn && !isDelimiter((*text)[k]); k++ {
									if (*text)[k] == codeColon || (*text)[k] == codeEqual {
										hasColon = true
										break
									}
								}
								if hasColon {
									foundEndQuote = false
								} else {
									foundEndQuote = true
								}
							}
						}
					}

					if !foundEndQuote {
						break
					}
				}
				if char == '"' {
					str.WriteString("\\\"")
				} else if isControlCharacter(char) {
					if replacement, ok := controlCharacters[char]; ok {
						str.WriteString(replacement)
					} else {
						str.WriteString(fmt.Sprintf("\\u%04x", char))
					}
				} else {
					str.WriteRune(char)
				}
				*i++
			}
		}
		content := str.String()
		if trimWhitespace {
			content = stringTrimRe.ReplaceAllString(content, "")
		} else {
			content = trailingWhitespaceRe.ReplaceAllString(content, "")
		}
		fmt.Fprintf(output, `"%s"`, content)
		return true, nil
	}
	return false, nil
}

func parseConcatenatedString(text *[]rune, i *int, output *strings.Builder, trimWhitespace bool) bool {
	processed := false
	iBeforeWhitespace := *i
	oBeforeWhitespace := output.Len()
	parseWhitespaceAndSkipComments(text, i, output, true)
	for *i < len(*text) && (*text)[*i] == '+' {
		processed = true
		*i++
		parseWhitespaceAndSkipComments(text, i, output, true)
		outputStr := stripLastOccurrence(output.String(), "\"", true)
		output.Reset()
		output.WriteString(outputStr)
		start := output.Len()
		stringProcessed, err := parseString(text, i, output, false, -1, trimWhitespace)
		if err != nil {
			stringProcessed = false
		}
		if stringProcessed {
			outputStr = output.String()
			if len(outputStr) > start {
				output.Reset()
				output.WriteString(removeAtIndex(outputStr, start, 1))
			}
		} else {
			outputStr = insertBeforeLastWhitespace(output.String(), "\"")
			output.Reset()
			output.WriteString(outputStr)
		}
	}
	if !processed {
		*i = iBeforeWhitespace
		tempStr := output.String()
		output.Reset()
		output.WriteString(tempStr[:oBeforeWhitespace])
	}
	return processed
}

func parseNumber(text *[]rune, i *int, output *strings.Builder) bool {
	start := *i
	if *i < len(*text) && ((*text)[*i] == codeMinus || (*text)[*i] == codePlus) {
		*i++
		if atEndOfNumber(text, i) {
			repairNumberEndingWithNumericSymbol(text, start, i, output)
			return true
		}
		if !isDigit((*text)[*i]) {
			*i = start
			return false
		}
	}
	for *i < len(*text) && isDigit((*text)[*i]) {
		*i++
	}
	if *i < len(*text) && (*text)[*i] == codeDot {
		*i++
		if atEndOfNumber(text, i) {
			repairNumberEndingWithNumericSymbol(text, start, i, output)
			return true
		}
		if !isDigit((*text)[*i]) {
			*i = start
			return false
		}
		for *i < len(*text) && isDigit((*text)[*i]) {
			*i++
		}
	}
	if *i < len(*text) && ((*text)[*i] == codeLowercaseE || (*text)[*i] == codeUppercaseE) {
		*i++
		hasMinus := false
		hasPlus := false
		for *i < len(*text) && ((*text)[*i] == codeMinus || (*text)[*i] == codePlus) {
			if (*text)[*i] == codeMinus {
				hasMinus = true
			} else {
				hasPlus = true
			}
			*i++
		}
		if atEndOfNumber(text, i) {
			repairNumberEndingWithNumericSymbol(text, start, i, output)
			return true
		}
		if !isDigit((*text)[*i]) {
			num := string((*text)[start:*i])
			cleanNum := strings.TrimRight(num, "eE+-")
			repairedNum := cleanNum + "e+0"
			output.WriteString(strings.TrimPrefix(repairedNum, "+"))
			return true
		}
		numSoFar := string((*text)[start:*i])
		eIdx := strings.LastIndexAny(numSoFar, "eE")
		if eIdx != -1 {
			cleanNum := numSoFar[:eIdx+1]
			if hasMinus {
				cleanNum += "-"
			} else if hasPlus {
				cleanNum += "+"
			}
			startOfDigits := *i
			for *i < len(*text) && isDigit((*text)[*i]) {
				*i++
			}
			finalNum := cleanNum + string((*text)[startOfDigits:*i])
			output.WriteString(strings.TrimPrefix(finalNum, "+"))
			return true
		}
	}
	if !atEndOfNumber(text, i) {
		*i = start
		return false
	}
	if *i > start {
		num := string((*text)[start:*i])
		hasInvalidLeadingZero := regexp.MustCompile(`^0\d`).MatchString(num)
		if hasInvalidLeadingZero {
			fmt.Fprintf(output, `"%s"`, num)
		} else {
			if strings.HasPrefix(num, "+") {
				num = num[1:]
			}
			output.WriteString(num)
		}
		return true
	}
	return false
}

func parseKeywords(text *[]rune, i *int, output *strings.Builder) bool {
	return parseKeyword(text, i, output, "true", "true") ||
		parseKeyword(text, i, output, "false", "false") ||
		parseKeyword(text, i, output, "null", "null") ||
		parseKeyword(text, i, output, "True", "true") ||
		parseKeyword(text, i, output, "False", "false") ||
		parseKeyword(text, i, output, "None", "null")
}

func parseKeyword(text *[]rune, i *int, output *strings.Builder, name, value string) bool {
	if len(*text)-*i >= len(name) && string((*text)[*i:*i+len(name)]) == name {
		output.WriteString(value)
		*i += len(name)
		return true
	}
	return false
}

func parseUnquotedString(text *[]rune, i *int, output *strings.Builder, trimWhitespace bool) bool {
	return parseUnquotedStringWithMode(text, i, output, false, trimWhitespace)
}

func parseUnquotedStringWithMode(text *[]rune, i *int, output *strings.Builder, isKey bool, trimWhitespace bool) bool {
	start := *i
	if *i >= len(*text) {
		return false
	}
	if isFunctionNameCharStart((*text)[*i]) {
		for *i < len(*text) && isFunctionNameChar((*text)[*i]) {
			*i++
		}
		j := *i
		for j < len(*text) && isWhitespace((*text)[j]) {
			j++
		}
		if j < len(*text) && (*text)[j] == codeOpenParenthesis {
			*i = j + 1
			_, _ = parseValue(text, i, output, trimWhitespace)
			if *i < len(*text) && (*text)[*i] == codeCloseParenthesis {
				*i++
				if *i < len(*text) && (*text)[*i] == codeSemicolon {
					*i++
				}
			}
			return true
		}
	}
	for *i < len(*text) && !isUnquotedStringDelimiter((*text)[*i]) {
		if isQuote((*text)[*i]) {
			if isKey {
				break
			}
			// If it's a value, we check if it's followed by a delimiter
			j := *i + 1
			for j < len(*text) && isWhitespace((*text)[j]) {
				j++
			}
			if j < len(*text) && (isUnquotedStringDelimiter((*text)[j]) || (*text)[j] == codeColon || (*text)[j] == codeEqual) {
				break
			}
		}
		if isKey && ((*text)[*i] == codeColon || (*text)[*i] == codeEqual) {
			isURLProtocol := false
			if (*text)[*i] == codeColon && *i+2 < len(*text) && (*text)[*i+1] == codeSlash && (*text)[*i+2] == codeSlash {
				protocolStart := *i - 1
				for protocolStart >= start && isLetter((*text)[protocolStart]) {
					protocolStart--
				}
				protocol := string((*text)[protocolStart+1 : *i])
				if protocol == "http" || protocol == "https" || protocol == "ftp" {
					isURLProtocol = true
				}
			}
			if !isURLProtocol {
				break
			}
		}
		if !isKey && ((*text)[*i] == codeColon || (*text)[*i] == codeEqual) {
			if !isURLStart(text, *i) && lookAheadForColon(text, *i) {
				break
			}
		}
		if !isKey && ((*text)[*i] == codeClosingBrace || (*text)[*i] == codeClosingBracket) {
			break
		}
		if (*text)[*i] == codeNewline || (*text)[*i] == codeReturn {
			if isKey {
				break
			}
			if lookAheadForColon(text, *i) {
				break
			}
		}
		if (*text)[*i] == codeSlash && *i+1 < len(*text) && ((*text)[*i+1] == codeSlash || (*text)[*i+1] == codeAsterisk) {
			isURLComment := false
			if *i > 0 && (*text)[*i-1] == codeColon {
				j := *i - 2
				for j >= start && (isLetter((*text)[j])) {
					j--
				}
				protocol := string((*text)[j+1 : *i-1])
				if protocol == "http" || protocol == "https" || protocol == "ftp" {
					isURLComment = true
				}
			}
			if !isURLComment {
				break
			}
		}
		*i++
	}
	if *i > start {
		end := *i
		for end > start && isWhitespace((*text)[end-1]) {
			end--
		}
		symbol := string((*text)[start:end])
		if symbol == "undefined" {
			output.WriteString("null")
		} else {
			repairedSymbol := strings.Builder{}
			for _, char := range symbol {
				if char == '"' || isDoubleQuoteLike(char) {
					repairedSymbol.WriteString("\\\"")
				} else if char == '\\' {
					repairedSymbol.WriteString("\\\\")
				} else if char == '\n' {
					repairedSymbol.WriteString("\\n")
				} else if char == '\r' {
					repairedSymbol.WriteString("\\r")
				} else if char == '\t' {
					repairedSymbol.WriteString("\\t")
				} else {
					repairedSymbol.WriteRune(char)
				}
			}
			content := repairedSymbol.String()
			if trimWhitespace {
				content = stringTrimRe.ReplaceAllString(content, "")
			}
			fmt.Fprintf(output, `"%s"`, content)
		}
		if *i < len(*text) && (*text)[*i] == codeDoubleQuote {
			*i++
		}
		return true
	}
	return false
}

func parseRegex(text *[]rune, i *int, output *strings.Builder) bool {
	if *i < len(*text) && (*text)[*i] == codeSlash {
		start := *i
		*i++
		for *i < len(*text) && ((*text)[*i] != codeSlash || (*text)[*i-1] == codeBackslash) {
			*i++
		}
		if *i < len(*text) && (*text)[*i] == codeSlash {
			*i++
		}
		regexContent := string((*text)[start:*i])
		regexContent = strings.ReplaceAll(regexContent, "\\", "\\\\")
		fmt.Fprintf(output, `"%s"`, regexContent)
		return true
	}
	return false
}

func parseMarkdownCodeBlock(text *[]rune, i *int, blocks []string, output *strings.Builder, trimWhitespace bool) bool {
	if skipMarkdownCodeBlock(text, i, blocks, output) {
		if *i < len(*text) && isFunctionNameCharStart((*text)[*i]) {
			j := *i
			for j < len(*text) && isFunctionNameChar((*text)[j]) {
				j++
			}
			name := string((*text)[*i:j])
			if name == "ObjectId" || name == "NumberLong" || name == "NumberInt" || name == "ISODate" || name == "BinData" {
				k := j
				for k < len(*text) && isWhitespace((*text)[k]) {
					k++
				}
				if k >= len(*text) || (*text)[k] != codeOpenParenthesis {
					*i = k
					var innerValue strings.Builder
					if parseUnquotedString(text, i, &innerValue, trimWhitespace) {
						fmt.Fprintf(output, `"%s"`, name)
						val := innerValue.String()
						if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
							output.WriteString("(")
							output.WriteString(val)
							output.WriteString(")")
						} else {
							fmt.Fprintf(output, `("%s")`, val)
						}
						return true
					}
				}
			}
			for *i < len(*text) && isFunctionNameChar((*text)[*i]) {
				*i++
			}
		}
		for *i < len(*text) && (isWhitespace((*text)[*i]) || isSpecialWhitespace((*text)[*i])) {
			if isWhitespace((*text)[*i]) {
				output.WriteRune((*text)[*i])
			} else {
				output.WriteRune(' ')
			}
			*i++
		}
		return true
	}
	return false
}

// ================================
// HELPER FUNCTIONS (from utils.go, errors.go)
// ================================

func prevNonWhitespaceIndex(text []rune, startIndex int) int {
	prev := startIndex
	for prev >= 0 && isWhitespace(text[prev]) {
		prev--
	}
	return prev
}

func atEndOfBlockComment(text *[]rune, i *int) bool {
	return *i+1 < len(*text) && (*text)[*i] == codeAsterisk && (*text)[*i+1] == codeSlash
}

func atEndOfNumber(text *[]rune, i *int) bool {
	return *i >= len(*text) || isDelimiter((*text)[*i]) || isWhitespace((*text)[*i])
}

func repairNumberEndingWithNumericSymbol(text *[]rune, start int, i *int, output *strings.Builder) {
	output.WriteString(string((*text)[start:*i]) + "0")
}

func stripLastOccurrence(text, textToStrip string, stripRemainingText bool) string {
	index := strings.LastIndex(text, textToStrip)
	if index != -1 {
		if stripRemainingText {
			return text[:index]
		}
		return text[:index] + text[index+len(textToStrip):]
	}
	return text
}

func insertBeforeLastWhitespace(s, textToInsert string) string {
	if len(s) == 0 || !isWhitespace(rune(s[len(s)-1])) {
		return s + textToInsert
	}
	index := len(s) - 1
	for index >= 0 {
		if !isWhitespace(rune(s[index])) {
			break
		}
		index--
	}
	return s[:index+1] + textToInsert + s[index+1:]
}

func removeAtIndex(text string, start, count int) string {
	return text[:start] + text[start+count:]
}

func isHex(code rune) bool {
	return (code >= codeZero && code <= codeNine) ||
		(code >= codeUppercaseA && code <= codeUppercaseF) ||
		(code >= codeLowercaseA && code <= codeLowercaseF)
}

func isDigit(code rune) bool {
	return code >= codeZero && code <= codeNine
}

func isLetter(code rune) bool {
	return (code >= 'a' && code <= 'z') || (code >= 'A' && code <= 'Z')
}

func isValidStringCharacter(char rune) bool {
	return char >= 0x0020
}

func isDelimiter(char rune) bool {
	switch char {
	case ',', ':', '[', ']', '/', '{', '}', '(', ')', '\n':
		return true
	}
	return false
}

func isNumericString(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func isStartOfValue(char rune) bool {
	return regexStartOfValue.MatchString(string(char)) || isQuote(char)
}

func isControlCharacter(code rune) bool {
	return code >= 0 && code <= 0x1f
}

func isWhitespace(code rune) bool {
	return code == codeSpace ||
		code == codeNewline ||
		code == codeTab ||
		code == codeReturn ||
		isSpecialWhitespace(code)
}

func isSpecialWhitespace(code rune) bool {
	return code == codeNonBreakingSpace ||
		(code >= codeEnQuad && code <= codeHairSpace) ||
		code == codeNarrowNoBreakSpace ||
		code == codeMediumMathematicalSpace ||
		code == codeIdeographicSpace ||
		code == codeOghamSpaceMark
}

func isQuote(code rune) bool {
	return isDoubleQuoteLike(code) || isSingleQuoteLike(code)
}

func isDoubleQuoteLike(code rune) bool {
	return code == codeDoubleQuote ||
		code == codeDoubleQuoteLeft ||
		code == codeDoubleQuoteRight
}

func isDoubleQuote(code rune) bool {
	return code == codeDoubleQuote
}

func isSingleQuoteLike(code rune) bool {
	return code == codeQuote ||
		code == codeQuoteLeft ||
		code == codeQuoteRight ||
		code == codeGraveAccent ||
		code == codeAcuteAccent
}

func isSingleQuote(code rune) bool {
	return code == codeQuote
}

func endsWithCommaOrNewline(text string) bool {
	if len(text) == 0 {
		return false
	}
	runes := []rune(text)
	i := len(runes) - 1
	for i >= 0 && (runes[i] == ' ' || runes[i] == '\t' || runes[i] == '\r') {
		i--
	}
	if i < 0 {
		return false
	}
	if runes[i] == ',' || runes[i] == '\n' {
		trimmed := strings.TrimSpace(text)
		if len(trimmed) > 0 && trimmed[len(trimmed)-1] == '"' {
			return regexp.MustCompile(`"[ \t\r]*[,\n][ \t\r]*$`).MatchString(text)
		}
		return true
	}
	return false
}

func isFunctionNameCharStart(code rune) bool {
	return (code >= 'a' && code <= 'z') || (code >= 'A' && code <= 'Z') || code == '_' || code == '$'
}

func isFunctionNameChar(code rune) bool {
	return isFunctionNameCharStart(code) || isDigit(code)
}

func isUnquotedStringDelimiter(char rune) bool {
	switch char {
	case ',', '[', ']', '{', '}', '+', ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

func isWhitespaceExceptNewline(code rune) bool {
	return code == codeSpace || code == codeTab || code == codeReturn
}

func isURLChar(code rune) bool {
	return regexURLChar.MatchString(string(code))
}

func hasExcessiveEscapeSequences(content string) bool {
	if len(content) < 3 {
		return false
	}
	unicodeMatches := unicodeEscapeRe.FindAllString(content, -1)
	if len(unicodeMatches) >= 2 {
		totalUnicodeLength := len(unicodeMatches) * 6
		if float64(totalUnicodeLength)/float64(len(content)) > 0.6 {
			return true
		}
	}
	escapeCount := 0
	for i := 0; i < len(content)-1; i++ {
		if content[i] == '\\' {
			next := content[i+1]
			if next == 'n' || next == 't' || next == 'r' || next == 'b' || next == 'f' || next == '"' || next == '\\' {
				escapeCount++
			}
		}
	}
	if escapeCount > 0 && float64(escapeCount*2)/float64(len(content)) > 0.3 {
		return true
	}
	return false
}

func isLikelyTextBlob(content string) bool {
	if len(content) < 3 {
		return false
	}
	if strings.Contains(content, "  ") {
		return true
	}
	if strings.Contains(content, "\n") || strings.Contains(content, "\t") || strings.Contains(content, "\r") {
		return true
	}
	if strings.Contains(content, ". ") || strings.Contains(content, "! ") || strings.Contains(content, "? ") {
		return true
	}
	spaceCount := strings.Count(content, " ")
	if spaceCount > 5 {
		return true
	}
	if len(content) > 10 && content[0] >= 'A' && content[0] <= 'Z' && spaceCount > 2 {
		lowercaseAfterSpace := 0
		foundSpace := false
		for _, r := range content[1:] {
			if r == ' ' {
				foundSpace = true
			} else if foundSpace && r >= 'a' && r <= 'z' {
				lowercaseAfterSpace++
			}
		}
		if lowercaseAfterSpace >= 3 {
			return true
		}
	}
	return false
}

func isBase64String(content string) bool {
	if len(content) < 20 {
		return false
	}
	return base64Re.MatchString(content)
}

func hasURLEncoding(content string) bool {
	return urlEncodingRe.MatchString(content)
}

func isWindowsAbsolutePath(content string) bool {
	return driveLetterRe.MatchString(content) || containsDriveRe.MatchString(content)
}

func isUNCPath(content string) bool {
	if !strings.HasPrefix(content, `\\`) || strings.HasPrefix(content, `\\\\`) {
		return false
	}
	parts := strings.Split(content, `\`)
	return len(parts) >= 4 && len(parts[2]) > 0 && len(parts[3]) > 0
}

func isUnixAbsolutePath(content string) bool {
	if strings.HasPrefix(content, "~/") {
		return true
	}
	if strings.HasPrefix(content, "/") && len(content) > 1 {
		// Avoid matching things like "/ " or "/)"
		if isWhitespace(rune(content[1])) || content[1] == ')' || content[1] == ']' || content[1] == '}' {
			return false
		}
		// Most absolute paths have at least one more segment or are a common root directory
		if strings.Count(content, "/") >= 2 {
			return true
		}
		// Check for common root directories
		lower := strings.ToLower(content)
		for _, p := range unixPathPatterns {
			if strings.HasPrefix(lower, p) {
				return true
			}
		}
	}
	return false
}

func isURLPath(content string) bool {
	lowerContent := strings.ToLower(content)
	if strings.HasPrefix(lowerContent, "http://") || strings.HasPrefix(lowerContent, "https://") {
		return false
	}
	if strings.HasPrefix(lowerContent, "file://") {
		pathPart := content[7:]
		return len(pathPart) > 1 && hasValidPathStructure(pathPart)
	}
	if strings.HasPrefix(lowerContent, "smb://") {
		pathPart := content[6:]
		return len(pathPart) > 1 && hasValidPathStructure(pathPart)
	}
	if strings.HasPrefix(lowerContent, "ftp://") {
		pathPart := content[6:]
		if slashIndex := strings.Index(pathPart, "/"); slashIndex > 0 {
			actualPath := pathPart[slashIndex:]
			return hasValidPathStructure(actualPath)
		}
	}
	return false
}

func containsPathSeparator(content string) bool {
	return strings.Contains(content, "/") || strings.Contains(content, "\\")
}

func countValidPathSegments(content string, separator string) int {
	parts := strings.Split(content, separator)
	meaningfulParts := 0
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) > 0 && part != "." && part != ".." {
			meaningfulParts++
		}
	}
	return meaningfulParts
}

func hasFileExtension(content string) bool {
	ext := filepath.Ext(content)
	if len(ext) > 1 && len(ext) <= 6 {
		return true
	}
	return fileExtensionRe.MatchString(content)
}

func hasValidPathStructure(pathStr string) bool {
	if len(pathStr) < 2 {
		return false
	}
	if !containsPathSeparator(pathStr) {
		return false
	}
	separator := "/"
	if strings.Contains(pathStr, "\\") {
		separator = "\\"
	}
	meaningfulParts := countValidPathSegments(pathStr, separator)
	if meaningfulParts < 2 {
		return false
	}
	hasExt := hasFileExtension(pathStr)
	if hasExt {
		return true
	}
	if meaningfulParts >= 3 {
		return true
	}
	lowerPath := strings.ToLower(pathStr)
	windowsDirs := []string{
		"program files", "windows", "users", "temp", "system32", "documents", "programdata",
		"desktop", "downloads", "music", "pictures", "videos", "appdata", "roaming", "public",
		"inetpub", "wwwroot", "node_modules", "npm",
	}
	for _, dir := range windowsDirs {
		if strings.Contains(lowerPath, dir) {
			return true
		}
	}
	if strings.HasPrefix(pathStr, "/") {
		unixDirs := []string{
			"/bin/", "/etc/", "/var/", "/usr/", "/opt/", "/home/", "/tmp/", "/lib/",
			"/proc/", "/dev/", "/sys/", "/run/", "/srv/", "/mnt/", "/media/", "/boot/",
			"/Applications/", "/Library/", "/System/", "/Users/",
		}
		for _, dir := range unixDirs {
			if strings.Contains(lowerPath, dir) {
				return true
			}
		}
	}
	return false
}

func isValidPathCharacter(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '/' || r == '\\' || r == ':' || r == '.' ||
		r == '-' || r == '_' || r == ' ' || r == '~'
}

func hasReasonableCharacterDistribution(content string) bool {
	if len(content) == 0 {
		return false
	}
	validChars := 0
	for _, r := range content {
		if isValidPathCharacter(r) {
			validChars++
		}
	}
	return float64(validChars)/float64(len(content)) >= 0.7
}

func matchesWindowsPathPattern(lowerContent, content string) bool {
	for _, pattern := range windowsPathPatterns {
		if strings.Contains(lowerContent, pattern) && containsPathSeparator(content) {
			return true
		}
	}
	return false
}

func matchesUnixPathPattern(lowerContent string) bool {
	for _, pattern := range unixPathPatterns {
		if strings.Contains(lowerContent, pattern) {
			return true
		}
	}
	return false
}

func hasCommonFileExtension(lowerContent string) bool {
	if !containsPathSeparator(lowerContent) {
		return false
	}
	for _, ext := range commonFileExtensions {
		if strings.HasSuffix(lowerContent, ext) {
			return true
		}
	}
	return false
}

func isExcludedURL(lowerContent, content string) bool {
	if strings.HasPrefix(lowerContent, "http://") || strings.HasPrefix(lowerContent, "https://") {
		return true
	}
	if strings.HasPrefix(lowerContent, "ftp://") && !strings.Contains(content[6:], "/") {
		return true
	}
	return false
}

func passesEarlyExclusionFilters(content string) bool {
	if hasExcessiveEscapeSequences(content) {
		return false
	}
	if isLikelyTextBlob(content) {
		return false
	}
	if isBase64String(content) {
		return false
	}
	if hasURLEncoding(content) {
		return false
	}
	return true
}

func matchesAbsolutePathFormat(content string) bool {
	return isURLPath(content) ||
		isWindowsAbsolutePath(content) ||
		isUNCPath(content) ||
		isUnixAbsolutePath(content)
}

func isLikelyFilePath(content string) bool {
	if len(content) < 2 {
		return false
	}
	lowerContent := strings.ToLower(content)
	if isExcludedURL(lowerContent, content) {
		return false
	}
	if matchesAbsolutePathFormat(content) {
		return true
	}
	if !passesEarlyExclusionFilters(content) {
		return false
	}
	if matchesWindowsPathPattern(lowerContent, content) {
		return true
	}
	if matchesUnixPathPattern(lowerContent) {
		return true
	}
	if hasCommonFileExtension(lowerContent) {
		return true
	}
	return false
}

func analyzePotentialFilePath(text *[]rune, startIndex int) bool {
	if startIndex < 0 || startIndex >= len(*text) {
		return false
	}

	// Find the end of the string
	endIndex := startIndex
	if isQuote((*text)[endIndex]) {
		quote := (*text)[endIndex]
		endIndex++
		for endIndex < len(*text) {
			if (*text)[endIndex] == quote {
				// Check if it's escaped (unless we are in file path mode, but we don't know that yet)
				// For analysis purposes, we assume standard escaping first
				if (*text)[endIndex-1] != codeBackslash {
					endIndex++
					break
				}
			}
			endIndex++
		}
	} else {
		// Unquoted string
		for endIndex < len(*text) && !isDelimiter((*text)[endIndex]) && !isWhitespace((*text)[endIndex]) {
			endIndex++
		}
	}

	content := string((*text)[startIndex:endIndex])
	// Remove surrounding quotes if present
	if len(content) >= 2 && isQuote(rune(content[0])) && isQuote(rune(content[len(content)-1])) {
		content = content[1 : len(content)-1]
	}

	return isLikelyFilePath(content)
}

func isURLStart(text *[]rune, i int) bool {
	if i <= 0 || (*text)[i] != codeColon {
		return false
	}
	j := i - 1
	for j >= 0 && isLetter((*text)[j]) {
		j--
	}
	protocol := string((*text)[j+1 : i])
	if protocol == "http" || protocol == "https" || protocol == "ftp" {
		if i+2 < len(*text) && (*text)[i+1] == codeSlash && (*text)[i+2] == codeSlash {
			return true
		}
	}
	return false
}

func skipMarkdownCodeBlock(text *[]rune, i *int, blocks []string, output *strings.Builder) bool {
	parseWhitespace(text, i, output, true)
	for _, block := range blocks {
		blockRunes := []rune(block)
		end := *i + len(blockRunes)
		if end <= len(*text) {
			match := true
			for j := 0; j < len(blockRunes); j++ {
				if (*text)[*i+j] != blockRunes[j] {
					match = false
					break
				}
			}
			if match {
				*i = end
				return true
			}
		}
	}
	return false
}

func newJSONRepairError(message string, position int, err ...error) *Error {
	var inner error
	if len(err) > 0 {
		inner = err[0]
	}
	return &Error{Message: message, Position: position, Err: inner}
}

func newUnexpectedEndError(position int) *Error {
	return newJSONRepairError("Unexpected end of json string", position, ErrUnexpectedEnd)
}

func newObjectKeyExpectedError(position int) *Error {
	return newJSONRepairError("Object key expected", position, ErrObjectKeyExpected)
}

func newColonExpectedError(position int) *Error {
	return newJSONRepairError("Colon expected", position, ErrColonExpected)
}

func newUnexpectedCharacterError(message string, position int) *Error {
	return newJSONRepairError(message, position, ErrUnexpectedCharacter)
}

func newInvalidUnicodeError(message string, position int) *Error {
	return newJSONRepairError(message, position, ErrInvalidUnicode)
}

func newInvalidCharacterError(message string, position int) *Error {
	return newJSONRepairError(message, position, ErrInvalidCharacter)
}
