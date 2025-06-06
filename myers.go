package diff

import (
	"fmt"
	"iter"
	"slices"
	"strings"
)

type opKind int

const (
	OPEqual opKind = iota
	OPAdd
	OPDelete
)

type Edit struct {
	op               opKind
	oldLine, newLine int
	content          string
}

func (e Edit) String() string {
	switch e.op {
	case OPAdd:
		return fmt.Sprintf("+%s", e.content)
	case OPDelete:
		return fmt.Sprintf("-%s", e.content)
	case OPEqual:
		return e.content
	default:
		panic(fmt.Sprintf("unexpected diff.opKind: %#v", e.op))
	}
}

const eqMaxCounter = 6
const overlap = 3

func EditsToHunkString(edits []Edit) string {
	startEdit := 0
	endEdit := 0
	addedLines := 0
	deletedLines := 0
	inHunk := false
	eqCounter := 0
	builder := strings.Builder{}
	n := 0
OUTER:
	for n < len(edits) {
		edit := edits[n]
		for edit.op == OPEqual && !inHunk {
			n++
			if n == len(edits) {
				break OUTER
			}
			edit = edits[n]
		}
		if inHunk {
			switch edit.op {
			case OPEqual:
				eqCounter++
				if eqCounter == eqMaxCounter {
					endEdit = max(n-(eqMaxCounter-overlap), startEdit)
					writeHunk(&builder, edits, startEdit, endEdit, addedLines, deletedLines)
					inHunk = false
					eqCounter = 0
					addedLines = 0
					deletedLines = 0
				}
			case OPAdd:
				addedLines++
				eqCounter = 0
			case OPDelete:
				deletedLines++
				eqCounter = 0
			}
		} else {
			inHunk = true
			switch edit.op {
			case OPAdd:
				addedLines++
			case OPDelete:
				deletedLines++
			}
			startEdit = max(n-overlap, 0)
		}
		n++
	}
	if inHunk {
		writeHunk(&builder, edits, startEdit, len(edits), addedLines, deletedLines)
	}
	return builder.String()
}

func writeHunk(builder *strings.Builder, edits []Edit, startEdit, endEdit, addedLines, deletedLines int) {
	eqLines := endEdit - startEdit - addedLines - deletedLines
	startLineOld := 0
	startLineNew := 0
	switch edits[startEdit].op {
	case OPEqual:
		startLineOld = edits[startEdit].oldLine + 1
		startLineNew = edits[startEdit].newLine + 1
	default:
		startLineOld = max(edits[startEdit].oldLine, edits[startEdit].newLine) + 1
		startLineNew = startLineOld
	}
	builder.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", startLineOld, eqLines+deletedLines, startLineNew, eqLines+addedLines))
	for i := startEdit; i < endEdit; i++ {
		builder.WriteString(edits[i].String())
		builder.WriteRune('\n')
	}
}

func Seq2Value[T, G any](seq iter.Seq2[T, G]) iter.Seq[G] {
	return func(yield func(G) bool) {
		for _, x := range seq {
			if !yield(x) {
				return
			}
		}
	}
}

func MyersDiff(before, after string) iter.Seq[Edit] {
	beforeLines := splitLines(before)
	afterLines := splitLines(after)
	operations := operations(beforeLines, afterLines)
	return Seq2Value(slices.Backward(backtrack(beforeLines, afterLines, operations)))
}

// The Myers algorithm is based on a matrix:
// - a's chars in the columns (index[0..N])
// - b's chars in the rows (index[0..M])
// moving right represents deleting the char of the column that you are moving from
// moving down represents inserting a char of the column that you are moving from
// moving diagonally represents NOOP (chars are equal)
// the final and optimized algorithm rotates the matrix by 45°
// The coordinate system will become:
// - d represents the x coordinate, the iteration of the algorithm (d[0..N+M])
// - k represents the y coordinate, and it represents the operation for the current character (k[-d..d], depends on the iteration)
// "k+1" represents deleting the char of the column that you are moving from
// "k-1" represents inserting the char of the column that you are moving from
// "k" represents NOOP (chars are equal)
// the cell in the dxk matrix represents the index of the char being deleted from "a" or interted from "b"
// this is the Myers algorithm, by default it returns "d", the min number of changes that transform "a" into "b"
// to expand it to know WHICH changes have been made, we need to keep track of all the changes at each iteration
func operations(a, b []string) [][]int {
	N := len(a)
	M := len(b)
	OFFSET := M + N
	V := make([]int, 2*OFFSET+1)
	traces := make([][]int, OFFSET+1)
	var x, y int
	for d := range OFFSET {
		copyV := make([]int, len(V))
		for k := -d; k <= d; k += 2 {
			if k == -d || (k != d && V[OFFSET+k-1] < V[OFFSET+k+1]) {
				x = V[OFFSET+k+1] // delete
			} else {
				x = V[OFFSET+k-1] + 1 // insert
			}
			y = x - k
			// follow the diagonal
			for x < N && y < M && a[x] == b[y] {
				x++
				y++
			}
			V[OFFSET+k] = x
			if x >= N && y >= M {
				copy(copyV, V)
				traces[d] = copyV
				return traces
			}
		}
		copy(copyV, V)
		traces[d] = copyV
	}
	return nil
}

// given the traces found during the Myers algorithm
// extract all changes done to transmute "a" to "b" in reverse.
// For each "d" in backwards order, figure out the operation that was done
// and try to follow the diagonal to add all the equal chars not explicitly detected
func backtrack(a, b []string, traces [][]int) []Edit {
	x, y := len(a), len(b)
	offset := x + y
	prev_k, prev_x, prev_y := 0, 0, 0
	edits := make([]Edit, 0, x+y+1)
	for d, v := range slices.Backward(traces) {
		if len(v) == 0 {
			continue
		}
		k := x - y
		if k == -d || (k != d && v[offset+k-1] < v[offset+k+1]) {
			prev_k = k + 1

		} else {
			prev_k = k - 1
		}
		prev_x = v[offset+prev_k]
		prev_y = prev_x - prev_k
		// follow the diagonal
		for x > prev_x && y > prev_y && x >= 0 && y >= 0 {
			x -= 1
			y -= 1
			edits = append(edits, Edit{oldLine: x, newLine: y, content: a[x], op: OPEqual})
		}
		// we don't have nothing more to add
		if d == 0 {
			break
		}
		if x == prev_x { //insert
			edits = append(edits, Edit{oldLine: -1, newLine: prev_y, content: b[prev_y], op: OPAdd})
		} else { //delete
			edits = append(edits, Edit{oldLine: prev_x, newLine: -1, content: a[prev_x], op: OPDelete})
		}
		x = prev_x
		y = prev_y
	}
	return edits
}

func splitLines(s string) []string {
	lines := strings.Split(s, "\n")
	l := len(lines) - 1
	if lines[l] == "" {
		lines = lines[:l]
	}
	return lines
}
