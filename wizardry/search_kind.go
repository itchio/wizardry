package wizardry

// SearchTest looks for a fixed pattern at any position within a certain length
func SearchTest(target []byte, targetIndex int, maxLen int, pattern string) int {
	sf := MakeStringFinder(pattern)
	targetMaxIndex := targetIndex + maxLen
	if targetMaxIndex > len(target) {
		targetMaxIndex = len(target)
	}
	text := string(target[targetIndex:targetMaxIndex])
	index := sf.next(text)
	if index == -1 {
		return -1
	}
	return index + targetIndex
}
