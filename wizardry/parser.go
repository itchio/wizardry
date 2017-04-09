package wizardry

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// LogFunc prints a debug message
type LogFunc func(format string, args ...interface{})

// ParseContext holds state for the parser
type ParseContext struct {
	Logf LogFunc
}

// Parse reads a magic rule file and puts it into a spell book
func (ctx *ParseContext) Parse(magicReader io.Reader, book Spellbook) error {
	scanner := bufio.NewScanner(magicReader)

	page := ""

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

		rule := Rule{}

		// read level
		for i < numBytes && lineBytes[i] == '>' {
			rule.Level++
			i++
		}

		if rule.Level < 1 {
			// end of the page, if any
			page = ""
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

		descriptionBytes := lineBytes[i:]

		// parse offset
		{
			offsetBytes := []byte(offset)
			j := 0
			if offsetBytes[j] == '&' {
				// offset is relative to globalOffset
				rule.Offset.IsRelative = true
				j++
			}

			if offsetBytes[j] == '(' {
				j++
				rule.Offset.OffsetType = OffsetTypeIndirect

				indirect := &IndirectOffset{}
				rule.Offset.Indirect = indirect

				if offsetBytes[j] == '&' {
					indirect.IsRelative = true
					j++
				}

				indirectAddr, err := parseInt(offsetBytes, j)
				if err != nil {
					ctx.Logf("error: couldn't parse indirect offset in part \"%s\" of rule %s", offsetBytes[j:], line)
					continue
				}

				j = indirectAddr.NewIndex

				indirect.OffsetAddress = indirectAddr.Value

				if offsetBytes[j] != '.' && offsetBytes[j] != ',' {
					ctx.Logf("malformed indirect offset in %s, expected [.,], got '%c'\n", string(offsetBytes), offsetBytes[j])
					continue
				}
				j++

				indirectAddrFormat := offsetBytes[j]
				j++

				indirect.Endianness = LittleEndian

				if isUpperLetter(indirectAddrFormat) {
					indirect.Endianness = BigEndian
					indirectAddrFormat = toLower(indirectAddrFormat)
				}

				switch indirectAddrFormat {
				case 'b':
					indirect.ByteWidth = 1
				case 'i':
					ctx.Logf("id3 format not supported, skipping %s", line)
					continue
				case 's':
					indirect.ByteWidth = 2
				case 'l':
					indirect.ByteWidth = 4
				case 'm':
					ctx.Logf("middle-endian format not supported, skipping %s", line)
					continue
				default:
					ctx.Logf("unsupported indirect addr format %c, skipping %s", indirectAddrFormat, line)
					continue
				}

				if offsetBytes[j] == '+' {
					indirect.OffsetAdjustmentType = OffsetAdjustmentAdd
				} else if offsetBytes[j] == '-' {
					indirect.OffsetAdjustmentType = OffsetAdjustmentSub
				} else if offsetBytes[j] == '*' {
					indirect.OffsetAdjustmentType = OffsetAdjustmentMul
				} else if offsetBytes[j] == '/' {
					indirect.OffsetAdjustmentType = OffsetAdjustmentDiv
				}

				if indirect.OffsetAdjustmentType != OffsetAdjustmentNone {
					j++
					// it's a relative pair
					if offsetBytes[j] == '(' {
						indirect.OffsetAdjustmentIsRelative = true
						j++
					}

					parsedRHS, err := parseInt(offsetBytes, j)
					if err != nil {
						ctx.Logf("malformed indirect offset rhs, skipping %s", line)
						continue
					}

					indirect.OffsetAdjustmentValue = parsedRHS.Value
					j = parsedRHS.NewIndex

					if indirect.OffsetAdjustmentIsRelative {
						if offsetBytes[j] != ')' {
							ctx.Logf("malformed relative offset adjustment, missing closing ')' - in %s", line)
							continue
						}
						j++
					}
				}

				if offsetBytes[j] != ')' {
					ctx.Logf("malformed indirect offset in %s, expected ')', got '%c', skipping", string(offsetBytes), offsetBytes[j])
					continue
				}
				j++
			} else {
				rule.Offset.OffsetType = OffsetTypeDirect

				parsedAbsolute, err := parseInt(offsetBytes, j)
				if err != nil {
					ctx.Logf("malformed absolute offset, expected number, got (%s), skipping", offsetBytes[j:])
					continue
				}

				rule.Offset.Direct = parsedAbsolute.Value
				j = parsedAbsolute.NewIndex
			}
		}

		// parse kind
		{
			j := 0
			parsedKind := parseKind(kind, j)
			j += parsedKind.NewIndex

			switch parsedKind.Value {
			case
				"ubyte", "ushort", "ulong", "uquad",
				"ubeshort", "ubelong", "ubequad",
				"uleshort", "ulelong", "ulequad",
				"byte", "short", "long", "quad",
				"beshort", "belong", "bequad",
				"leshort", "lelong", "lequad":

				ik := &IntegerKind{}
				rule.Kind.Family = KindFamilyInteger
				rule.Kind.Data = ik

				ik.Signed = true
				ik.Endianness = LittleEndian

				simpleKind := parsedKind.Value
				if strings.HasPrefix(simpleKind, "u") {
					simpleKind = simpleKind[1:]
					ik.Signed = false
				}

				if strings.HasPrefix(simpleKind, "le") {
					simpleKind = simpleKind[2:]
				} else if strings.HasPrefix(simpleKind, "be") {
					simpleKind = simpleKind[2:]
					ik.Endianness = BigEndian
				}

				switch simpleKind {
				case "byte":
					ik.ByteWidth = 1
				case "short":
					ik.ByteWidth = 1
				case "long":
					ik.ByteWidth = 1
				case "quad":
					ik.ByteWidth = 1
				default:
					ctx.Logf("unrecognized integer kind %s, skipping rule %s", simpleKind, line)
					continue
				}

				ik.DoAnd = false

				if j < len(kind) && kind[j] == '&' {
					j++
					parsedAndValue, err := parseUint(kind, j)
					if err != nil {
						ctx.Logf("in integer test, couldn't parse and value %s, skipping\n", kind[j:])
						continue
					}
					ik.DoAnd = true
					ik.AndValue = parsedAndValue.Value
					j = parsedAndValue.NewIndex
				}

				ik.IntegerTest = IntegerTestEqual

				k := 0
				switch test[k] {
				case 'x':
					ik.MatchAny = true
					k++
				case '=':
					ik.IntegerTest = IntegerTestEqual
					k++
				case '!':
					ik.IntegerTest = IntegerTestNotEqual
					k++
				case '<':
					ik.IntegerTest = IntegerTestLessThan
					k++
				case '>':
					ik.IntegerTest = IntegerTestGreaterThan
					k++
				}

				if !ik.MatchAny {
					parsedMagicValue, err := parseInt(test, k)
					if err != nil {
						ctx.Logf("for integer test, couldn't parse magic value %s, ignoring", string(test[k:]))
						continue
					}

					ik.Value = parsedMagicValue.Value
					k = parsedMagicValue.NewIndex
				}

			case "string":
				sk := &StringKind{}
				rule.Kind.Family = KindFamilyString
				rule.Kind.Data = sk

				k := 0
				sk.Negate = false
				if test[k] == '!' {
					sk.Negate = true
					k++
				}

				parsedRHS, err := parseString(test, k)
				if err != nil {
					ctx.Logf("in string test, couldn't parse rhs: %s - skipping", err.Error())
					continue
				}
				sk.Value = parsedRHS.Value

				if j < len(kind) && kind[j] == '/' {
					j++
					parsedFlags := parseStringTestFlags(kind, j)
					j = parsedFlags.NewIndex
					sk.Flags = parsedFlags.Flags
				}

			case "search":
				sk := &SearchKind{}
				rule.Kind.Family = KindFamilySearch
				rule.Kind.Data = sk

				sk.MaxLen = 8192
				if j < len(kind) && kind[j] == '/' {
					j++
					parsedLen, err := parseUint(kind, j)
					if err != nil {
						ctx.Logf("in search test, couldn't parse max len in %s: %s - skipping\n", kind[j:], err.Error())
						continue
					}

					j = parsedLen.NewIndex
					sk.MaxLen = int(parsedLen.Value)
				}

				k := 0

				parsedRHS, err := parseString(test, k)
				if err != nil {
					fmt.Printf("in search test, couldn't parse rhs: %s - skipping", err.Error())
					continue
				}
				k = parsedRHS.NewIndex
				sk.Value = parsedRHS.Value

			case "default":
				rule.Kind.Family = KindFamilyDefault
			case "clear":
				rule.Kind.Family = KindFamilyClear
			default:
				fmt.Printf("unhandled kind (%s)\n", parsedKind.Value)
				continue
			}

			rule.Description = descriptionBytes
			book.AddRule(page, rule)
		}
	}

	return nil
}
