package wizcompiler

import (
	"fmt"

	"github.com/fasterthanlime/wizardry/wizardry/wizparser"
)

func switchify(node *ruleNode) *ruleNode {
	var lastChild *ruleNode
	var streak []*ruleNode

	endStreak := func() {
		if len(streak) == 0 {
			return
		}

		fmt.Printf("Found streak of %d switchify candidates\n", len(streak))
		for _, s := range streak {
			fmt.Printf("  - %s\n", s.rule.Line)
		}
		streak = nil
	}

	for _, child := range node.children {
		candidate := false

		if child.rule.Kind.Family == wizparser.KindFamilyInteger {
			ik, _ := child.rule.Kind.Data.(*wizparser.IntegerKind)
			if ik.IntegerTest == wizparser.IntegerTestEqual && !ik.DoAnd {
				candidate = true
			}
		}

		if !candidate {
			endStreak()
		} else {
			if len(streak) > 0 {
				if !lastChild.rule.Offset.Equals(child.rule.Offset) {
					endStreak()
				}
				ik, _ := child.rule.Kind.Data.(*wizparser.IntegerKind)
				jk, _ := lastChild.rule.Kind.Data.(*wizparser.IntegerKind)
				if ik.ByteWidth != jk.ByteWidth {
					endStreak()
				}
				if ik.Signed != jk.Signed {
					endStreak()
				}
			}
			streak = append(streak, child)
		}

		lastChild = child
	}

	return node
}
