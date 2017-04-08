package wizardry

import (
	"errors"
	"fmt"
	"strconv"
)

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t'
}

func isNumber(b byte) bool {
	return '0' <= b && b <= '9'
}

func isHexNumber(b byte) bool {
	return ('0' <= b && b <= '9') || ('a' <= b && b <= 'f')
}

func isLowerLetter(b byte) bool {
	return 'a' <= b && b <= 'z'
}

func isUpperLetter(b byte) bool {
	return 'a' <= b && b <= 'z'
}

func toLower(b byte) byte {
	if isUpperLetter(b) {
		return b + ('a' - 'A')
	}
	return b
}

func toUpper(b byte) byte {
	if isLowerLetter(b) {
		return b - ('a' - 'A')
	}
	return b
}

type parsedInt struct {
	Value    int64
	NewIndex int
}

func parseInt(input []byte, j int) (*parsedInt, error) {
	inputSize := len(input)
	startJ := j
	base := 10

	if (j+1 < inputSize) && input[j] == '0' && input[j+1] == 'x' {
		// hexadecimal
		base = 16
		j += 2
		startJ = j
		for j < inputSize && isHexNumber(input[j]) {
			j++
		}
	} else {
		// decimal
		for j < inputSize && isNumber(input[j]) {
			j++
		}
	}

	value, err := strconv.ParseInt(string(input[startJ:j]), base, 64)
	if err != nil {
		return nil, err
	}

	return &parsedInt{
		Value:    value,
		NewIndex: j,
	}, nil
}

type parsedKind struct {
	Value    string
	NewIndex int
}

func parseKind(input []byte, j int) *parsedKind {
	inputSize := len(input)
	startJ := j

	for j < inputSize && (isNumber(input[j]) || isLowerLetter(input[j])) {
		j++
	}

	return &parsedKind{
		Value:    string(input[startJ:j]),
		NewIndex: j,
	}
}

type parsedString struct {
	Value    []byte
	NewIndex int
}

func parseString(input []byte, j int) (*parsedString, error) {
	inputSize := len(input)

	var result []byte
	for j < inputSize {
		if input[j] == '\\' {
			j++
			switch input[j] {
			case '\\':
				result = append(result, '\\')
				j++
			case ' ':
				result = append(result, ' ')
				j++
			case '0':
				result = append(result, 0)
				j++
			default: // ?
				return nil, errors.New(fmt.Sprintf("unrecognized escape character %x", input[j]))
			}
		} else {
			result = append(result, input[j])
			j++
		}
	}

	return &parsedString{
		Value:    result,
		NewIndex: j,
	}, nil
}

type stringTestFlags struct {
	// the "W" flag compacts whitespace in the target,
	// which must contain at least one whitespace character
	CompactWhitespace bool
	// the "w" flag treats every blank in the magic as an optional blank
	OptionalBlanks bool
	// the "c" flag specifies case-insensitive matching: lower case
	// characters in the magic match both lower and upper case characters
	// in the target
	LowerMatchesBoth bool
	// the "C" flag specifies case-insensitive matching: upper case
	// characters in the magic match both lower and upper case characters
	// in the target
	UpperMatchesBoth bool
	// the "t" flag forces the test to be done for text files
	ForceText bool
	// the "b" flag forces the test to be done for binary files
	ForceBinary bool
}

type parsedStringTestFlags struct {
	Flags    stringTestFlags
	NewIndex int
}

func parseStringTestFlags(input []byte, j int) *parsedStringTestFlags {
	inputSize := len(input)

	result := &parsedStringTestFlags{}

	for j < inputSize {
		switch input[j] {
		case 'W':
			result.Flags.CompactWhitespace = true
		case 'w':
			result.Flags.OptionalBlanks = true
		case 'c':
			result.Flags.LowerMatchesBoth = true
		case 'C':
			result.Flags.UpperMatchesBoth = true
		case 't':
			result.Flags.ForceText = true
		case 'b':
			result.Flags.ForceBinary = true
		default:
			break
		}
		j++
	}

	return result
}

func stringTest(target []byte, targetIndex int, magic []byte, flags stringTestFlags) bool {
	targetSize := len(target)
	magicSize := len(magic)
	magicIndex := 0

	for targetIndex < targetSize {
		magicByte := magic[magicIndex]
		targetByte := target[targetIndex]

		matches := magicByte == targetByte
		if matches {
			// perfect match, advance both
			targetIndex++
			magicIndex++
		} else if flags.OptionalBlanks && isWhitespace(magicByte) {
			// cool, it's optional then
			magicIndex++
		} else if flags.LowerMatchesBoth && isLowerLetter(magicByte) && toLower(targetByte) == magicByte {
			// case insensitive match
			targetIndex++
			magicIndex++
		} else if flags.UpperMatchesBoth && isUpperLetter(magicByte) && toUpper(targetByte) == magicByte {
			// case insensitive match
			targetIndex++
			magicIndex++
		} else {
			// not a match
			return false
		}

		if flags.CompactWhitespace && isWhitespace(targetByte) {
			// if we had whitespace, skip any whitespace coming after it
			for targetIndex < targetSize && isWhitespace(target[targetIndex]) {
				targetIndex++
			}
		}

		if magicIndex >= magicSize {
			// hey it matched all the way!
			return true
		}
	}

	// reached the end of target without matching magic, hence not a match
	return false
}
