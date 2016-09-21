package patch

import (
	"fmt"
)

type RemoveOp struct {
	Path Pointer
}

func (op RemoveOp) Apply(doc interface{}) (interface{}, error) {
	tokens := op.Path.Tokens()

	if len(tokens) == 1 {
		return nil, fmt.Errorf("Cannot remove entire document")
	}

	obj := doc
	prevUpdate := func(newObj interface{}) { doc = newObj }

	for i, token := range tokens[1:] {
		isLast := i == len(tokens)-2

		switch typedToken := token.(type) {
		case IndexToken:
			idx := typedToken.Index

			typedObj, ok := obj.([]interface{})
			if !ok {
				return nil, newOpArrayMismatchTypeErr(tokens[:i+2], obj)
			}

			if idx >= len(typedObj) {
				errMsg := "Expected to find array index '%d' but found array of length '%d'"
				return nil, fmt.Errorf(errMsg, idx, len(typedObj))
			}

			if isLast {
				var newAry []interface{}
				newAry = append(newAry, typedObj[:idx]...)
				newAry = append(newAry, typedObj[idx+1:]...)
				prevUpdate(newAry)
			} else {
				obj = typedObj[idx]
				prevUpdate = func(newObj interface{}) { typedObj[idx] = newObj }
			}

		case MatchingIndexToken:
			typedObj, ok := obj.([]interface{})
			if !ok {
				return nil, newOpArrayMismatchTypeErr(tokens[:i+2], obj)
			}

			var idxs []int

			for itemIdx, item := range typedObj {
				typedItem, ok := item.(map[interface{}]interface{})
				if ok {
					if typedItem[typedToken.Key] == typedToken.Value {
						idxs = append(idxs, itemIdx)
					}
				}
			}

			if len(idxs) != 1 {
				errMsg := "Expected to find exactly one matching array item for path '%s' but found %d"
				return nil, fmt.Errorf(errMsg, NewPointer(tokens[:i+2]), len(idxs))
			}

			idx := idxs[0]

			if isLast {
				var newAry []interface{}
				newAry = append(newAry, typedObj[:idx]...)
				newAry = append(newAry, typedObj[idx+1:]...)
				prevUpdate(newAry)
			} else {
				obj = typedObj[idx]
				// no need to change prevUpdate since matching item can only be a map
			}

		case KeyToken:
			typedObj, ok := obj.(map[interface{}]interface{})
			if !ok {
				return nil, newOpMapMismatchTypeErr(tokens[:i+2], obj)
			}

			var found bool

			obj, found = typedObj[typedToken.Key]
			if !found {
				if typedToken.Optional {
					return doc, nil
				}

				errMsg := "Expected to find a map key '%s' for path '%s'"
				return nil, fmt.Errorf(errMsg, typedToken.Key, NewPointer(tokens[:i+2]))
			}

			if isLast {
				delete(typedObj, typedToken.Key)
			} else {
				prevUpdate = func(newObj interface{}) { typedObj[typedToken.Key] = newObj }
			}

		default:
			return nil, fmt.Errorf("Expected to not find token '%T' at '%s'", token, NewPointer(tokens[:i+2]))
		}
	}

	return doc, nil
}
