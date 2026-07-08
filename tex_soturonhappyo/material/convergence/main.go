package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"image/color"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

const (
	OutputDir = "output"
)

// ---------------------------------------------------------
// 図1: 区間と3点の配置
// ---------------------------------------------------------
func generateDiagram1() error {
	p := plot.New()
	p.Title.Text = "Golden Section Method: 3-Point Arrangement"
	p.X.Label.Text = "Position"
	p.Y.Label.Text = ""

	// 軸を隠す
	p.Y.Max = 2
	p.Y.Min = 0
	p.X.Min = 0
	p.X.Max = 10

	// 線を描画（区間を表現）
	line, _ := plotter.NewLine(plotter.XYs{
		{X: 0, Y: 1},
		{X: 10, Y: 1},
	})
	line.LineStyle.Width = vg.Centimeter * 0.1
	line.Color = color.RGBA{0, 0, 0, 255}
	p.Add(line)

	// 3点をプロット
	points := plotter.XYs{
		{X: 0, Y: 1},    // a
		{X: 3.82, Y: 1}, // b (W ≈ 0.382)
		{X: 10, Y: 1},   // c
	}
	scatter, _ := plotter.NewScatter(points)
	scatter.GlyphStyle.Color = color.RGBA{255, 0, 0, 255}
	scatter.GlyphStyle.Radius = vg.Points(8)
	p.Add(scatter)

	// テキストラベル
	p.Legend.Add("a (左)", scatter)

	err := p.Save(15*vg.Centimeter, 6*vg.Centimeter, filepath.Join(OutputDir, "diagram1_points.png"))
	return err
}

// ---------------------------------------------------------
// 図2: 比率の説明
// ---------------------------------------------------------
func generateDiagram2() error {
	p := plot.New()
	p.Title.Text = "Step 2: Optimization of W and 1-W"
	p.X.Label.Text = "Ratio"
	p.Y.Label.Text = ""

	p.Y.Max = 2
	p.Y.Min = 0
	p.X.Min = 0
	p.X.Max = 1

	// W の情報
	W := (3.0 - math.Sqrt(5)) / 2.0

	// グラフに重要な情報を表示するための簡単な線を描画
	line1, _ := plotter.NewLine(plotter.XYs{
		{X: W, Y: 1},
		{X: 1, Y: 1},
	})
	line1.LineStyle.Width = vg.Centimeter * 0.1
	line1.Color = color.RGBA{0, 0, 200, 255}
	p.Add(line1)

	line2, _ := plotter.NewLine(plotter.XYs{
		{X: 0, Y: 0.5},
		{X: W, Y: 0.5},
	})
	line2.LineStyle.Width = vg.Centimeter * 0.1
	line2.Color = color.RGBA{200, 0, 0, 255}
	p.Add(line2)

	err := p.Save(15*vg.Centimeter, 6*vg.Centimeter, filepath.Join(OutputDir, "diagram2_optimization.png"))
	return err
}

// ---------------------------------------------------------
// 図3: スケール相似性
// ---------------------------------------------------------
func generateDiagram3() error {
	p := plot.New()
	p.Title.Text = "Step 3: Scale Similarity"
	p.X.Label.Text = "Scale"
	p.Y.Label.Text = ""

	p.Y.Max = 3
	p.Y.Min = 0
	p.X.Min = 0
	p.X.Max = 1

	// 2つの相似な線を描画
	line1, _ := plotter.NewLine(plotter.XYs{
		{X: 0, Y: 2},
		{X: 1, Y: 2},
	})
	line1.LineStyle.Width = vg.Centimeter * 0.1
	line1.Color = color.RGBA{100, 100, 100, 255}
	p.Add(line1)

	line2, _ := plotter.NewLine(plotter.XYs{
		{X: 0, Y: 1},
		{X: 0.618, Y: 1},
	})
	line2.LineStyle.Width = vg.Centimeter * 0.1
	line2.Color = color.RGBA{100, 100, 100, 255}
	p.Add(line2)

	err := p.Save(15*vg.Centimeter, 6*vg.Centimeter, filepath.Join(OutputDir, "diagram3_similarity.png"))
	return err
}

// ---------------------------------------------------------
// 図4: 黄金比と収束
// ---------------------------------------------------------
func generateDiagram4() error {
	p := plot.New()
	p.Title.Text = "Step 4: Golden Ratio and Convergence"
	p.X.Label.Text = "Iterations"
	p.Y.Label.Text = "Interval Width (Normalized)"

	p.Y.Max = 1.1
	p.Y.Min = 0
	p.X.Min = 0
	p.X.Max = 10

	// 収束曲線を描画
	phi := (math.Sqrt(5) - 1.0) / 2.0
	convergenceData := plotter.XYs{}
	for i := 0; i < 10; i++ {
		convergenceData = append(convergenceData, plotter.XY{
			X: float64(i),
			Y: math.Pow(phi, float64(i)),
		})
	}

	line, _ := plotter.NewLine(convergenceData)
	line.LineStyle.Width = vg.Points(2)
	line.Color = color.RGBA{0, 100, 200, 255}
	p.Add(line)

	scatter, _ := plotter.NewScatter(convergenceData)
	scatter.GlyphStyle.Color = color.RGBA{200, 0, 0, 255}
	scatter.GlyphStyle.Radius = vg.Points(4)
	p.Add(scatter)

	err := p.Save(15*vg.Centimeter, 8*vg.Centimeter, filepath.Join(OutputDir, "diagram4_convergence.png"))
	return err
}

// ---------------------------------------------------------
// メイン
// ---------------------------------------------------------
func main() {
	os.MkdirAll(OutputDir, 0755)

	fmt.Println("Generating diagrams for golden ratio derivation...")

	if err := generateDiagram1(); err != nil {
		fmt.Printf("Warning: Failed to generate Diagram 1: %v\n", err)
	} else {
		fmt.Println("✓ Successfully generated Diagram 1 (3 Points and Ratios)")
	}

	if err := generateDiagram2(); err != nil {
		fmt.Printf("Warning: Failed to generate Diagram 2: %v\n", err)
	} else {
		fmt.Println("✓ Successfully generated Diagram 2 (Optimization Condition)")
	}

	if err := generateDiagram3(); err != nil {
		fmt.Printf("Warning: Failed to generate Diagram 3: %v\n", err)
	} else {
		fmt.Println("✓ Successfully generated Diagram 3 (Similarity)")
	}

	if err := generateDiagram4(); err != nil {
		fmt.Printf("Warning: Failed to generate Diagram 4: %v\n", err)
	} else {
		fmt.Println("✓ Successfully generated Diagram 4 (Convergence)")
	}

	fmt.Printf("✓ Successfully generated all diagrams in %s\n", OutputDir)
}
