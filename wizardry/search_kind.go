package wizardry

import "fmt"

// SearchTest looks for a fixed pattern at any position within a certain length
func SearchTest(target []byte, targetIndex int, maxLen int, pattern string) int {
	sf := MakeStringFinder(pattern)
	if targetIndex >= len(target) {
		fmt.Printf("SearchTest out of bounds: %d > %d (for pattern %s)", targetIndex, len(target), pattern)
		return -1
	}

	targetMaxIndex := targetIndex + maxLen
	if targetMaxIndex > len(target) {
		targetMaxIndex = len(target)
	}
	text := string(target[targetIndex:targetMaxIndex])
	return sf.next(text)
}
