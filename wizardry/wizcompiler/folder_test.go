package wizcompiler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Fold(t *testing.T) {
	{
		node := &BinaryOp{
			LHS:      &NumberLiteral{1},
			Operator: OperatorAdd,
			RHS:      &NumberLiteral{3},
		}

		assert.EqualValues(t, "1+3", node.String())
		assert.EqualValues(t, "4", node.Fold().String())
	}
	{
		node := &BinaryOp{
			LHS:      &NumberLiteral{3},
			Operator: OperatorSub,
			RHS:      &NumberLiteral{1},
		}

		assert.EqualValues(t, "3-1", node.String())
		assert.EqualValues(t, "2", node.Fold().String())
	}
	{
		node := &BinaryOp{
			LHS:      &NumberLiteral{1},
			Operator: OperatorAdd,
			RHS: &BinaryOp{
				LHS:      &NumberLiteral{2},
				Operator: OperatorAdd,
				RHS:      &NumberLiteral{3},
			},
		}
		assert.EqualValues(t, "1+2+3", node.String())
		assert.EqualValues(t, "6", node.Fold().String())
	}
	{
		node := &BinaryOp{
			LHS:      &NumberLiteral{2},
			Operator: OperatorMul,
			RHS: &BinaryOp{
				LHS:      &NumberLiteral{3},
				Operator: OperatorAdd,
				RHS:      &NumberLiteral{4},
			},
		}
		assert.EqualValues(t, "2*(3+4)", node.String())
		assert.EqualValues(t, "14", node.Fold().String())
	}
	{
		node := &BinaryOp{
			LHS: &BinaryOp{
				LHS:      &NumberLiteral{3},
				Operator: OperatorAdd,
				RHS:      &NumberLiteral{4},
			},
			Operator: OperatorMul,
			RHS:      &NumberLiteral{2},
		}
		assert.EqualValues(t, "(3+4)*2", node.String())
		assert.EqualValues(t, "14", node.Fold().String())
	}
	{
		node := &BinaryOp{
			LHS:      &VariableAccess{"x"},
			Operator: OperatorAdd,
			RHS: &BinaryOp{
				LHS:      &NumberLiteral{3},
				Operator: OperatorAdd,
				RHS:      &NumberLiteral{2},
			},
		}
		assert.EqualValues(t, "x+3+2", node.String())
		assert.EqualValues(t, "x+5", node.Fold().String())
	}
	{
		node := &BinaryOp{
			LHS:      &NumberLiteral{2},
			Operator: OperatorAdd,
			RHS: &BinaryOp{
				LHS:      &NumberLiteral{3},
				Operator: OperatorAdd,
				RHS:      &VariableAccess{"x"},
			},
		}
		assert.EqualValues(t, "2+3+x", node.String())
		assert.EqualValues(t, "5+x", node.Fold().String())
	}
	{
		node := &BinaryOp{
			LHS:      &NumberLiteral{2},
			Operator: OperatorAdd,
			RHS: &BinaryOp{
				LHS:      &VariableAccess{"x"},
				Operator: OperatorAdd,
				RHS:      &NumberLiteral{3},
			},
		}
		assert.EqualValues(t, "2+x+3", node.String())
		assert.EqualValues(t, "x+5", node.Fold().String())
	}
	{
		node := &BinaryOp{
			LHS: &BinaryOp{
				LHS:      &NumberLiteral{3},
				Operator: OperatorAdd,
				RHS:      &VariableAccess{"x"},
			},
			Operator: OperatorAdd,
			RHS:      &NumberLiteral{2},
		}
		assert.EqualValues(t, "3+x+2", node.String())
		assert.EqualValues(t, "5+x", node.Fold().String())
	}
	{
		node := &BinaryOp{
			LHS: &BinaryOp{
				LHS:      &VariableAccess{"x"},
				Operator: OperatorAdd,
				RHS:      &NumberLiteral{3},
			},
			Operator: OperatorAdd,
			RHS:      &NumberLiteral{2},
		}
		assert.EqualValues(t, "x+3+2", node.String())
		assert.EqualValues(t, "x+5", node.Fold().String())
	}
	{
		node := &BinaryOp{
			LHS:      &NumberLiteral{3},
			Operator: OperatorAdd,
			RHS:      &NumberLiteral{0},
		}
		assert.EqualValues(t, "3+0", node.String())
		assert.EqualValues(t, "3", node.Fold().String())
	}
	{
		node := &BinaryOp{
			LHS:      &VariableAccess{"x"},
			Operator: OperatorAdd,
			RHS:      &NumberLiteral{0},
		}
		assert.EqualValues(t, "x+0", node.String())
		assert.EqualValues(t, "x", node.Fold().String())
	}
	{
		node := &BinaryOp{
			LHS:      &VariableAccess{"x"},
			Operator: OperatorSub,
			RHS:      &NumberLiteral{0},
		}
		assert.EqualValues(t, "x-0", node.String())
		assert.EqualValues(t, "0", node.Fold().String())
	}
	{
		node := &BinaryOp{
			LHS:      &VariableAccess{"x"},
			Operator: OperatorMul,
			RHS:      &NumberLiteral{0},
		}
		assert.EqualValues(t, "x*0", node.String())
		assert.EqualValues(t, "0", node.Fold().String())
	}
}
