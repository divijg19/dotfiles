package tool

func Load(gobin string) ([]Tool, error) {
	candidates, err := discover(gobin)
	if err != nil {
		return nil, err
	}

	var tools []Tool
	for _, c := range candidates {
		t, err := inspect(c)
		if err != nil {
			continue
		}
		tools = append(tools, t)
	}

	return tools, nil
}
