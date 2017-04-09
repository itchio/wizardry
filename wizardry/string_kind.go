package wizardry

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

func stringTest(target []byte, targetIndex int, magic []byte, flags stringTestFlags) int {
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
			return -1
		}

		if flags.CompactWhitespace && isWhitespace(targetByte) {
			// if we had whitespace, skip any whitespace coming after it
			for targetIndex < targetSize && isWhitespace(target[targetIndex]) {
				targetIndex++
			}
		}

		if magicIndex >= magicSize {
			// hey it matched all the way!
			return targetIndex
		}
	}

	// reached the end of target without matching magic, hence not a match
	return -1
}
