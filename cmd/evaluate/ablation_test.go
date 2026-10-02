package main

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
)

func TestPickCategoriesIsDeterministicSubset(t *testing.T) {
	all := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	got1 := pickCategories(all, 4, rand.New(rand.NewSource(7)))
	got2 := pickCategories(all, 4, rand.New(rand.NewSource(7)))
	if !reflect.DeepEqual(got1, got2) {
		t.Fatalf("same seed gave different subsets: %v vs %v", got1, got2)
	}
	if len(got1) != 4 {
		t.Fatalf("want 4 categories, got %d", len(got1))
	}
	seen := map[string]bool{}
	for _, c := range got1 {
		if seen[c] {
			t.Fatalf("duplicate category %q", c)
		}
		seen[c] = true
	}
}

func TestPickCategoriesDoesNotMutateInput(t *testing.T) {
	all := []string{"a", "b", "c", "d"}
	want := append([]string(nil), all...)
	pickCategories(all, 2, rand.New(rand.NewSource(1)))
	if !reflect.DeepEqual(all, want) {
		t.Fatalf("input mutated: %v", all)
	}
}

func TestSplitCategoriesPartitions(t *testing.T) {
	all := []string{"article", "pronoun", "social", "time"}
	f, c := splitCategories(all)
	if !reflect.DeepEqual(f, []string{"article", "pronoun"}) {
		t.Fatalf("function cats = %v", f)
	}
	if !reflect.DeepEqual(c, []string{"social", "time"}) {
		t.Fatalf("content cats = %v", c)
	}
	if len(f)+len(c) != len(all) {
		t.Fatal("split is not a partition")
	}
}

func TestProjectAUC2x(t *testing.T) {
	// base 0.55, 50% level 0.53: slope 0.02 per 50% -> +0.04 at 200%.
	if got := projectAUC2x(0.55, 0.53, 50); math.Abs(got-0.59) > 1e-9 {
		t.Fatalf("want 0.59, got %v", got)
	}
	// Steep slope from a low level clamps at 1: 0.55 + (0.55−0.05)×2 = 1.55.
	if got := projectAUC2x(0.55, 0.05, 50); got != 1 {
		t.Fatalf("want clamped 1, got %v", got)
	}
	// Degenerate low level is NaN.
	if got := projectAUC2x(0.55, 0.53, 100); !math.IsNaN(got) {
		t.Fatalf("want NaN at level 100, got %v", got)
	}
}

func TestParseLevels(t *testing.T) {
	levels, err := parseLevels("75, 50,100,50")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(levels, []int{75, 50}) {
		t.Fatalf("want [75 50] (sorted desc, 100 dropped, duplicates removed), got %v", levels)
	}
	if _, err := parseLevels("0"); err == nil {
		t.Fatal("want error for level 0")
	}
	if _, err := parseLevels("100"); err == nil {
		t.Fatal("want error when no level is below 100")
	}
}

func TestDictFromRawKeepsOnlyNamedCategories(t *testing.T) {
	raw := map[string][]string{
		"social":  {"friend", "talk"},
		"time":    {"yesterday"},
		"emotion": {"happy"},
	}
	dict, err := dictFromRaw(raw, []string{"social", "time"})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(dict.Categories()); got != 2 {
		t.Fatalf("want 2 categories, got %d", got)
	}
	if cats := dict.Lookup("happy"); len(cats) != 0 {
		t.Fatalf("word from a dropped category leaked: %v", cats)
	}
	if cats := dict.Lookup("friend"); len(cats) != 1 || cats[0] != "social" {
		t.Fatalf("kept-word lookup wrong: %v", cats)
	}
	if _, err := dictFromRaw(raw, []string{"nope"}); err == nil {
		t.Fatal("want error for unknown category")
	}
}
