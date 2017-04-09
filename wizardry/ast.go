package wizardry

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// Spellbook contains a set of rules - at least one "" page, potentially others
type Spellbook map[string][]Rule

// AddRule appends a rule to the spellbook on the given page
func (sb Spellbook) AddRule(page string, rule Rule) {
	sb[page] = append(sb[page], rule)
}

// Rule is a single magic rule
type Rule struct {
	Level       int
	Offset      Offset
	Kind        Kind
	Description []byte
}

func (r Rule) String() string {
	return fmt.Sprintf("%s%s    %s    %s",
		strings.Repeat(">", r.Level),
		r.Offset, r.Kind, r.Description)
}

func (o Offset) String() string {
	s := ""

	switch o.OffsetType {
	case OffsetTypeDirect:
		s = fmt.Sprintf("0x%x", o.Direct)
	case OffsetTypeIndirect:
		s = "("
		indirect := o.Indirect
		if indirect.IsRelative {
			s += "&"
		}

		s += fmt.Sprintf("0x%x", indirect.OffsetAddress)
		s += "."

		switch indirect.ByteWidth {
		case 1:
			s += "byte"
		case 2:
			s += "short"
		case 4:
			s += "long"
		case 8:
			s += "quad"
		}
		if indirect.Endianness == LittleEndian {
			s += "le"
		} else {
			s += "be"
		}

		switch indirect.OffsetAdjustmentType {
		case OffsetAdjustmentAdd:
			s += "+"
		case OffsetAdjustmentSub:
			s += "-"
		case OffsetAdjustmentMul:
			s += "*"
		case OffsetAdjustmentDiv:
			s += "/"
		}

		if indirect.OffsetAdjustmentType != OffsetAdjustmentNone {
			if indirect.OffsetAdjustmentIsRelative {
				s += "("
			}
			s += fmt.Sprintf("%d", indirect.OffsetAdjustmentValue)
			if indirect.OffsetAdjustmentIsRelative {
				s += ")"
			}
		}

		s += ")"
	}

	if o.IsRelative {
		s = "&" + s
	}
	return s
}

func (k Kind) String() string {
	switch k.Family {
	case KindFamilyInteger:
		ik, _ := k.Data.(*IntegerKind)
		s := ""
		if !ik.Signed {
			s += "u"
		}
		switch ik.ByteWidth {
		case 1:
			s += "byte"
		case 2:
			s += "short"
		case 4:
			s += "long"
		case 8:
			s += "quad"
		}
		if ik.Endianness == LittleEndian {
			s += "le"
		} else {
			s += "be"
		}
		s += "    "
		s += fmt.Sprintf("%x", ik.Value)
		if ik.DoAnd {
			s += fmt.Sprintf("&0x%x", ik.AndValue)
		}
		return s
	case KindFamilyString:
		sk, _ := k.Data.(*StringKind)
		return fmt.Sprintf("string    %s", strconv.Quote(string(sk.Value)))
	case KindFamilySearch:
		sk, _ := k.Data.(*SearchKind)
		return fmt.Sprintf("search/0x%x    %s", sk.MaxLen, strconv.Quote(string(sk.Value)))
	case KindFamilyDefault:
		return "default"
	case KindFamilyClear:
		return "clear"
	case KindFamilyUse:
		uk, _ := k.Data.(*UseKind)
		s := "use   "
		if uk.SwapEndian {
			s += "\\^"
		}
		s += uk.Page
		return s
	default:
		return fmt.Sprintf("kind family %d", k.Family)
	}
}

// Endianness describes the order in which a multi-byte number is stored
type Endianness int

// ByteOrder translates our in-house Endianness constant into a binary.ByteOrder decoder
func (en Endianness) ByteOrder() binary.ByteOrder {
	if en == BigEndian {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

// Swapped returns LittleEndian if you give it BigEndian, and vice versa
func (en Endianness) Swapped() Endianness {
	if en == BigEndian {
		return LittleEndian
	}
	return BigEndian
}

// MaybeSwapped returns swapped endianness if swap is true
func (en Endianness) MaybeSwapped(swap bool) Endianness {
	if !swap {
		return en
	}
	return en.Swapped()
}

const (
	// LittleEndian numbers are stored with the least significant byte first
	LittleEndian Endianness = iota
	// BigEndian numbers are stored with the most significant byte first
	BigEndian = iota
)

// Kind describes the type of tests a magic rule performs
type Kind struct {
	Family KindFamily
	Data   interface{}
}

// IntegerKind describes how to perform a test on an integer
type IntegerKind struct {
	ByteWidth   int
	Endianness  Endianness
	Signed      bool
	DoAnd       bool
	AndValue    uint64
	IntegerTest IntegerTest
	Value       int64
	MatchAny    bool
}

// IntegerTest describes which comparison to perform on an integer
type IntegerTest int

const (
	// IntegerTestEqual tests that two integers have the same value
	IntegerTestEqual IntegerTest = iota
	// IntegerTestNotEqual tests that two integers have different values
	IntegerTestNotEqual = iota
	// IntegerTestLessThan tests that one integer is less than the other
	IntegerTestLessThan = iota
	// IntegerTestGreaterThan tests that one integer is greater than the other
	IntegerTestGreaterThan = iota
)

// StringKind describes how to match a string pattern
type StringKind struct {
	Value  []byte
	Negate bool
	Flags  stringTestFlags
}

// SearchKind describes how to look for a fixed pattern
type SearchKind struct {
	Value  []byte
	MaxLen int
}

// KindFamily groups tests in families (all integer tests, for example)
type KindFamily int

const (
	// KindFamilyInteger tests numbers for equality, inequality, etc.
	KindFamilyInteger KindFamily = iota
	// KindFamilyString looks for a string, with casing and whitespace rules
	KindFamilyString = iota
	// KindFamilySearch looks for a precise string in a slice of the target
	KindFamilySearch = iota
	// KindFamilyDefault succeeds if no tests succeeded before on that level
	KindFamilyDefault = iota
	// KindFamilyClear resets the matched flag for that level
	KindFamilyClear = iota
	// KindFamilyUse acts like a subroutine call, to peruse another page of rules
	KindFamilyUse = iota
)

// Offset describes where to look to compare something
type Offset struct {
	OffsetType OffsetType
	IsRelative bool
	Direct     int64
	Indirect   *IndirectOffset
}

// OffsetType describes whether an offset is direct or indirect
type OffsetType int

const (
	// OffsetTypeIndirect is an offset read from somewhere in a file
	OffsetTypeIndirect OffsetType = iota
	// OffsetTypeDirect is an offset directly specified by the magic
	OffsetTypeDirect = iota
)

// IndirectOffset indicates where to look in a file to find the real offset
type IndirectOffset struct {
	IsRelative                 bool
	ByteWidth                  int
	Endianness                 Endianness
	OffsetAddress              int64
	OffsetAdjustmentType       OffsetAdjustment
	OffsetAdjustmentIsRelative bool
	OffsetAdjustmentValue      int64
}

// OffsetAdjustment describes which operation to apply to an offset
type OffsetAdjustment int

const (
	// OffsetAdjustmentNone is a no-op
	OffsetAdjustmentNone OffsetAdjustment = iota
	// OffsetAdjustmentAdd adds a value
	OffsetAdjustmentAdd = iota
	// OffsetAdjustmentSub subtracts a value
	OffsetAdjustmentSub = iota
	// OffsetAdjustmentMul multiplies by a value
	OffsetAdjustmentMul = iota
	// OffsetAdjustmentDiv divides by a value
	OffsetAdjustmentDiv = iota
)

// UseKind describes which page of the spellbook to use, and whether or not to swap endianness
type UseKind struct {
	SwapEndian bool
	Page       string
}
