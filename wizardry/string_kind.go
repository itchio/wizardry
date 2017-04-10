package wizardry

import "github.com/fasterthanlime/wizardry/wizardry/wizutil"

// StringTestFlags describes how to perform a string test
type StringTestFlags struct {
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

// StringTest looks for a string pattern in target, at given index
func StringTest(target []byte, targetIndex int, pattern []byte, flags StringTestFlags) int {
	targetSize := len(target)
	patternSize := len(pattern)
	patternIndex := 0

	for targetIndex < targetSize {
		patternByte := pattern[patternIndex]
		targetByte := target[targetIndex]

		matches := patternByte == targetByte
		if matches {
			// perfect match, advance both
			targetIndex++
			patternIndex++
		} else if flags.OptionalBlanks && wizutil.IsWhitespace(patternByte) {
			// cool, it's optional then
			patternIndex++
		} else if flags.LowerMatchesBoth && wizutil.IsLowerLetter(patternByte) && wizutil.ToLower(targetByte) == patternByte {
			// case insensitive match
			targetIndex++
			patternIndex++
		} else if flags.UpperMatchesBoth && wizutil.IsUpperLetter(patternByte) && wizutil.ToUpper(targetByte) == patternByte {
			// case insensitive match
			targetIndex++
			patternIndex++
		} else {
			// not a match
			return -1
		}

		if flags.CompactWhitespace && wizutil.IsWhitespace(targetByte) {
			// if we had whitespace, skip any whitespace coming after it
			for targetIndex < targetSize && wizutil.IsWhitespace(target[targetIndex]) {
				targetIndex++
			}
		}

		if patternIndex >= patternSize {
			// hey it matched all the way!
			return targetIndex
		}
	}

	// reached the end of target without matching pattern, hence not a match
	return -1
}
