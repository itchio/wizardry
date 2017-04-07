package wizardry

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t'
}

func Identify(rules io.Reader, targetContents []byte) ([]string, error) {
	var outStrings []string
	scanner := bufio.NewScanner(rules)

	currentLevel := 0
	globalOffset := int64(0)

	for scanner.Scan() {
		line := scanner.Text()
		bytes := []byte(line)
		numBytes := len(bytes)

		if numBytes == 0 {
			// empty line, ignore
			continue
		}

		i := 0

		if bytes[i] == '#' {
			// comment, ignore
			continue
		}

		if bytes[i] == '!' {
			// fmt.Printf("ignoring instruction %s\n", line)
			continue
		}

		fmt.Printf("\nline %s\n", line)

		// read level
		level := 0
		for i < numBytes && bytes[i] == '>' && i < len(bytes) {
			level++
			i++
		}

		// read offset
		offsetStart := i
		for i < numBytes && !isWhitespace(bytes[i]) {
			i++
		}
		offsetEnd := i
		offset := line[offsetStart:offsetEnd]

		// skip whitespace
		for i < numBytes && isWhitespace(bytes[i]) {
			i++
		}

		// read kind
		kindStart := i
		for i < numBytes && !isWhitespace(bytes[i]) {
			i++
		}
		kindEnd := i
		kind := line[kindStart:kindEnd]

		// skip whitespace
		for i < numBytes && isWhitespace(bytes[i]) {
			i++
		}

		// read test
		testStart := i
		for i < numBytes && !isWhitespace(bytes[i]) {
			if bytes[i] == '\\' {
				i += 2
			} else {
				i++
			}
		}
		testEnd := i
		test := line[testStart:testEnd]

		// skip whitespace
		for i < numBytes && isWhitespace(bytes[i]) {
			i++
		}

		fmt.Printf("level (%d/%d), offset (%s), kind (%s), test (%s), extra (%s)\n", level, currentLevel, offset, kind, test, line[i:])

		localOffsetBase := int64(0)
		offsetBytes := []byte(offset)

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

			indirectAddrStart := j
			base := 10

			if offsetBytes[j] == '0' && offsetBytes[j+1] == 'x' {
				// hexadecimal
				fmt.Printf("indirect addr start is hexadecimal\n")
				base = 16
				j += 2
				indirectAddrStart = j
				for (offsetBytes[j] >= '0' && offsetBytes[j] <= '9') || (offsetBytes[j] >= 'a' && offsetBytes[j] <= 'f') {
					j++
				}
			} else {
				// decimal
				for offsetBytes[j] >= '0' && offsetBytes[j] <= '9' {
					j++
				}
			}
			indirectAddrEnd := j
			indirectAddrBytes := offsetBytes[indirectAddrStart:indirectAddrEnd]
			fmt.Printf("indirect addr = %s\n", string(indirectAddrBytes))

			indirectAddrVal, err := strconv.ParseInt(string(indirectAddrBytes), base, 64)
			if err != nil {
				return nil, err
			}
			fmt.Printf("indirect addr value = %d\n", indirectAddrVal)

			indirectAddrVal += indirectAddrOffset
			fmt.Printf("indirect addr value after offset = %d\n", indirectAddrVal)

			if offsetBytes[j] != '.' {
				fmt.Printf("malformed indirect offset in %s, expected '.'\n", string(offsetBytes))
				continue
			}
			j++

			if offsetBytes[j] != ')' {
				fmt.Printf("malformed indirect offset in %s, expected ')'\n", string(offsetBytes))
				continue
			}
			j++
		}
	}

	return outStrings, nil
}
