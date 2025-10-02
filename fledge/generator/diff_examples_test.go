package generator_test

import (
	"fmt"

	"github.com/simonhull/firebird-suite/fledge/generator"
)

func ExampleGenerateDiffDefault() {
	old := []byte("line 1\nline 2\nline 3\n")
	newer := []byte("line 1\nline 2\nline 2.5\nline 3\n")

	diff := generator.GenerateDiffDefault("old.txt", "new.txt", old, newer)
	fmt.Println("Diff generated successfully")
	_ = diff // Use the diff
	// Output: Diff generated successfully
}

func ExampleDiffGenerator() {
	gen := generator.NewDiffGenerator()

	// Generate multiple diffs efficiently
	diff1 := gen.GenerateDiffDefault("a.txt", "a.txt", []byte("old"), []byte("new"))
	diff2 := gen.GenerateDiffDefault("b.txt", "b.txt", []byte("foo"), []byte("bar"))

	fmt.Printf("Generated %d diffs", 2)
	_, _ = diff1, diff2 // Use the diffs
	// Output: Generated 2 diffs
}

func ExampleGenerateDiff_withLineNumbers() {
	old := []byte("func main() {\n\tfmt.Println(\"old\")\n}\n")
	newer := []byte("func main() {\n\tfmt.Println(\"new\")\n}\n")

	opts := &generator.DiffOptions{
		ContextLines: 2,
		TabWidth:     4,
		ShowLineNums: true,
	}

	diff := generator.GenerateDiff("main.go", "main.go", old, newer, opts)
	fmt.Println("Diff with line numbers generated")
	_ = diff // Use the diff
	// Output: Diff with line numbers generated
}
