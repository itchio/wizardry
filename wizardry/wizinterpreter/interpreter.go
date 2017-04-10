package wizinterpreter

import (
	"fmt"

	"github.com/fasterthanlime/wizardry/wizardry"
	"github.com/fasterthanlime/wizardry/wizardry/wizparser"
)

// MaxLevels is the maximum level of magic rules that are interpreted
const MaxLevels = 32

// LogFunc logs something somewhere
type LogFunc func(format string, args ...interface{})

// InterpretContext holds state for the interpreter
type InterpretContext struct {
	Logf LogFunc
	Book wizparser.Spellbook
}

// Identify follows the rules in a spellbook to find out the type of a file
func (ctx *InterpretContext) Identify(target []byte) ([]string, error) {
	outStrings, err := ctx.identifyInternal(target, 0, "", false)
	if err != nil {
		return nil, err
	}

	return outStrings, nil
}

func (ctx *InterpretContext) identifyInternal(target []byte, pageOffset int64, page string, swapEndian bool) ([]string, error) {
	var outStrings []string

	matchedLevels := make([]bool, MaxLevels)
	everMatchedLevels := make([]bool, MaxLevels)
	globalOffset := int64(0)

	ctx.Logf("|====> identifying at %d using page %s (%d rules)", pageOffset, page, len(ctx.Book[page]))

	if page != "" {
		matchedLevels[0] = true
		everMatchedLevels[0] = true
	}

	for _, rule := range ctx.Book[page] {
		stopProcessing := false

		// if any of the deeper levels have ever matched, stop working
		for l := rule.Level + 1; l < len(matchedLevels); l++ {
			if everMatchedLevels[l] {
				stopProcessing = true
				break
			}
		}

		if stopProcessing {
			break
		}

		skipRule := false
		for l := 0; l < rule.Level; l++ {
			if !matchedLevels[l] {
				// if any of the parent levels aren't matched, skip the rule entirely
				skipRule = true
				break
			}
		}

		if skipRule {
			continue
		}

		lookupOffset := int64(0)

		ctx.Logf("| %s", rule)

		switch rule.Offset.OffsetType {
		case wizparser.OffsetTypeIndirect:
			indirect := rule.Offset.Indirect
			offsetAddress := indirect.OffsetAddress

			if indirect.IsRelative {
				offsetAddress += int64(globalOffset)
			}

			readAddress, err := readAnyUint(target, int(offsetAddress), indirect.ByteWidth, indirect.Endianness.MaybeSwapped(swapEndian))
			if err != nil {
				ctx.Logf("Error while dereferencing: %s - skipping rule", err.Error())
				continue
			}
			lookupOffset = int64(readAddress)

			offsetAdjustValue := indirect.OffsetAdjustmentValue
			if indirect.OffsetAdjustmentIsRelative {
				offsetAdjustAddress := int64(offsetAddress) + offsetAdjustValue
				readAdjustAddress, err := readAnyUint(target, int(offsetAdjustAddress), indirect.ByteWidth, indirect.Endianness)
				if err != nil {
					ctx.Logf("Error while dereferencing: %s - skipping rule", err.Error())
					continue
				}
				offsetAdjustValue = int64(readAdjustAddress)
			}

			switch indirect.OffsetAdjustmentType {
			case wizparser.OffsetAdjustmentAdd:
				lookupOffset = lookupOffset + offsetAdjustValue
			case wizparser.OffsetAdjustmentSub:
				lookupOffset = lookupOffset - offsetAdjustValue
			case wizparser.OffsetAdjustmentMul:
				lookupOffset = lookupOffset * offsetAdjustValue
			case wizparser.OffsetAdjustmentDiv:
				lookupOffset = lookupOffset / offsetAdjustValue
			}

		case wizparser.OffsetTypeDirect:
			lookupOffset = rule.Offset.Direct + pageOffset
		}

		if rule.Offset.IsRelative {
			lookupOffset += globalOffset
		}

		if lookupOffset < 0 || lookupOffset >= int64(len(target)) {
			ctx.Logf("we done goofed, lookupOffset %d is out of bounds, skipping %#v", lookupOffset, rule)
			continue
		}

		success := false

		switch rule.Kind.Family {
		case wizparser.KindFamilyInteger:
			ki, _ := rule.Kind.Data.(*wizparser.IntegerKind)

			if ki.MatchAny {
				success = true
			} else {
				targetValue, err := readAnyUint(target, int(lookupOffset), ki.ByteWidth, ki.Endianness)
				if err != nil {
					ctx.Logf("in integer test, while reading target value: %s", err.Error())
					continue
				}

				if ki.DoAnd {
					targetValue &= ki.AndValue
				}

				switch ki.IntegerTest {
				case wizparser.IntegerTestEqual:
					success = targetValue == uint64(ki.Value)
				case wizparser.IntegerTestNotEqual:
					success = targetValue != uint64(ki.Value)
				case wizparser.IntegerTestLessThan:
					if ki.Signed {
						switch ki.ByteWidth {
						case 1:
							success = int8(targetValue) < int8(ki.Value)
						case 2:
							success = int16(targetValue) < int16(ki.Value)
						case 4:
							success = int32(targetValue) < int32(ki.Value)
						case 8:
							success = int64(targetValue) < int64(ki.Value)
						}
					} else {
						success = targetValue < uint64(ki.Value)
					}
				case wizparser.IntegerTestGreaterThan:
					if ki.Signed {
						switch ki.ByteWidth {
						case 1:
							success = int8(targetValue) > int8(ki.Value)
						case 2:
							success = int16(targetValue) > int16(ki.Value)
						case 4:
							success = int32(targetValue) > int32(ki.Value)
						case 8:
							success = int64(targetValue) > int64(ki.Value)
						}
					} else {
						success = targetValue > uint64(ki.Value)
					}
				}

				if success {
					globalOffset = lookupOffset + int64(ki.ByteWidth)
				}
			}

		case wizparser.KindFamilyString:
			sk, _ := rule.Kind.Data.(*wizparser.StringKind)

			matchLen := wizardry.StringTest(target, int(lookupOffset), sk.Value, sk.Flags)
			success = matchLen >= 0

			if sk.Negate {
				success = !success
			} else {
				if success {
					globalOffset = lookupOffset + int64(matchLen)
				}
			}

		case wizparser.KindFamilySearch:
			sk, _ := rule.Kind.Data.(*wizparser.SearchKind)

			matchPos := wizardry.SearchTest(target, int(lookupOffset), sk.MaxLen, string(sk.Value))
			success = matchPos >= 0

			if success {
				globalOffset = int64(matchPos + len(sk.Value))
			}

		case wizparser.KindFamilyDefault:
			// default tests match if nothing has matched before
			if !everMatchedLevels[rule.Level] {
				success = true
			}

		case wizparser.KindFamilyUse:
			uk, _ := rule.Kind.Data.(*wizparser.UseKind)

			ctx.Logf("|====> using %s", uk.Page)

			subStrings, err := ctx.identifyInternal(target, lookupOffset, uk.Page, uk.SwapEndian)
			if err != nil {
				return nil, err
			}
			outStrings = append(outStrings, subStrings...)

		case wizparser.KindFamilyClear:
			everMatchedLevels[rule.Level] = false
		}

		if success {
			descString := string(rule.Description)

			ctx.Logf("|==========> rule matched!")

			if descString != "" {
				outStrings = append(outStrings, descString)
			}
			matchedLevels[rule.Level] = true
			everMatchedLevels[rule.Level] = true
		} else {
			matchedLevels[rule.Level] = false
		}
	}

	ctx.Logf("|====> done identifying at %d using page %s (%d rules)", pageOffset, page, len(ctx.Book[page]))

	return outStrings, nil
}

func readAnyUint(input []byte, j int, byteWidth int, endianness wizparser.Endianness) (uint64, error) {
	if j+byteWidth >= len(input) {
		return 0, fmt.Errorf("not enough bytes in input to read uint (we'd have to read at %d, only got %d)", j+byteWidth, len(input))
	}

	var ret uint64
	intBytes := input[j : j+byteWidth]

	switch byteWidth {
	case 1:
		ret = uint64(input[j])
	case 2:
		ret = uint64(endianness.ByteOrder().Uint16(intBytes))
	case 4:
		ret = uint64(endianness.ByteOrder().Uint32(intBytes))
	case 8:
		ret = uint64(endianness.ByteOrder().Uint64(intBytes))
	default:
		return 0, fmt.Errorf("dunno how to read an uint of %d bytes", byteWidth)
	}

	return ret, nil
}
