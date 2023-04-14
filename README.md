# Helm Hog

Clean your helm charts.

Helm Hog is a tool used to quickly test many different combinations of values for a helm chart.

"Hog" comes from a nautical term, referring to a brush used to clean the bottom of a ship.

## Installing

```bash
go install github.com/meln5674/helm-hog@latest
```

## Running

```bash
# Validate your configuration
helm-hog validate
# List all cases to be run
helm-hog list
# Run tests
helm-hog test
```

## Basic concepts

Helm Hog works by quickly generating many combinations of overlayed value files from sets you provide.

### Part

A part is a single helm values.yaml file.

### Choice

A choice is a set of parts

### Variable

A Variable is a mutually exclusive set of named Choices.

### Mapping

A Mapping is a pair of a Variable and one of its Choices.

### Case

A "Case", analogous to unit testing frameworks, is a single generated release of a helm chart that is checked for validity.

A Case consists of one Mapping for every Variable.

A Case "passes" if the chart templates produce no errors and the generated resources pass schema validation from a a kubernetes api server.

### Requirement

A requirement is an "if X then Y" where "X" and "Y" are sets of Mappings, where any case with "X" must also have "Y".

### Restriction

A restriction is a rule consisting of a set of Mappings, where no case may have that particular combination.

### Project

A Project is a Helm Chart, a set of Variables, Requirements, and Restrictions, which define a set of Cases to be executed. A Project "passes" if all Cases "pass".

## Project structure

Helm Hog Projects are defined by a "hog.yaml" file with the following structure:

```yaml
# hog.yaml
apiVersion: helm-hog.meln5674.github.com/v1alpha1
Kind: Project

# By default, assume the hog.yaml is in the same directory as Chart.yaml and values.yaml.
# Otherwise, specify the path to that directory.
chart: path/to/chart

# To define parts on the filesystem, provide one or more directories of files with .yml or .yaml extensions
# The filename becomes the name of the parts, and the contents defines the values.yaml that is contributed to the case
# Directories are not recursively searched, only files directly in the specified directories are used
partsDirs:
- path/to/dir
# ...

# To define parts within the project yaml, provide a map from part name to a nested YAML object containing the values to contribute to the case
parts:
  part-name:
    values: to
    set: [in, the, chart]
  # ...

# To define variables, provide a map from the variable name to the map of choices, which are themselves maps from the choice name to the list of part names
variables:
  variable-name:
    choice-name: [part,names,to,include]
    # ...
  # ...
# Optionally specify an order for variables to be evaluated in.
# If omitted, variables are evaluated in lexigraphical order as defined by golang string comparison
variableOrder: [order,of,variables] 

# To only allow combinations of Mappings when other combinations are also present, provide a map from rule names to their "if" (combination to match) and "then" (combinations to require if "if" is matched)
requirements:
  rule-name:
    if: {variable:choices, to:match}
    then: {variable:choices, to:require}

# To disallow certain combinations of Mappings, provide a map from rule names to the combinations to reject
restrictions:
  rule-name: {variable:choices, to:reject}
```
