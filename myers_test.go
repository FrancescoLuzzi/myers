package diff

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMyersDiff(t *testing.T) {
	from := "A\nB\nC\nA\nB\nB\nA"
	to := "C\nB\nA\nB\nA\nC\nC"
	expectedEdits := []Edit{
		{oldLine: 0, newLine: -1, content: "A", op: OPDelete},
		{oldLine: 1, newLine: -1, content: "B", op: OPDelete},
		{oldLine: 2, newLine: 0, content: "C", op: OPEqual},
		{oldLine: -1, newLine: 1, content: "B", op: OPAdd},
		{oldLine: 3, newLine: 2, content: "A", op: OPEqual},
		{oldLine: 4, newLine: 3, content: "B", op: OPEqual},
		{oldLine: 5, newLine: -1, content: "B", op: OPDelete},
		{oldLine: 6, newLine: 4, content: "A", op: OPEqual},
		{oldLine: -1, newLine: 5, content: "C", op: OPAdd},
		{oldLine: -1, newLine: 6, content: "C", op: OPAdd},
	}
	edits := slices.Collect(MyersDiff(from, to))
	assert.Equal(t, expectedEdits, edits)
}

func TestMyersDiff2(t *testing.T) {
	from := "A\nA\nA\nB\nC\nA\nB\nB\nA"
	to := "A\nA\nC\nB\nA\nB\nA\nC\nC"
	expectedEdits := []Edit{
		{oldLine: 0, newLine: 0, content: "A", op: OPEqual},
		{oldLine: 1, newLine: 1, content: "A", op: OPEqual},
		{oldLine: 2, newLine: -1, content: "A", op: OPDelete},
		{oldLine: 3, newLine: -1, content: "B", op: OPDelete},
		{oldLine: 4, newLine: 2, content: "C", op: OPEqual},
		{oldLine: -1, newLine: 3, content: "B", op: OPAdd},
		{oldLine: 5, newLine: 4, content: "A", op: OPEqual},
		{oldLine: 6, newLine: 5, content: "B", op: OPEqual},
		{oldLine: 7, newLine: -1, content: "B", op: OPDelete},
		{oldLine: 8, newLine: 6, content: "A", op: OPEqual},
		{oldLine: -1, newLine: 7, content: "C", op: OPAdd},
		{oldLine: -1, newLine: 8, content: "C", op: OPAdd},
	}
	edits := slices.Collect(MyersDiff(from, to))
	assert.Equal(t, expectedEdits, edits)
}

func TestEditsToHunkString(t *testing.T) {
	from := "A\nA\nA\nB\nC\nA\nB\nB\nA"
	to := "A\nA\nC\nB\nA\nB\nA\nC\nC"
	expectedHunk := `@@ -1,9 +1,9 @@
A
A
-A
-B
C
+B
A
B
-B
A
+C
+C
`
	edits := slices.Collect(MyersDiff(from, to))
	hunk := EditsToHunkString(edits)
	assert.Equal(t, expectedHunk, hunk)
}
