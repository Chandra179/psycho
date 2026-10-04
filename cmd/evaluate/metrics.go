package main

import "psycho/modules/supervised"

func averageRanks(v []float64) []float64      { return supervised.AverageRanks(v) }
func spearman(x, y []float64) float64         { return supervised.Spearman(x, y) }
func aucBinary(s []float64, y []bool) float64 { return supervised.AUCBinary(s, y) }
func aucBootstrapCI(s []float64, y []bool, n int, rng interface{ Intn(int) int }) (float64, float64) {
	return supervised.AUCBootstrapCI(s, y, n, rng)
}
