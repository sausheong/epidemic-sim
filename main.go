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

// simulation data the simulation
var living, dead, recovered, infected, neverInfected int
var infections, deaths, infecteds []int

// simulation configurations
var width *int       // the number of cells on one side of the image
var numDays *int     // number of simulation days
var filename *string // name of data file

// epidemic simulation parameters
var rate *float64     // probability that infection happens
var incubation *int   // how long the disease stays dormant before being infectious
var duration *int     // how long the disease lasts before fatality or recovery
var density *float64  // percentage of simulation grid that is populated
var fatality *float64 // deadliness of the disease, lower is less deadly
var immunity *float64 // how immune the cell is to infection after recovery

// preventive measures
var medIntroduced *int        // day when medicine is introduced
var medEffectiveness *float64 // effectiveness of medicine
var qIntroduced *int          // day when quarantine is introduced
var qEffectiveness *float64   // effectiveness of quarantine

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	// capturing user input
	numDays = flag.Int("t", 300, "number of simulation days")
	width = flag.Int("w", 60, "the number of cells on one side of the image")
	rate = flag.Float64("r", 0.15, "how likely infection happens")
	incubation = flag.Int("n", 3, "how long before the cell becomes infectious")
	duration = flag.Int("d", 4, "how long the cell remains infectious")
	fatality = flag.Float64("f", 0.02, "probability of fatality")
	immunity = flag.Float64("i", 0.5, "how immune the cell is to infection after recovery")
	density = flag.Float64("c", 0.7, "percentage of simulation grid that is populated")
	medIntroduced = flag.Int("m", *numDays, "day when medicine is introduced")
	medEffectiveness = flag.Float64("e", 0.0, "effectiveness of medicine")
	qIntroduced = flag.Int("q", *numDays, "day when quarantine is introduced")
	qEffectiveness = flag.Float64("g", 0.0, "effectiveness of quarantine")
	filename = flag.String("name", "data", "file name of data file")
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

		for n := range cells {
			// if cell is empty or not infected, go to the next cell
			if cells[n].getRGB() == 0 || !cells[n].Infected {
				continue
			}

			// process infected cells
			cells[n].process()

			// if cell has recovered or still incubating, go to next cell
			if !cells[n].Infected || cells[n].Incubation > 0 {
				continue
			}

			// apply medicine if it's time
			if !cells[n].Medicated && (t > *medIntroduced) {
				cells[n].medicate()
			}

			// if medicine is successful, cell is recovered so go to next cell
			if !cells[n].Infected {
				continue
			}

			// quarantine has been started
			if !cells[n].Quarantined && (t > *qIntroduced) && (rand.Float64() < *qEffectiveness) {
				cells[n].quarantine()
			}

			// unless quarantined, look for neighbours and infect them （｀ー´）
			if !cells[n].Quarantined {
				// find all the cell's neighbours
				neighbours := findNeighboursIndex(n)

				// for every neighbour
				for _, neighbour := range neighbours {
					// if the cell is empty or already infected, go to the next neighbour
					if cells[neighbour].getRGB() == 0 || cells[neighbour].Infected {
						continue
					}

					// check probability of re-infection
					if rand.Float64() > cells[neighbour].Immunity {
						// if probability less than infection rate then cell gets infected
						if rand.Float64() < *rate {
							cells[neighbour].infected()
						}
					}
				}
			}
		}

		img = draw(*width*CELLSIZE+CELLSIZE, *width*CELLSIZE+CELLSIZE, cells)
		printImage(img.SubImage(img.Rect))

		neverInfected = countNeverInfected()
		count := countInfected()

		fmt.Printf("\nCurrent infected: %d cells\n", count)
		output(t)
		fmt.Println("\n\nCtrl-Q to quit simulation.")

		// collect simulation data
		infections = append(infections, count)
		deaths = append(deaths, dead)
		infecteds = append(infecteds, living-neverInfected)
	}
	termbox.Close()
	saveData(*filename)
	fmt.Println("\nDATA")
	output(t)
	fmt.Println()
}

func output(t int) {
	fmt.Printf("Time      : %d/%d days", t, *numDays)
	fmt.Printf("\nInfected  : %d out of %d (%2.1f%%)", living-neverInfected, living,
		float64(living-neverInfected)*100.0/float64(living))
	fmt.Printf("\nDied      : %d out of %d (%2.1f%%)", dead, living,
		float64(dead)*100.0/float64(living))
	fmt.Printf("\nRecovered : %d out of %d infected (%2.1f%%)", recovered, infected,
		float64(recovered)*100.0/float64(infected))

	fmt.Println("\n\nPARAMETERS")
	fmt.Printf("Density      : %2.0f%% populated", (*density)*100)
	fmt.Printf("\nInfection    : %2.1f%% ", *rate*100)
	fmt.Printf("\nRe-infection : %2.1f%% ", (1-*immunity)*100)
	fmt.Printf("\nIncubation   : %d days", *incubation)
	fmt.Printf("\nInfectious   : %d days", *duration)
	fmt.Printf("\nFatality     : %2.1f%% fatal", *fatality*100)
	if *qIntroduced < *numDays {
		fmt.Println("\n\nQUARANTINE")
		fmt.Printf("Quarantine introduced    : %dth day", *qIntroduced)
		fmt.Printf("\nQuarantine effectiveness : %2.1f%% found and quarantined", *qEffectiveness*100)
	}
	if *medIntroduced < *numDays {
		fmt.Println("\n\nMEDICINE")
		fmt.Printf("Med introduced    : %dth day", *medIntroduced)
		fmt.Printf("\nMed effectiveness : %2.1f%% recovery\n", *medEffectiveness*100)
	}
}

// save simulation data
func saveData(name string) {
	// snapshot of grid at the end of the simulation
	cellsfile, err := os.Create(fmt.Sprintf("data/epidemic-%s.csv", name))
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	csvwriter := csv.NewWriter(cellsfile)
	_ = csvwriter.Write([]string{"infections", "deaths", "infecteds"})
	for i := 0; i < len(infections); i++ {
		_ = csvwriter.Write([]string{strconv.Itoa(infections[i]), strconv.Itoa(deaths[i]), strconv.Itoa(infecteds[i])})
	}
	csvwriter.Flush()
	cellsfile.Close()
}
