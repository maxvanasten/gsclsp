package analysis

import (
	p "github.com/maxvanasten/gscp/parser"
)

type FunctionSignature struct {
	Name      string
	Arguments []string
}

func GenerateFunctionSignatures(nodes []p.Node) []FunctionSignature {
	functionSignatures := []FunctionSignature{}
	for _, n := range nodes {
		if n.Type == "function_declaration" {
			functionSignatures = append(functionSignatures, signatureFromNode(n))
		}

		// Safely recurse into children - avoid index out of bounds
		if len(n.Children) > 0 {
			functionSignatures = append(functionSignatures, GenerateFunctionSignatures(n.Children)...)
		}
	}

	return functionSignatures
}

func signatureFromNode(n p.Node) FunctionSignature {
	sig := FunctionSignature{
		Name:      n.Data.FunctionName,
		Arguments: []string{},
	}
	if len(n.Children) > 0 && len(n.Children[0].Children) > 0 {
		for _, c := range n.Children[0].Children {
			sig.Arguments = append(sig.Arguments, c.Data.VarName)
		}
	}
	return sig
}
