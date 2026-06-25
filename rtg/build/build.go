package build

import (
	"sort"

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
	sort.Slice(units, func(i int, j int) bool {
		return units[i].ImportPath < units[j].ImportPath
	})
	return units, nil
}
