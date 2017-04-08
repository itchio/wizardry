package wizardry

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"regexp"
	"strings"
)

type LogFunc func(format string, args ...interface{})

type ParseContext struct {
	Logf LogFunc
}

func (ctx *ParseContext) Identify(rules io.Reader, targetContents []byte) (string, error) {
	var outStrings []string
	scanner := bufio.NewScanner(rules)

	matchedLevels := make([]bool, 32)
	everMatchedLevels := make([]bool, 32)
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
			continue
		}

		// read level
		level := 0
		for i < numBytes && lineBytes[i] == '>' {
			level++
			i++
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

		// clear all deeper levels
		for l := level; l < len(matchedLevels); l++ {
			matchedLevels[l] = false
		}

		ctx.Logf("| %s", line)

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

			if offsetBytes[j] == '(' {
				j++

				indirectAddrOffset := uint64(0)
				if offsetBytes[j] == '&' {
					indirectAddrOffset = uint64(localOffsetBase)
					j++
				}

				indirectAddr, err := parseUint(offsetBytes, j)
				if err != nil {
					ctx.Logf("error: couldn't parse rule %s", line)
					continue
				}

				j = indirectAddr.NewIndex

				indirectAddr.Value += indirectAddrOffset

				if offsetBytes[j] != '.' {
					ctx.Logf("malformed indirect offset in %s, expected '.', got '%c'\n", string(offsetBytes), offsetBytes[j])
					continue
				}
				j++

				indirectAddrFormat := offsetBytes[j]
				j++

				indirectAddrFormatWidth := 0
				var byteOrder binary.ByteOrder = binary.LittleEndian

				if isUpperLetter(indirectAddrFormat) {
					byteOrder = binary.BigEndian
					indirectAddrFormat = toLower(indirectAddrFormat)
				}

				switch indirectAddrFormat {
				case 'b':
					indirectAddrFormatWidth = 1
				case 'i':
					ctx.Logf("id3 format not supported, skipping %s", line)
					continue
				case 's':
					indirectAddrFormatWidth = 2
				case 'l':
					indirectAddrFormatWidth = 4
				case 'm':
					ctx.Logf("middle-endian format not supported, skipping %s", line)
					continue
				default:
					ctx.Logf("unsupported indirect addr format %c, skipping %s", indirectAddrFormat, line)
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

				indirectOffsetOperator := '@'
				indirectOffsetRHS := uint64(0)

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
					parsedRHS, err := parseUint(offsetBytes, j)
					if err != nil {
						ctx.Logf("malformed indirect offset rhs, skipping %s", line)
						continue
					}

					indirectOffsetRHS = parsedRHS.Value
					j = parsedRHS.NewIndex
				}

				finalOffset = dereferencedValue
				switch indirectOffsetOperator {
				case '+':
					finalOffset += indirectOffsetRHS
				case '-':
					finalOffset -= indirectOffsetRHS
				case '*':
					finalOffset *= indirectOffsetRHS
				case '/':
					finalOffset /= indirectOffsetRHS
				}

				if offsetBytes[j] != ')' {
					ctx.Logf("malformed indirect offset in %s, expected ')', got '%c', skipping", string(offsetBytes), offsetBytes[j])
					continue
				}
				j++
			} else {
				parsedAbsolute, err := parseUint(offsetBytes, j)
				if err != nil {
					ctx.Logf("malformed absolute offset, expected number, got (%s), skipping", offsetBytes[j:])
					continue
				}

				finalOffset = parsedAbsolute.Value
				j = parsedAbsolute.NewIndex
			}
		}

		lookupOffset := finalOffset + localOffsetBase

		if lookupOffset < 0 || lookupOffset >= uint64(len(targetContents)) {
			ctx.Logf("we done goofed, lookupOffset %d is out of bounds, skipping %s", lookupOffset, line)
			continue
		}

		{
			j := 0
			parsedKind := parseKind(kind, j)
			j += parsedKind.NewIndex

			success := false

			switch parsedKind.Value {
			case
				"ubyte", "ushort", "ulong", "uquad",
				"ubeshort", "ubelong", "ubequad",
				"uleshort", "ulelong", "ulequad",
				"byte", "short", "long", "quad",
				"beshort", "belong", "bequad",
				"leshort", "lelong", "lequad":

				signed := true

				var byteOrder binary.ByteOrder = binary.LittleEndian
				simpleKind := parsedKind.Value
				if strings.HasPrefix(simpleKind, "u") {
					simpleKind = simpleKind[1:]
					signed = false
				}

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

				doAnd := false
				andValue := uint64(0)
				if j < len(kind) && kind[j] == '&' {
					j++
					doAnd = true
					parsedAndValue, err := parseUint(kind, j)
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

				parsedMagicValue, err := parseUint(test, k)
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
					if signed {
						switch simpleKind {
						case "byte":
							success = int8(targetValue) < int8(magicValue)
						case "short":
							success = int16(targetValue) < int16(magicValue)
						case "long":
							success = int32(targetValue) < int32(magicValue)
						case "quad":
							success = int64(targetValue) < int64(magicValue)
						}
					} else {
						success = targetValue < magicValue
					}
				case '>':
					if signed {
						switch simpleKind {
						case "byte":
							success = int8(targetValue) > int8(magicValue)
						case "short":
							success = int16(targetValue) > int16(magicValue)
						case "long":
							success = int32(targetValue) > int32(magicValue)
						case "quad":
							success = int64(targetValue) > int64(magicValue)
						}
					} else {
						success = targetValue > magicValue
					}
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
			case "search":
				maxLen := 8192
				if j < len(kind) && kind[j] == '/' {
					j++
					parsedLen, err := parseUint(kind, j)
					if err != nil {
						fmt.Printf("in search test, couldn't parse max len in %s: %s - skipping\n", kind[j:], err.Error())
					}

					j = parsedLen.NewIndex
					maxLen = int(parsedLen.Value)
				}

				k := 0

				parsedRHS, err := parseString(test, k)
				if err != nil {
					fmt.Printf("in string test, couldn't parse rhs: %s - skipping\n", err.Error())
					continue
				}
				k = parsedRHS.NewIndex
				rhs := parsedRHS.Value

				success = stringSearch(targetContents, int(lookupOffset), maxLen, string(rhs))
			case "default":
				// default tests match if nothing has matched before
				if !everMatchedLevels[level] {
					success = true
				}
			case "clear":
				everMatchedLevels[level] = false
			default:
				fmt.Printf("unhandled kind (%s)\n", parsedKind.Value)
				continue
			}

			if success {
				extraString := string(extra)

				fmt.Printf("> test succeeded! matching level %d, appending (%s), new offset %d\n",
					level, extraString, lookupOffset)
				if extraString != "" {
					outStrings = append(outStrings, extraString)
				}
				matchedLevels[level] = true
				everMatchedLevels[level] = true
				globalOffset = lookupOffset
			} else {
				matchedLevels[level] = false
			}
		}
	}

	outString := strings.Join(outStrings, " ")

	re := regexp.MustCompile(`.\\b`)
	outString = re.ReplaceAllString(outString, "")
	outString = strings.TrimSpace(outString)

	return outString, nil
}
