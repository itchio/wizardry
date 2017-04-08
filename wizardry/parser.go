package wizardry

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

func Identify(rules io.Reader, targetContents []byte) ([]string, error) {
	var outStrings []string
	scanner := bufio.NewScanner(rules)

	matchedLevels := make([]bool, 32)
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

		// read level
		level := 0
		for i < numBytes && lineBytes[i] == '>' {
			level++
			i++
		}

		if matchedLevels[level] {
			// if we've already matched at this level, we can stop processing
			break
		}

		skipRule := false
		for l := 0; l < level; l++ {
			if !matchedLevels[l] {
				// if any of the parent levels aren't matched, skip the rule entirely
				skipRule = true
				break
			}
		}

		if skipRule {
			continue
		}

		fmt.Printf("\n| %s\n", line)

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
			// this isn't the greatest trick in the world tbh
			if lineBytes[i] == '\\' {
				i += 2
			} else {
				i++
			}
		}
		testEnd := i
		test := lineBytes[testStart:testEnd]

		// skip whitespace
		for i < numBytes && isWhitespace(lineBytes[i]) {
			i++
		}

		extra := lineBytes[i:]
		// fmt.Printf("level (%d/%d), offset (%s), kind (%s), test (%s), extra (%s)\n", level, currentLevel, offset, kind, test, line[i:])

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

			// fmt.Printf("local offset base = %d\n", localOffsetBase)

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

		if lookupOffset < 0 || lookupOffset >= int64(len(targetContents)) {
			fmt.Printf("we done goofed, lookupOffset %d is out of bounds, skipping %s\n", lookupOffset, line)
			continue
		}

		{
			j := 0
			parsedKind := parseKind(kind, j)
			j += parsedKind.NewIndex

			success := false

			switch parsedKind.Value {
			case "byte", "short", "long", "quad",
				"beshort", "belong", "bequad",
				"leshort", "lelong", "lequad":

				var byteOrder binary.ByteOrder = binary.LittleEndian
				simpleKind := parsedKind.Value
				if strings.HasPrefix(simpleKind, "le") {
					simpleKind = simpleKind[2:]
				} else if strings.HasPrefix(simpleKind, "be") {
					simpleKind = simpleKind[2:]
					byteOrder = binary.BigEndian
				}
				var rhsValue uint64

				switch simpleKind {
				case "byte":
					rhsValue = uint64(targetContents[lookupOffset])
				case "short":
					rhsValue = uint64(byteOrder.Uint16(targetContents[lookupOffset : lookupOffset+2]))
				case "long":
					rhsValue = uint64(byteOrder.Uint32(targetContents[lookupOffset : lookupOffset+4]))
				case "quad":
					rhsValue = uint64(byteOrder.Uint64(targetContents[lookupOffset : lookupOffset+8]))
				}

				fmt.Printf("rhs value = %d aka 0x%x, test = %s\n", rhsValue, rhsValue, test)

			case "string":
				parsedRHS, err := parseString(test, 0)
				if err != nil {
					fmt.Printf("in string test, couldn't parse rhs: %s - skipping\n", err.Error())
					continue
				}
				rhs := parsedRHS.Value

				var flags stringTestFlags
				if j < len(kind) && kind[j] == '/' {
					j++
					parsedFlags := parseStringTestFlags(kind, j)
					j = parsedFlags.NewIndex
					flags = parsedFlags.Flags
				}

				// fmt.Printf("> performing string test at (%d) with test (%s), flags %+v\n",
				// 	lookupOffset, rhs, flags)

				success = stringTest(targetContents, int(lookupOffset), []byte(rhs), flags)
			default:
				fmt.Printf("unhandled kind (%s)\n", parsedKind.Value)
				continue
			}

			if success {
				fmt.Printf("> test succeeded! matching level %d, appending %s, new offset %d\n",
					level, string(extra), lookupOffset)
				outStrings = append(outStrings, string(extra))
				matchedLevels[level] = true
				globalOffset = lookupOffset
			}
		}
	}

	return outStrings, nil
}
