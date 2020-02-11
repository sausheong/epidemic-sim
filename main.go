package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"image"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/nsf/termbox-go"
)

// image shown on the screen
var img *image.RGBA

// the simulation grid
var cells []Cell

var infections []int
var deaths []int

// number of 'live' cells in the simulation
var living int

// number of cells that 'died'
var dead int

// number of cells that recovered
var recovered int

// number of cells that got infected
var infected int

var neverInfected int

// the number of cells on one side of the image
var width *int

// number of simulation days
var numDays *int

// duratrion
var duration *int

// probability that infection happens
var rate *float64

// percentage of simulation grid that is populated
var coverage *float64

// deadliness of the disease, lower is less deadly
var fatality *float64

// likelihood the cell gets infected again after recovery
var immunity *float64

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	// simulation configuration
	numDays = flag.Int("t", 500, "number of simulation days")
	width = flag.Int("w", 60, "the number of cells on one side of the image")

	// capture the simulation parameters
	duration = flag.Int("d", 4, "how long the individual remains infectious")
	rate = flag.Float64("r", 0.3, "how likely infection happens")
	fatality = flag.Float64("f", 0.3, "probability of fatality")
	immunity = flag.Float64("i", 0.5, "likelihood the cell gets infected again after recovery")
	coverage = flag.Float64("c", 0.3, "percentage of simulation grid that is populated")

	flag.Parse()

	// using termbox to control the simulation
	termbox.Init()
	endSim := false
	dead = 0

	// poll for keyboard events in another goroutine
	events := make(chan termbox.Event, 1000)
	go func() {
		for {
			events <- termbox.PollEvent()
		}
	}()

	// create the initial population
	createPopulation()
	infectOneCell()

	var t int
	// main simulation loop
	for t = 0; !endSim && (t < *numDays); t++ {
		// capture the ctrl-q key to end the simulation
		select {
		case ev := <-events:
			if ev.Type == termbox.EventKey {
				if ev.Key == termbox.KeyCtrlQ {
					endSim = true
				}
			}
		default:
		}

		for n, cell := range cells {
			// if cell is empty or not infected, go to the next cell
			if cell.getRGB() == 0 || !cell.Infected {
				continue
			}
			// process infected cells
			cells[n].process()

			// find all the cell's neighbours
			neighbours := findNeighboursIndex(n)

			// for every neighbour
			for _, neighbour := range neighbours {

				// if the cell is empty or not infected, go to the next neighbour
				if cells[neighbour].getRGB() == 0 || cells[neighbour].Infected {
					continue
				}

				// check probability of re-infection
				if rand.Float64() > cells[neighbour].Immunity {
					// if probability less than infection probability then cell gets infected
					if rand.Float64() < *rate {
						cells[neighbour].infected()
					}
				}
			}
		}

		img = draw(*width*CELLSIZE+CELLSIZE, *width*CELLSIZE+CELLSIZE, cells)
		printImage(img.SubImage(img.Rect))

		neverInfected = countNeverInfected()
		count := countInfected()
		if count == 0 {
			endSim = true
		}
		fmt.Printf("\nCurrent infected: %d\n", count)
		output(t)
		fmt.Println()
		fmt.Println("\nCtrl-Q to quit simulation.")

		// collect simulation data
		infections = append(infections, count)
		deaths = append(deaths, dead)

	}
	termbox.Close()
	simName := fmt.Sprintf("d%d-r%1.1f-i%1.1f-f%1.1f-w%d-c%1.1f", *duration, *rate, *fatality, *immunity, *width, *coverage)
	saveData(simName)
	fmt.Println("\nDATA")
	output(t)
	fmt.Println()
}

func output(t int) {
	fmt.Printf("Simulation time: %d/%d days", t, *numDays)
	fmt.Printf("\nInfected: %d out of %d (%2.1f%%)", living-neverInfected, living, float64(living-neverInfected)*100.0/float64(living))
	fmt.Printf("\nDied: %d out of %d (%2.1f%%)", dead, living, float64(dead)*100.0/float64(living))
	fmt.Printf("\nRecovered: %d out of %d infected (%2.1f%%)", recovered, infected, float64(recovered)*100.0/float64(infected))

	fmt.Println("\n\nPARAMETERS")
	fmt.Printf("Population density: %2.0f%%", (*coverage)*100)
	fmt.Printf("\nHow long the individual remains infectious: %d", *duration)
	fmt.Printf("\nProbability infection happens: %2.1f%%", *rate*100)
	fmt.Printf("\nProbability of fatality: %2.1f%%", *fatality*100)
	fmt.Printf("\nProbability the cell gets infected again after recovery: %2.1f%%", *immunity*100)
}

// save simulation data
func saveData(name string) {
	// snapshot of grid at the end of the simulation
	cellsfile, err := os.Create(fmt.Sprintf("data/epidemic-%s.csv", name))
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	csvwriter := csv.NewWriter(cellsfile)
	for i := 0; i < len(infections); i++ {
		_ = csvwriter.Write([]string{strconv.Itoa(infections[i]), strconv.Itoa(deaths[i])})
	}
	csvwriter.Flush()
	cellsfile.Close()

	// save the last image of the grid
	saveImage("data/"+name+".png", img)
}
