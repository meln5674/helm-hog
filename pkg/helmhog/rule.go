package helmhog

type RuleName string

type Requirement struct {
	If   MappingSet `json:"if"`
	Then MappingSet `json:"then"`
}

func (r *Requirement) Allows(c Case) bool {
	for k, v := range r.If {
		if c[k] != v {
			return true
		}
	}
	for k, v := range r.Then {
		if c[k] != v {
			return false
		}
	}
	return true
}

type Restriction MappingSet

func (r Restriction) Allows(c Case) bool {
	for k, v := range r {
		if c[k] != v {
			return true
		}
	}
	return false
}
