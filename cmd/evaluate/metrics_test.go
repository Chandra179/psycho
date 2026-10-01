package main

import (
	"math"
	"math/rand"
	"testing"
)

func TestAverageRanksWithTies(t *testing.T) {
	got := averageRanks([]float64{0.1, 0.2, 0.2, 0.3})
	want := []float64{1, 2.5, 2.5, 4}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rank[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestSpearmanPerfectAndInverse(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5}
	if got := spearman(x, []float64{2, 4, 6, 8, 10}); got < 0.9999 {
		t.Fatalf("monotone increasing spearman = %v, want ~1", got)
	}
	if got := spearman(x, []float64{10, 8, 6, 4, 2}); got > -0.9999 {
		t.Fatalf("monotone decreasing spearman = %v, want ~-1", got)
	}
}

func TestSpearmanKnownValue(t *testing.T) {
	// Classic textbook example: ranks of x vs ranks of y give ρ = 0.5.
	got := spearman([]float64{1, 2, 3, 4}, []float64{1, 3, 2, 4})
	if math.Abs(got-0.8) > 1e-9 { // 1 - 6*Σd²/(n(n²-1)) with d² = 0,1,1,0 → 1-12/60 = 0.8
		t.Fatalf("spearman = %v, want 0.8", got)
	}
}

func TestSpearmanDegenerate(t *testing.T) {
	if !math.IsNaN(spearman([]float64{1, 1, 1}, []float64{1, 2, 3})) {
		t.Fatal("constant input should give NaN, not a fake correlation")
	}
	if !math.IsNaN(spearman([]float64{1}, []float64{1})) {
		t.Fatal("n=1 should give NaN")
	}
}

func TestAUCBinary(t *testing.T) {
	cases := []struct {
		name   string
		scores []float64
		pos    []bool
		want   float64
	}{
		{"perfect", []float64{1, 2, 3, 4}, []bool{false, false, true, true}, 1},
		{"inverted", []float64{1, 2, 3, 4}, []bool{true, true, false, false}, 0},
		{"chance", []float64{1, 2, 3, 4}, []bool{false, true, true, false}, 0.5},
		{"ties", []float64{1, 1, 2, 2}, []bool{false, true, false, true}, 0.5},
		{"ties-win", []float64{1, 1, 2, 2}, []bool{false, false, true, true}, 1},
	}
	for _, c := range cases {
		if got := aucBinary(c.scores, c.pos); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s: auc = %v, want %v", c.name, got, c.want)
		}
	}
	if !math.IsNaN(aucBinary([]float64{1, 2}, []bool{true, true})) {
		t.Error("single-class input should give NaN")
	}
}

func TestAUCBootstrapCI(t *testing.T) {
	// A well-separated sample must produce a CI well above chance.
	scores := []float64{1, 2, 3, 4, 5, 6, 7, 8}
	pos := []bool{false, false, false, false, true, true, true, true}
	rng := rand.New(rand.NewSource(7))
	lo, hi := aucBootstrapCI(scores, pos, 500, rng)
	if !(0.75 <= lo && lo <= hi && hi <= 1.0) {
		t.Fatalf("CI = [%v, %v], expected a tight high interval", lo, hi)
	}
}

func TestCP1252Conversion(t *testing.T) {
	// 0x92 is a right single quote in cp1252, 0x93/0x94 are double quotes.
	got := cp1252ToUTF8("don\x92t \x93quoted\x94")
	if got != "don\u2019t \u201Cquoted\u201D" {
		t.Fatalf("converted %q", got)
	}
	if cp1252ToUTF8("plain ascii") != "plain ascii" {
		t.Fatal("ascii passthrough changed")
	}
}
