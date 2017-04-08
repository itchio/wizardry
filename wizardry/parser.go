package wizardry

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
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

type ParsedInt struct {
	Value    int64
	NewIndex int
}

func parseInt(input []byte, j int) (*ParsedInt, error) {
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

	return &ParsedInt{
		Value:    value,
		NewIndex: j,
	}, nil
}

type ParsedKind struct {
	Value    string
	NewIndex int
}

func parseKind(input []byte, j int) *ParsedKind {
	inputSize := len(input)
	startJ := j

	for j < inputSize && (isNumber(input[j]) || isLowerLetter(input[j])) {
		j++
	}

	return &ParsedKind{
		Value:    string(input[startJ:j]),
		NewIndex: j,
	}
}

func Identify(rules io.Reader, targetContents []byte) ([]string, error) {
	var outStrings []string
	scanner := bufio.NewScanner(rules)

	currentLevel := 0
	globalOffset := int64(0)

	for scanner.Scan() {
		line := scanner.Text()
		lineBytes := []byte(line)
		numBytes := len(lineBytes)

		if numBytes == 0 {
			// empty line, ignore
			continue
		}

		i := 0

		if lineBytes[i] == '#' {
			// comment, ignore
			continue
		}

		if lineBytes[i] == '!' {
			// fmt.Printf("ignoring instruction %s\n", line)
			continue
		}

		fmt.Printf("\nline %s\n", line)

		// read level
		level := 0
		for i < numBytes && lineBytes[i] == '>' {
			level++
			i++
		}

		// read offset
		offsetStart := i
		for i < numBytes && !isWhitespace(lineBytes[i]) {
			i++
		}
		offsetEnd := i
		offset := line[offsetStart:offsetEnd]

		// skip whitespace
		for i < numBytes && isWhitespace(lineBytes[i]) {
			i++
		}

		// read kind
		kindStart := i
		for i < numBytes && !isWhitespace(lineBytes[i]) {
			i++
		}
		kindEnd := i
		kind := lineBytes[kindStart:kindEnd]

		// skip whitespace
		for i < numBytes && isWhitespace(lineBytes[i]) {
			i++
		}

		// read test
		testStart := i
		for i < numBytes && !isWhitespace(lineBytes[i]) {
			if lineBytes[i] == '\\' {
				i += 2
			} else {
				i++
			}
		}
		testEnd := i
		test := line[testStart:testEnd]

		// skip whitespace
		for i < numBytes && isWhitespace(lineBytes[i]) {
			i++
		}

		fmt.Printf("level (%d/%d), offset (%s), kind (%s), test (%s), extra (%s)\n", level, currentLevel, offset, kind, test, line[i:])

		localOffsetBase := int64(0)
		offsetBytes := []byte(offset)
		finalOffset := int64(0)

		{
			j := 0
			if offsetBytes[j] == '&' {
				// offset is relative to globalOffset
				localOffsetBase = globalOffset
				j++
			}

			fmt.Printf("local offset base = %d\n", localOffsetBase)

			if offsetBytes[j] == '(' {
				fmt.Printf("found indirect offset\n")
				j++

				indirectAddrOffset := int64(0)
				if offsetBytes[j] == '&' {
					indirectAddrOffset = localOffsetBase
					fmt.Printf("indirect offset is relative\n")
					j++
				}

				indirectAddr, err := parseInt(offsetBytes, j)
				if err != nil {
					fmt.Printf("error: couldn't parse rule %s\n", line)
					continue
				}

				j = indirectAddr.NewIndex

				fmt.Printf("indirect addr = %d\n", indirectAddr.Value)

				indirectAddr.Value += indirectAddrOffset
				fmt.Printf("indirect addr after offset = %d\n", indirectAddr.Value)

				if offsetBytes[j] != '.' {
					fmt.Printf("malformed indirect offset in %s, expected '.', got '%c'\n", string(offsetBytes), offsetBytes[j])
					continue
				}
				j++

				indirectAddrFormat := offsetBytes[j]
				fmt.Printf("format: %c\n", indirectAddrFormat)
				j++

				indirectAddrFormatWidth := 0
				var byteOrder binary.ByteOrder = binary.LittleEndian

				switch indirectAddrFormat {
				case 'b':
					indirectAddrFormatWidth = 1
				case 'i':
					fmt.Printf("id3 format not supported, skipping %s\n", line)
					continue
				case 's':
					indirectAddrFormatWidth = 2
				case 'l':
					indirectAddrFormatWidth = 4
				case 'B':
					indirectAddrFormatWidth = 1
					byteOrder = binary.BigEndian
				case 'I':
					fmt.Printf("id3 format not supported, skipping %s\n", line)
					continue
				case 'S':
					indirectAddrFormatWidth = 2
					byteOrder = binary.BigEndian
				case 'L':
					indirectAddrFormatWidth = 4
					byteOrder = binary.BigEndian
				case 'm':
					fmt.Printf("middle-endian format not supported, skipping %s\n", line)
					continue
				default:
					fmt.Printf("unsupported indirect addr format %c, skipping %s\n", indirectAddrFormat, line)
					continue
				}

				var dereferencedValue int64
				addrBytes := targetContents[indirectAddr.Value : indirectAddr.Value+int64(indirectAddrFormatWidth)]

				switch indirectAddrFormatWidth {
				case 1:
					dereferencedValue = int64(addrBytes[0])
				case 2:
					dereferencedValue = int64(byteOrder.Uint16(addrBytes))
				case 4:
					dereferencedValue = int64(byteOrder.Uint32(addrBytes))
				}

				fmt.Printf("Dereferenced value: %d\n", dereferencedValue)

				indirectOffsetOperator := '@'
				indirectOffsetRhs := int64(0)

				if offsetBytes[j] == '+' {
					indirectOffsetOperator = '+'
				} else if offsetBytes[j] == '-' {
					indirectOffsetOperator = '-'
				} else if offsetBytes[j] == '*' {
					indirectOffsetOperator = '*'
				} else if offsetBytes[j] == '/' {
					indirectOffsetOperator = '/'
				}

				if indirectOffsetOperator != '@' {
					j++
					parsedRhs, err := parseInt(offsetBytes, j)
					if err != nil {
						fmt.Printf("malformed indirect offset rhs, skipping %s\n", line)
						continue
					}

					indirectOffsetRhs = parsedRhs.Value
					j = parsedRhs.NewIndex
				}

				fmt.Printf("indirectOffset operator = %c, rhs = %d\n", indirectOffsetOperator, indirectOffsetRhs)

				finalOffset = dereferencedValue
				switch indirectOffsetOperator {
				case '+':
					finalOffset += indirectOffsetRhs
				case '-':
					finalOffset -= indirectOffsetRhs
				case '*':
					finalOffset *= indirectOffsetRhs
				case '/':
					finalOffset /= indirectOffsetRhs
				}

				fmt.Printf("final offset = %d\n", finalOffset)

				if offsetBytes[j] != ')' {
					fmt.Printf("malformed indirect offset in %s, expected ')', got '%c'\n", string(offsetBytes), offsetBytes[j])
					continue
				}
				j++
			} else {
				parsedAbsolute, err := parseInt(offsetBytes, j)
				if err != nil {
					fmt.Printf("malformed absolute offset, expected number, got %s\n", offsetBytes[j:])
					continue
				}

				finalOffset = parsedAbsolute.Value
				j = parsedAbsolute.NewIndex
			}
		}

		lookupOffset := finalOffset + localOffsetBase
		fmt.Printf("lookup offset = %d\n", lookupOffset)

		{
			j := 0
			parsedKind := parseKind(kind, j)
			fmt.Printf("parsed kind = %s\n", parsedKind.Value)
		}
	}

	return outStrings, nil
}
