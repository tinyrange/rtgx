package build

import (
	"j5.nz/rtg/rtg/check"
	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/lower"
	"j5.nz/rtg/rtg/unit"
)

func Units(graph *load.Graph) ([]unit.Unit, error) {
	if err := check.Graph(graph); err != nil {
		return nil, err
	}
	units := make([]unit.Unit, 0, len(graph.Packages))
	for _, pkg := range graph.Packages {
		u, err := lower.PackageWithGraph(pkg, graph)
		if err != nil {
			return nil, err
		}
		units = append(units, u)
	}
	sortUnitsByImportPath(units)
	return units, nil
}

func sortUnitsByImportPath(units []unit.Unit) {
	for i := 1; i < len(units); i++ {
		value := units[i]
		j := i - 1
		for j >= 0 && units[j].ImportPath > value.ImportPath {
			units[j+1] = units[j]
			j = j - 1
		}
		units[j+1] = value
	}
}
