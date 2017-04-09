package wizardry

func searchTest(target []byte, targetIndex int, maxLen int, pattern string) int {
	sf := makeStringFinder(pattern)
	targetMaxIndex := targetIndex + maxLen
	if targetMaxIndex > len(target) {
		targetMaxIndex = len(target)
	}
	text := string(target[targetIndex:targetMaxIndex])
	return sf.next(text)
}
