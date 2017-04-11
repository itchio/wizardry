package main

func identifyv2() {
	if false /* rule 0 failed */ {
		goto r0f
	}

	// rule 0 passed, adjust offsets, append description etc.

	if false /* rule 1 (sub of rule 0) failed */ {
		goto r1f
	}

	// rule 1 success here
	goto r0s
r1f:

	if false /* rule 1 (sub of rule 0) failed */ {
		goto r2f
	}

	// rule 2 success here
	goto r0s
r2f:

	goto r0s
r0s: // end of sub-rules for r0
	goto end

r0f: // r1 failed

	if false /* rule 3 failed */ {
		goto r3f
	}

	// rule 3 passed, adjust offsets, append description etc.

	// sub-rules if any

	goto r3s
r3s: // end of sub-rules for r3
	goto end

r3f:

end:
	// yay we're done, return stuff
}
