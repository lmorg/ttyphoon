package config

var ligPairs [][]rune = nil

func LigaturePairs() [][]rune {
	if ligPairs == nil {
		for _, s := range Config.TypeFace.LigaturePairs {
			ligPairs = append(ligPairs, []rune(s))
		}
	}

	return ligPairs
}
