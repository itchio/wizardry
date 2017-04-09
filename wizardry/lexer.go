package wizardry

import (
	"fmt"
	"strconv"
)

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t'
}

func isNumber(b byte) bool {
	return '0' <= b && b <= '9'
}

func isOctalNumber(b byte) bool {
	return '0' <= b && b <= '7'
}

func isHexNumber(b byte) bool {
	return ('0' <= b && b <= '9') || ('a' <= b && b <= 'f') || ('A' <= b && b <= 'F')
}

func isLowerLetter(b byte) bool {
	return 'a' <= b && b <= 'z'
}

func isUpperLetter(b byte) bool {
	return 'A' <= b && b <= 'Z'
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

type parsedUint struct {
	Value    uint64
	NewIndex int
}

func parseInt(input []byte, j int) (*parsedInt, error) {
	inputSize := len(input)

	startJ := j
	if j < inputSize && input[j] == '-' {
		j++
	}

	base := 10

	if (j+1 < inputSize) && input[j] == '0' && input[j+1] == 'x' {
		// hexadecimal
		base = 16
		j += 2
		startJ = j
		for j < inputSize && isHexNumber(input[j]) {
			j++
		}
	} else if j+1 < inputSize && input[j] == '0' && isOctalNumber(input[j+1]) {
		// octal
		base = 8
		j++
		startJ = j
		for j < inputSize && isOctalNumber(input[j]) {
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

func parseUint(input []byte, j int) (*parsedUint, error) {
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
	} else if j+1 < inputSize && input[j] == '0' && isOctalNumber(input[j+1]) {
		// octal
		base = 8
		j++
		startJ = j
		for j < inputSize && isOctalNumber(input[j]) {
			j++
		}
	} else {
		// decimal
		for j < inputSize && isNumber(input[j]) {
			j++
		}
	}

	value, err := strconv.ParseUint(string(input[startJ:j]), base, 64)
	if err != nil {
		return nil, err
	}

	return &parsedUint{
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
			case 'r':
				result = append(result, '\r')
				j++
			case 'n':
				result = append(result, '\n')
				j++
			case 't':
				result = append(result, '\t')
				j++
			case 'v':
				result = append(result, '\v')
				j++
			case 'b':
				result = append(result, '\b')
				j++
			case 'a':
				result = append(result, '\a')
				j++
			case ' ':
				result = append(result, ' ')
				j++
			case 'x':
				j++
				// hexadecimal escape, e.g. "\x" or "\xeb"
				hexLen := 0
				if j < inputSize && isHexNumber(input[j]) {
					hexLen++
					if j+1 < inputSize && isHexNumber(input[j+1]) {
						hexLen++
					}
				}

				if hexLen == 0 {
					return nil, fmt.Errorf("invalid/unfinished hex escape in %s", input)
				}

				hexInput := string(input[j : j+hexLen])

				val, err := strconv.ParseUint(hexInput, 16, 8)
				if err != nil {
					return nil, fmt.Errorf("in hex escape %s: %s", hexInput, err.Error())
				}
				result = append(result, byte(val))
				j += hexLen
			default:
				if isOctalNumber(input[j]) {
					numOctal := 1
					k := j + 1
					for k < inputSize && numOctal < 3 && isOctalNumber(input[k]) {
						numOctal++
						k++
					}

					// octal escape e.g. "\0", "\11", "\222", but no longer
					octInput := string(input[j:k])
					val, err := strconv.ParseUint(octInput, 8, 8)
					if err != nil {
						return nil, fmt.Errorf("in oct escape %s: %s", octInput, err.Error())
					}
					result = append(result, byte(val))
					j = k
				} else {
					return nil, fmt.Errorf("unrecognized escape sequence starting with 0x%x, aka '\\%c'", input[j], input[j])
				}
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
