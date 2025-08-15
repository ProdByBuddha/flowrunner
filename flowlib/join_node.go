package flowlib

import "fmt"

/* ---------- JoinNode (collect parallel results) ---------- */

type JoinNode struct {
	baseNode
}

func NewJoinNode() *JoinNode {
	return &JoinNode{newBaseNode()}
}

func (jn *JoinNode) Run(shared any) (Action, error) {
	// Get the format parameter (default to "array")
	format := "array"
	if params := jn.Params(); params != nil {
		if f, ok := params["format"].(string); ok {
			format = f
		}
	}

    // The shared context should contain collected results from parallel execution
    // This is typically set by the SplitNode or AsyncSplitNode
    if sharedMap, ok := shared.(map[string]any); ok {
        // Prefer mapper_results (set by SplitNode collector), fallback to legacy _parallel_results
        var results any
        if r, exists := sharedMap["mapper_results"]; exists {
            results = r
        } else if r, exists := sharedMap["_parallel_results"]; exists {
            results = r
        }

        if resultSlice, ok := results.([]any); ok {
            // Format the results based on the requested format
            switch format {
            case "array":
                sharedMap["_join_output"] = resultSlice
            case "object":
                obj := make(map[string]any)
                for i, result := range resultSlice {
                    obj[fmt.Sprintf("result_%d", i)] = result
                }
                sharedMap["_join_output"] = obj
            case "map":
                obj := make(map[string]any)
                for i, result := range resultSlice {
                    key := fmt.Sprintf("branch_%d", i)
                    if resultMap, ok := result.(map[string]any); ok {
                        if branch, hasBranch := resultMap["branch"]; hasBranch {
                            if branchStr, ok := branch.(string); ok {
                                key = branchStr
                            }
                        }
                    }
                    obj[key] = result
                }
                sharedMap["_join_output"] = obj
            default:
                return "", fmt.Errorf("unsupported join format: %s", format)
            }

            // Set the joined results as the current input for the next node
            if joinOutput, exists := sharedMap["_join_output"]; exists {
                sharedMap["input"] = joinOutput
            }
            return DefaultAction, nil
        }

        // If no parallel results found, return empty based on format
        switch format {
        case "array":
            sharedMap["input"] = []any{}
        case "object", "map":
            sharedMap["input"] = map[string]any{}
        }
    }

	return DefaultAction, nil
}
