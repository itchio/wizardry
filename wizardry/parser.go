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
	globalOffset := uint64(0)

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

		stopProcessing := false

		for l := level + 1; l < len(matchedLevels); l++ {
			// if any deeper level was already matched, we can stop processing here
			if matchedLevels[l] {
				stopProcessing = true
				break
			}
		}

		if stopProcessing {
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

		localOffsetBase := uint64(0)
		offsetBytes := []byte(offset)
		finalOffset := uint64(0)

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

				indirectAddrOffset := uint64(0)
				if offsetBytes[j] == '&' {
					indirectAddrOffset = uint64(localOffsetBase)
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

				var dereferencedValue uint64
				addrBytes := targetContents[indirectAddr.Value : indirectAddr.Value+uint64(indirectAddrFormatWidth)]

				switch indirectAddrFormatWidth {
				case 1:
					dereferencedValue = uint64(addrBytes[0])
				case 2:
					dereferencedValue = uint64(byteOrder.Uint16(addrBytes))
				case 4:
					dereferencedValue = uint64(byteOrder.Uint32(addrBytes))
				}

				fmt.Printf("Dereferenced value: %d\n", dereferencedValue)

				indirectOffsetOperator := '@'
				indirectOffsetRhs := uint64(0)

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

		if lookupOffset < 0 || lookupOffset >= uint64(len(targetContents)) {
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
				var targetValue uint64

				switch simpleKind {
				case "byte":
					targetValue = uint64(targetContents[lookupOffset])
				case "short":
					targetValue = uint64(byteOrder.Uint16(targetContents[lookupOffset : lookupOffset+2]))
				case "long":
					targetValue = uint64(byteOrder.Uint32(targetContents[lookupOffset : lookupOffset+4]))
				case "quad":
					targetValue = uint64(byteOrder.Uint64(targetContents[lookupOffset : lookupOffset+8]))
				}

				fmt.Printf("target value = %d aka 0x%x, test = %s\n", targetValue, targetValue, test)

				// TODO: AND-ing in kind
				doAnd := false
				andValue := uint64(0)
				if j < len(kind) && kind[j] == '&' {
					j++
					doAnd = true
					parsedAndValue, err := parseInt(kind, j)
					if err != nil {
						fmt.Printf("in integer test, couldn't parse and value %s, skipping\n", kind[j:])
						break
					}
					andValue = parsedAndValue.Value
					j = parsedAndValue.NewIndex
				}

				operator := '='
				negate := false
				k := 0
				switch test[k] {
				case '!':
					negate = true
					k++
				case '<':
					operator = '<'
					k++
				case '>':
					operator = '>'
					k++
				}

				parsedMagicValue, err := parseInt(test, k)
				if err != nil {
					fmt.Printf("for integer test, couldn't parse magic value %s, ignoring", string(test[k:]))
					continue
				}

				magicValue := parsedMagicValue.Value
				k = parsedMagicValue.NewIndex

				if doAnd {
					targetValue &= andValue
				}

				switch operator {
				case '=':
					success = targetValue == magicValue
				case '<':
					success = targetValue < magicValue
				case '>':
					success = targetValue > magicValue
				default:
					fmt.Printf("for integer test, unsupported operator (%c), skipping\n", operator)
					continue
				}

				if negate {
					success = !success
				}

			case "string":
				k := 0
				negate := false
				if test[k] == '!' {
					negate = true
					k++
				}

				parsedRHS, err := parseString(test, k)
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

				if negate {
					success = !success
				}
			default:
				fmt.Printf("unhandled kind (%s)\n", parsedKind.Value)
				continue
			}

			if success {
				extraString := string(extra)
				extraString = strings.Replace(extraString, "\\b", "", -1)

				fmt.Printf("> test succeeded! matching level %d, appending %s, new offset %d\n",
					level, extraString, lookupOffset)
				outStrings = append(outStrings, extraString)
				matchedLevels[level] = true
				globalOffset = lookupOffset
			} else {
				matchedLevels[level] = false
			}
		}
	}

	return outStrings, nil
}
