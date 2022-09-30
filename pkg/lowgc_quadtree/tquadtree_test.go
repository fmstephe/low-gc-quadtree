package lowgc_quadtree

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

const dups = 10

// testRand will produce the same random numbers every time
// This is done to make the benchmarks consistent between runs
var testRand = rand.New(rand.NewSource(1))

func buildTestTrees() []Tree[string] {
	return []Tree[string]{
		NewQuadTree[string](NewView(0, 10, 10, 0)),
		NewQuadTree[string](NewView(0, 1, 2, 0)),
		NewQuadTree[string](NewView(0, 100, 300, 0)),
		NewQuadTree[string](NewView(0, 20.4, 35.6, 0)),
		NewQuadTree[string](NewView(0, 1e10, 500.00000001, 0)),
		// Negative regions
		NewQuadTree[string](NewView(-10, 10, 10, -10)),
		NewQuadTree[string](NewView(-1, 1, 2, -2)),
		NewQuadTree[string](NewView(-100, 100, 300, -300)),
		NewQuadTree[string](NewView(-20.4, 20.4, 35.6, -35.6)),
		NewQuadTree[string](NewView(-1e10, 1e10, 500.00000001, -500.00000001)),
	}
}

func TestOverflowLeaf(t *testing.T) {
	tree := NewQuadTree[string](NewView(0, 1, 1, 0))
	ps := fillView(tree.View(), 70)
	for i, p := range ps {
		err := tree.Insert(p.x, p.y, fmt.Sprintf("test-%d", i))
		assert.NoError(t, err)
	}
	fun, results := SimpleSurvey[string]()
	tree.Survey(tree.View(), fun)
	if 70 != results.Len() {
		t.Errorf("Failed to retrieve 70 elements in scatter test, found only %d", results.Len())
	}
}

// Test that we can insert a single element into the tree and then retrieve it
func TestOneElement(t *testing.T) {
	testTrees := buildTestTrees()
	for _, tree := range testTrees {
		testOneElement(tree, t)
	}
}

func testOneElement(tree Tree[string], t *testing.T) {
	x, y := randomPosition(tree.View())
	err := tree.Insert(x, y, "test")
	assert.NoError(t, err)
	fun, results := SimpleSurvey[string]()
	tree.Survey(tree.View(), fun)
	if results.Len() != 1 || "test" != results.Front().Value {
		t.Errorf("Failed to find required element at (%f,%f), in tree \n%v", x, y, tree)
	}
}

// Here we fill up each quadrant of the root leaves of the tree. We exploit the
// implementation detail that each quadrant can hold LEAF_SIZE many elements
// before it overflows.  So we take care to insert more than LEAF_SIZE many
// elements into each quadrant.
func TestFullLeaf(t *testing.T) {
	testTrees := buildTestTrees()
	for _, tree := range testTrees {
		views := tree.View().quarters()
		for _, view := range views {
			testFullLeaf(tree, view, "v1", t)
		}
	}
}

func testFullLeaf(tree Tree[string], v View, msg string, t *testing.T) {
	inserts := LEAF_SIZE * 2
	for i := 0; i < inserts; i++ {
		x, y := randomPosition(v)
		name := "test" + strconv.Itoa(i)
		err := tree.Insert(x, y, name)
		assert.NoError(t, err)
	}
	fun, results := SimpleSurvey[string]()
	tree.Survey(v, fun)
	if results.Len() != inserts {
		t.Error(msg, "Inserted %d elements into a fresh quadtree and retrieved only %s", inserts, results.Len())
	}
}

// Show that any insert of a point which is not contained in the view of a tree
// returns and error
func TestBadInsert(t *testing.T) {
	v1, v2 := disjoint()
	tree := NewQuadTree[string](v1)
	ps := fillView(v2, 100)
	for _, p := range ps {
		err := tree.Insert(p.x, p.y, "test")
		assert.Error(t, err)
	}
}

// Tests that we can add a large number of random elements to a tree
// and create random views for collecting from the populated tree.
func TestScatter(t *testing.T) {
	testTrees := buildTestTrees()
	for _, tree := range testTrees {
		testScatter(tree, t)
	}
	/*
		testTrees = buildTestTrees()
		for _, tree := range testTrees {
			testScatterDup(tree, t)
		}
	*/
}

func testScatter(tree Tree[string], t *testing.T) {
	ps := fillView(tree.View(), 10000)
	for i, p := range ps {
		err := tree.Insert(p.x, p.y, "test")
		assert.NoError(t, err)
		count := tree.Count(tree.View())
		if int64(i+1) != count {
			t.Errorf("Failed to count %d elements in scatter test, found %d", i, count)
		}
	}

	for i := 0; i < 1; i++ {
		sv := subView(tree.View())
		var pointCount int64
		for _, v := range ps {
			if sv.containsPoint(v.x, v.y) {
				pointCount++
			}
		}

		fun, results := SimpleSurvey[string]()
		tree.Survey(sv, fun)
		if pointCount != int64(results.Len()) {
			t.Errorf("Failed to retrieve %d elements in scatter test, found %d", pointCount, results.Len())
		}

		count := tree.Count(sv)
		if pointCount != count {
			t.Errorf("Failed to count %d elements in scatter test, found %d", pointCount, count)
		}
	}
}

// Tests that we can add multiple elements to the same location
// and still retrieve all elements, including duplicates, using
// randomly generated views.
func testScatterDup(tree Tree[string], t *testing.T) {
	return
	ps := fillView(tree.View(), 1000)
	for _, p := range ps {
		for i := 0; i < dups; i++ {
			err := tree.Insert(p.x, p.y, "test_"+strconv.Itoa(i))
			assert.NoError(t, err)
		}
	}
	for i := 0; i < 1000; i++ {
		sv := subView(tree.View())
		var pointCount int64
		for _, v := range ps {
			if sv.containsPoint(v.x, v.y) {
				pointCount++
			}
		}

		fun, results := SimpleSurvey[string]()
		tree.Survey(sv, fun)
		if pointCount*dups != int64(results.Len()) {
			t.Error("Failed to retrieve %i elements in duplicate scatter test, found only %i", pointCount*dups, results.Len())
		}

		count := tree.Count(sv)
		if pointCount != 1 { //count {
			t.Errorf("Failed to count %d elements in duplicate scatter test, found only %d", pointCount, count)
		}
	}
}

// Demonstrate that we can terminate a Survey by having the survey func return false
// Here we use this to limit the number of elements returned from a survey
func TestLimitedSurvey(t *testing.T) {
	testTrees := buildTestTrees()
	for _, tree := range testTrees {
		ps := fillView(tree.View(), 1000)
		for _, p := range ps {
			err := tree.Insert(p.x, p.y, "test")
			assert.NoError(t, err)
		}
		for i := 0; i < 1000; i++ {
			fun, results := LimitSurvey[string](i)
			tree.Survey(tree.View(), fun)
			if results.Len() != i {
				t.Errorf("Failed to retrieve %d elements, found %d instead", i, results.Len())
			}
		}
	}
}

func randomPosition(v View) (x, y float64) {
	x = testRand.Float64()*(v.rx-v.lx) + v.lx
	y = testRand.Float64()*(v.by-v.ty) + v.ty
	return
}

func fillView(v View, c int) []point {
	ps := make([]point, c)
	for i := 0; i < c; i++ {
		x, y := randomPosition(v)
		ps[i] = point{x: x, y: y}
	}
	return ps
}

func subView(v View) View {
	lx := testRand.Float64()*(v.rx-v.lx) + v.lx
	rx := testRand.Float64()*(v.rx-lx) + lx
	by := testRand.Float64()*(v.ty-v.by) + v.by
	ty := testRand.Float64()*(v.ty-by) + by
	return NewView(lx, rx, ty, by)
}
