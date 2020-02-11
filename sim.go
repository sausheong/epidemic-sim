package main

import (
	"image/color"
	"math/rand"
)

// CELLSIZE is the radius of each cell
var CELLSIZE = 10

// MASKARRAY is an array of masks used to replace the traits
var MASKARRAY []int = []int{0xFFFFF0, 0xFFFF0F, 0xFFF0FF, 0xFF0FFF, 0xF0FFFF, 0x0FFFFF}

// Cell is a representation of a cell within the grid
type Cell struct {
	X        int
	Y        int
	R        int
	Infected bool    // infected or not?
	Duration int     // how long the infection lasts
	Immunity float64 // immunity from getting it again
	Color    color.Color
}

// get the color integer back from the cell in the form 0x1A2B3C
func (c *Cell) getRGB() int {
	r, g, b, _ := c.Color.RGBA()
	return int((r & 0x00FF << 16) + (g & 0x00FF << 8) + b&0x00FF)
}

// set the color using the color interger in the form 0x1A2B3C
func (c *Cell) setRGB(i int) {
	c.Color = color.RGBA{getR(i), getG(i), getB(i), uint8(255)}
}

// cell becomes infected
func (c *Cell) infected() {
	c.setRGB(0xFF0000)
	c.Infected = true
	c.Duration = *duration
	infected++
}

// cell recovers and gain a level of immunity
func (c *Cell) recover() {
	c.setRGB(0x00FF00)
	c.Infected = false
	c.Duration = 0
	c.Immunity = *immunity
	recovered++
}

// cell dies :(
func (c *Cell) die() {
	c.setRGB(0)
	c.Infected = false
	c.Duration = 0
	dead++
}

// process the infected cell
func (c *Cell) process() {
	if c.Infected {
		if c.Duration > 0 {
			c.Duration = c.Duration - 1
		} else {
			if rand.Float64() > *fatality {
				c.recover()
			} else {
				c.die()
			}
		}
	}
}

// create a cell
func createCell(x, y, clr int) (c Cell) {
	c = Cell{
		X:        x,
		Y:        y,
		R:        CELLSIZE, // radius of cell
		Color:    color.RGBA{getR(clr), getG(clr), getB(clr), uint8(255)},
		Infected: false,
		Duration: 0,
		Immunity: 0.0,
	}
	return
}

// create the initial population
func createPopulation() {
	cells = make([]Cell, *width*(*width))
	n := 0
	for i := 1; i <= *width; i++ {
		for j := 1; j <= *width; j++ {
			p := rand.Float64()
			if p < *coverage {
				cells[n] = createCell(i*CELLSIZE, j*CELLSIZE, 0x00FF00)
				living++
			} else {
				cells[n] = createCell(i*CELLSIZE, j*CELLSIZE, 0)
			}
			n++
		}
	}
}

// choose 1 cell to be patient zero
func infectOneCell() {
	i := rand.Intn(*width * (*width))
	cells[i].setRGB(0xFF0000)
	cells[i].Infected = true
	cells[i].Duration = *duration
}

func countNeverInfected() int {
	count := 0
	for _, cell := range cells {
		if (cell.getRGB() == 0x00FF00) && cell.Immunity == 0.0 {
			count++
		}
	}
	return count
}

// count the number of infected cells
func countInfected() int {
	count := 0
	for _, cell := range cells {
		if cell.Infected {
			count++
		}
	}
	return count
}

// the color integer is 0x1A2B3CFF where
// 1A is the red, 2B is green and 3C is blue

// get the red (R) from the color integer i
func getR(i int) uint8 {
	return uint8((i >> 16) & 0x0000FF)
}

// get the green (G) from the color integer i
func getG(i int) uint8 {
	return uint8((i >> 8) & 0x0000FF)
}

// get the blue (B) from the color integer i
func getB(i int) uint8 {
	return uint8(i & 0x0000FF)
}

// find the index of a random empty cell in the grid
func findRandomEmpty(empty []int) int {
	r := rand.Intn(len(empty))
	return empty[r]
}

// find all cells that are empty in the grid
func findEmpty() (empty []int) {
	for n, cell := range cells {
		if cell.getRGB() == 0 {
			empty = append(empty, n)
		}
	}
	return
}
