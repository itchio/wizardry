package main

func identifyv1() {
lv0_0:
	for {
	rl0:
		for {
			// if it fails
			if false {
				break rl0
			}

			// adjust offset, etc.

			// sub-rules of rl0
		lv1_0:
			for {
			rl1:
				for {
					if false {
						break rl1
					}

					// we don't want to match any other rule at this level
					break lv1_0
				}
			rl2:
				for {
					if false {
						break rl2
					}

					// we don't want to match any other rule at this level
					break lv1_0
				}
			}

			// we don't want to match any other rule at this level
			break lv0_0
		}

	rl3:
		for {
			// if it fails
			if false {
				break rl3
			}

			// adjust offset, etc.

			// sub-rules of rl1

			// we don't want to match any other rule at this level
			break lv0_0
		}
	}
}
