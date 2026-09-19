package baton

import "fmt"

type arguments struct {
	values map[string]string
	flags  map[string]bool
	pos    []string
}

func parseArguments(args []string, valueFlags, boolFlags map[string]bool) (arguments, error) {
	parsed := arguments{values: map[string]string{}, flags: map[string]bool{}}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if valueFlags[arg] {
			if i+1 >= len(args) {
				return parsed, fmt.Errorf("missing value for %s", arg)
			}
			i++
			parsed.values[arg] = args[i]
			continue
		}
		if boolFlags[arg] {
			parsed.flags[arg] = true
			continue
		}
		if len(arg) > 0 && arg[0] == '-' {
			return parsed, fmt.Errorf("unknown argument: %s", arg)
		}
		parsed.pos = append(parsed.pos, arg)
	}
	return parsed, nil
}

func requireValue(parsed arguments, name string) (string, error) {
	value := parsed.values[name]
	if value == "" {
		return "", fmt.Errorf("missing %s", name)
	}
	return value, nil
}
