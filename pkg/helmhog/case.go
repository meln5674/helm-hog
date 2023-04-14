package helmhog

type MappingSet map[VariableName]ChoiceName

type Case MappingSet

func (c Case) With(name VariableName, choice ChoiceName) Case {
	newC := make(Case, len(c))
	for k, v := range c {
		newC[k] = v
	}
	newC[name] = choice
	return newC
}
