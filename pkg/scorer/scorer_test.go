package scorer

import (
	"reflect"
	"testing"
)

func TestSignificantWords_FiltersStopWordsAndShortWords(t *testing.T) {
	got := SignificantWords("the price of a coffee")
	want := []string{"price", "coffee"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SignificantWords = %v, want %v", got, want)
	}
}

func TestSignificantWords_CountsRunesNotBytes(t *testing.T) {
	// Single-letter Cyrillic function words are 2 bytes but 1 rune — they must
	// be dropped just like single-letter Latin words, or word-overlap scoring
	// on Ukrainian queries is diluted by «і»/«в»/«з»/«у».
	got := SignificantWords("пошук і фільтри в каталозі")
	want := []string{"пошук", "фільтри", "каталозі"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SignificantWords = %v, want %v", got, want)
	}
	// Two-letter Cyrillic words stay.
	got = SignificantWords("на головну")
	want = []string{"на", "головну"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SignificantWords = %v, want %v", got, want)
	}
}
