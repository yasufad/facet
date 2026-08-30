// Ported from Taffy tests/xml.rs (MIT).
//
// The XML fixture harness. Each fixture is an XML file describing an input
// tree, a viewport, and an expected output tree. The harness parses the file,
// builds a TaffyTree, runs layout, and compares the computed layout against
// the expectations.
package layout

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// xmlTest is the root element of a fixture file.
type xmlTest struct {
	XMLName      xml.Name     `xml:"test"`
	Name         string       `xml:"name,attr"`
	UseRounding  string       `xml:"use-rounding,attr"`
	Viewport     xmlViewport  `xml:"viewport"`
	Input        xmlContainer `xml:"input"`
	Expectations xmlContainer `xml:"expectations"`
}

// xmlViewport is the viewport element.
type xmlViewport struct {
	Width  string `xml:"width,attr"`
	Height string `xml:"height,attr"`
}

// xmlContainer wraps either an input div or an expectations node. For input,
// the children are divs; for expectations, the children are nodes.
type xmlContainer struct {
	Children []xmlNode `xml:",any"`
}

// xmlNode is either an input div or an expectation node. The harness
// distinguishes them by tag name: "div" for input, "node" for expectations.
type xmlNode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Children []xmlNode  `xml:",any"`
	Text     string     `xml:",chardata"`
}

// attrMap returns the node's attributes as a map.
func (n *xmlNode) attrMap() map[string]string {
	m := make(map[string]string, len(n.Attrs))
	for _, a := range n.Attrs {
		m[a.Name.Local] = a.Value
	}
	return m
}

// outputNode is the computed or expected layout for a single node.
type outputNode struct {
	nodeID   NodeID
	location Point[float32]
	size     Size[float32]
	children []outputNode
}

func (n outputNode) equal(other outputNode) bool {
	return n.nodeID == other.nodeID &&
		abs(n.location.X-other.location.X) < 0.1 &&
		abs(n.location.Y-other.location.Y) < 0.1 &&
		abs(n.size.Width-other.size.Width) < 0.1 &&
		abs(n.size.Height-other.size.Height) < 0.1 &&
		childrenEqual(n.children, other.children)
}

func childrenEqual(a, b []outputNode) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].equal(b[i]) {
			return false
		}
	}
	return true
}

func (n outputNode) String() string {
	var sb strings.Builder
	sb.WriteString("TREE\n")
	n.writeTree(&sb, false, "")
	return sb.String()
}

func (n outputNode) writeTree(sb *strings.Builder, hasSibling bool, prefix string) {
	fork := "└── "
	if hasSibling {
		fork = "├── "
	}
	fmt.Fprintf(sb, "%s%s %d [x: %.1f y: %.1f w: %.1f h: %.1f]\n",
		prefix, fork, n.nodeID.raw, n.location.X, n.location.Y, n.size.Width, n.size.Height)
	bar := "    "
	if hasSibling {
		bar = "│   "
	}
	newPrefix := prefix + bar
	for i, c := range n.children {
		c.writeTree(sb, i < len(n.children)-1, newPrefix)
	}
}

// parseAvailableSpace parses "Npx", "min-content", "max-content", or empty.
func parseAvailableSpace(s string) AvailableSpace {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "", "max-content":
		return maxContent
	case "min-content":
		return minContent
	}
	if strings.HasSuffix(s, "px") {
		s = strings.TrimSuffix(s, "px")
	}
	if v, err := parseFloat32(s); err == nil {
		return definiteAvail(v)
	}
	return maxContent
}

// buildStyleFromAttrs builds a Style from XML attributes.
func buildStyleFromAttrs(attrs map[string]string) Style {
	style := newStyle()
	parseOr := func(key string, fn func(string) error) {
		if v, ok := attrs[key]; ok {
			if err := fn(v); err != nil {
				panic(fmt.Sprintf("parse %s=%q: %v", key, v, err))
			}
		}
	}
	// Grid display is not supported; treat grid as flex for fixture compatibility.
	if v, ok := attrs["display"]; ok && strings.Contains(strings.ToLower(v), "grid") {
		style.Display = displayFlex
	} else {
		parseOr("display", func(s string) error { d, err := parseDisplay(s); style.Display = d; return err })
	}
	parseOr("direction", func(s string) error { d, err := parseDirection(s); style.Direction = d; return err })
	parseOr("box-sizing", func(s string) error { b, err := parseBoxSizing(s); style.BoxSizing = b; return err })
	parseOr("overflow-x", func(s string) error { o, err := parseOverflow(s); style.Overflow.X = o; return err })
	parseOr("overflow-y", func(s string) error { o, err := parseOverflow(s); style.Overflow.Y = o; return err })
	parseOr("position", func(s string) error { p, err := parsePosition(s); style.Position = p; return err })
	parseOr("flex-direction", func(s string) error { d, err := parseFlexDirection(s); style.FlexDirection = d; return err })
	parseOr("flex-wrap", func(s string) error { w, err := parseFlexWrap(s); style.FlexWrap = w; return err })
	parseOr("flex-grow", func(s string) error { v, err := parseFloat32(s); style.FlexGrow = v; return err })
	parseOr("flex-shrink", func(s string) error { v, err := parseFloat32(s); style.FlexShrink = v; return err })
	parseOr("flex-basis", func(s string) error { d, err := parseDimension(s); style.FlexBasis = d; return err })
	parseOr("scrollbar-width", func(s string) error { v, err := parseFloat32(s); style.ScrollbarWidth = v; return err })

	parseDim := func(key string, target *Dimension) {
		parseOr(key, func(s string) error { d, err := parseDimension(s); *target = d; return err })
	}
	parseDim("width", &style.Size.Width)
	parseDim("height", &style.Size.Height)

	parseLPA := func(key string, target *LengthPercentageAuto) {
		parseOr(key, func(s string) error { l, err := parseLengthPercentageAuto(s); *target = l; return err })
	}
	parseLPA("min-width", &style.MinSize.Width)
	parseLPA("min-height", &style.MinSize.Height)
	parseLPA("max-width", &style.MaxSize.Width)
	parseLPA("max-height", &style.MaxSize.Height)
	parseLPA("top", &style.Inset.Top)
	parseLPA("left", &style.Inset.Left)
	parseLPA("bottom", &style.Inset.Bottom)
	parseLPA("right", &style.Inset.Right)
	parseLPA("margin-top", &style.Margin.Top)
	parseLPA("margin-left", &style.Margin.Left)
	parseLPA("margin-bottom", &style.Margin.Bottom)
	parseLPA("margin-right", &style.Margin.Right)

	parseLP := func(key string, target *LengthPercentage) {
		parseOr(key, func(s string) error { l, err := parseLengthPercentage(s); *target = l; return err })
	}
	parseLP("padding-top", &style.Padding.Top)
	parseLP("padding-left", &style.Padding.Left)
	parseLP("padding-bottom", &style.Padding.Bottom)
	parseLP("padding-right", &style.Padding.Right)
	parseLP("border-top", &style.Border.Top)
	parseLP("border-left", &style.Border.Left)
	parseLP("border-bottom", &style.Border.Bottom)
	parseLP("border-right", &style.Border.Right)
	parseLP("column-gap", &style.Gap.Width)
	parseLP("row-gap", &style.Gap.Height)

	if v, ok := attrs["aspect-ratio"]; ok {
		if r, err := parseFloat32(v); err == nil {
			style.AspectRatio = &r
		}
	}
	if v, ok := attrs["align-items"]; ok {
		if ai, err := parseAlignItems(v); err == nil {
			style.AlignItems = &ai
		}
	}
	if v, ok := attrs["align-self"]; ok {
		if as, err := parseAlignItems(v); err == nil {
			style.AlignSelf = &as
		}
	}
	if v, ok := attrs["align-content"]; ok {
		if ac, err := parseAlignContent(v); err == nil {
			style.AlignContent = &ac
		}
	}
	if v, ok := attrs["justify-content"]; ok {
		if jc, err := parseAlignContent(v); err == nil {
			style.JustifyContent = &jc
		}
	}
	return style
}

// constructTree builds a TaffyTree from the input XML and returns the
// expected output tree.
func constructTree(input *xmlNode, expected *xmlNode, tree *TaffyTree, parent *NodeID, contexts map[NodeID]*testNodeContext) outputNode {
	attrs := input.attrMap()
	style := buildStyleFromAttrs(attrs)

	var nodeID NodeID
	hasChildren := len(input.Children) > 0
	if hasChildren {
		nodeID = tree.NewWithChildren(style, nil)
	} else {
		nodeID = tree.NewLeaf(style)
		textContent := strings.TrimSpace(input.Text)
		wm := writingHorizontal
		if v, ok := attrs["writing-mode"]; ok && strings.Contains(strings.ToLower(v), "vertical") {
			wm = writingVertical
		}
		if textContent != "" {
			ctx := newTestContextAhemText(textContent, wm)
			tree.SetNodeContext(nodeID, &ctx)
			if contexts != nil {
				contexts[nodeID] = &ctx
			}
		}
	}

	if parent != nil {
		tree.AddChild(*parent, nodeID)
	}

	expectedOut := buildExpectations(expected, nodeID)

	if hasChildren {
		for i, child := range input.Children {
			if i >= len(expected.Children) {
				break
			}
			expectedOut.children = append(expectedOut.children,
				constructTree(&child, &expected.Children[i], tree, &nodeID, contexts))
		}
	}
	return expectedOut
}

// buildExpectations builds an outputNode from an expectation XML node.
func buildExpectations(node *xmlNode, id NodeID) outputNode {
	attrs := node.attrMap()
	out := outputNode{nodeID: id}
	if v, ok := attrs["x"]; ok {
		out.location.X, _ = parseFloat32(v)
	}
	if v, ok := attrs["y"]; ok {
		out.location.Y, _ = parseFloat32(v)
	}
	if v, ok := attrs["width"]; ok {
		out.size.Width, _ = parseFloat32(v)
	}
	if v, ok := attrs["height"]; ok {
		out.size.Height, _ = parseFloat32(v)
	}
	return out
}

// getComputedExpectations builds an outputNode from the computed layout.
func getComputedExpectations(tree *TaffyTree, id NodeID) outputNode {
	layout := tree.Layout(id)
	out := outputNode{
		nodeID:   id,
		location: layout.Location,
		size:     layout.Size,
	}
	for _, child := range tree.Children(id) {
		out.children = append(out.children, getComputedExpectations(tree, child))
	}
	return out
}

// runXMLTest loads and runs a single XML fixture.
func runXMLTest(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var xt xmlTest
	if err := xml.Unmarshal(raw, &xt); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	avail := Size[AvailableSpace]{
		Width:  parseAvailableSpace(xt.Viewport.Width),
		Height: parseAvailableSpace(xt.Viewport.Height),
	}

	tree := NewTaffyTree()
	if strings.ToLower(xt.UseRounding) == "false" {
		tree.DisableRounding()
	}
	contexts := make(map[NodeID]*testNodeContext)
	if len(xt.Input.Children) == 0 || len(xt.Expectations.Children) == 0 {
		t.Fatalf("empty input or expectations in %s", path)
	}
	expected := constructTree(&xt.Input.Children[0], &xt.Expectations.Children[0], tree, nil, contexts)
	rootID := expected.nodeID

	tree.ComputeLayoutWithMeasure(rootID, avail, func(in LayoutInput, id NodeID, ctx any, style *Style) LayoutOutput {
		return testMeasureFunction(in, id, ctx, style)
	})

	actual := getComputedExpectations(tree, rootID)
	if !expected.equal(actual) {
		t.Errorf("layout mismatch in %s\nEXPECTED:\n%s\nACTUAL:\n%s",
			filepath.Base(path), expected, actual)
	}
}

// runXMLTestDir runs all XML fixtures in a directory.
func runXMLTestDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".xml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".xml")
		t.Run(name, func(t *testing.T) {
			runXMLTest(t, filepath.Join(dir, e.Name()))
		})
	}
}
