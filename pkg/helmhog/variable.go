package helmhog

import "encoding/json"

type PartName = string

type VariableName = string

type ChoiceName = string

type Part = json.RawMessage

type Variable = map[ChoiceName]Choice

type Choice = []PartName
