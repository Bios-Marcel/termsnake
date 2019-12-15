package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gdamore/tcell"
)

var (
	greenStyle = tcell.StyleDefault.Foreground(tcell.ColorGreen)
	apple = []rune("🍎")
)

const fullBlock = '█'

type loc struct {
	x int
	y int
}

type gameState struct {
	snake []*loc
	apple *loc
	score int
	
	lock        *sync.Mutex
	heading     direction
	lastHeading direction
	width       int
	height      int
}

func main() {
	screen, err := tcell.NewScreen()
	if err != nil {
		panic(err)
	}

	defer func() {
		err := recover()
		if err != nil {
			panic(err)
			screen.Fini() 
		}
	}()

	initError := screen.Init()
	if initError != nil {
		panic(initError)
	}

	screen.Clear()

	width, height := screen.Size()
	height = height - 1
	//-1 since we use the bottom for drawing a status bar
	halfWidth := width / 2
	state := &gameState{
		lock:    &sync.Mutex{},
		width:   width,
		height:  height,
		//Let snake start at the bottom and make sure it's not on an odd x coordinate.
		snake:   []*loc{{halfWidth - (halfWidth % 2), height - 1}},
		heading: up,
	}

	screen.SetCell(0, state.height, tcell.StyleDefault, 'S')
	screen.SetCell(1, state.height, tcell.StyleDefault, 'c')
	screen.SetCell(2, state.height, tcell.StyleDefault, 'o')
	screen.SetCell(3, state.height, tcell.StyleDefault, 'r')
	screen.SetCell(4, state.height, tcell.StyleDefault, 'e')
	screen.SetCell(5, state.height, tcell.StyleDefault, ':')
	state.draw(screen)

	//User Eventloop
	go func() {
		for {
			event := screen.PollEvent()
			switch event.(type) {
			case *tcell.EventKey:
				keyEvent := event.(*tcell.EventKey)
				switch keyEvent.Key() {
				case tcell.KeyUp:
					state.changeDirection(up)
				case tcell.KeyRight:
					state.changeDirection(right)
				case tcell.KeyDown:
					state.changeDirection(down)
				case tcell.KeyLeft:
					state.changeDirection(left)
				case tcell.KeyCtrlC:
					screen.Fini()
					os.Exit(0)
				}

				/*case *tcell.EventResize:
				panic("Sorry, resize isn't supported.")*/
			}
		}
	}()

	//Game Updateloop
	gameTicker := time.NewTicker((1000 / 22) * time.Millisecond)
	for {
		<-gameTicker.C
		state.updateSnake(screen)
	}
}

type direction int

const (
    none  direction = 0  
	up              = 1
	right           = 2
	down            = 3
	left            = 4
)

func (state *gameState) changeDirection(newDirection direction) {
	state.lock.Lock()
	defer state.lock.Unlock()

	//Directions can only be changed one inbetween one screen-update
	if state.heading == none {
		if len(state.snake) == 1 || state.lastHeading == up && newDirection != down ||
			state.lastHeading == left && newDirection != right || 
			state.lastHeading == right && newDirection != left ||
			state.lastHeading == down && newDirection != up {
			state.heading = newDirection
		}	
	}
}

func (state *gameState) gameOver(screen tcell.Screen) {
	screen.Fini()
	fmt.Println("You died")
	os.Exit(0)
}

func (state *gameState) updateSnake(screen tcell.Screen) {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.clearScreen(screen)

	tail := state.snake[0]

	var oldHead *loc
	if len(state.snake) == 0 {
		oldHead = tail
	} else {
		oldHead = state.snake[len(state.snake)-1]
	}

	//if the head is out of screen, we are dead
	oldHeadChar, _, _, _ := screen.GetContent(oldHead.x, oldHead.y)
	if oldHeadChar == 0 {
		state.gameOver(screen)
	}

	var heading direction
	if state.heading == none {
		heading = state.lastHeading
	} else {
		heading = state.heading		
	}

	newHead := &loc{oldHead.x, oldHead.y}
	
	switch heading {
	case up:
		newHead.y = oldHead.y - 1
	case right:
		newHead.x = oldHead.x + 2
	case down:
		newHead.y = oldHead.y + 1
	case left:
		newHead.x = oldHead.x - 2
	}

	for _, bodyPart := range state.snake {
		if bodyPart.x == newHead.x && bodyPart.y == newHead.y {
			state.gameOver(screen)
		}
	}

	grow := false
	if state.apple == nil {
		state.addApple()
	} else {
		if state.apple != nil && newHead.x == state.apple.x && newHead.y == state.apple.y {
			//TODO Check whether snake fills out the field
			grow = true
			state.score = state.score + 1

			state.addApple()
		}
	}

	//Since we don't want to grow, we can remove the tail. If we want to
	//grow, we just therefore just grow by keeping the tail instead of
	//removing it.
	if !grow {
		state.snake = state.snake[1:]
	}

	state.snake = append(state.snake, newHead)
	state.draw(screen)
	
	state.lastHeading = heading
	//Resetting the direction to; since user input is ignored if it's
	//not "none". This is in order to avoid two direction changes within
	//a single screen-update.
	state.heading = none
}

// clearScreen only clears the fields of the screen that have already been
// drawn to according to the current state.
func (state *gameState) clearScreen(screen tcell.Screen) {
	if state.apple != nil {
		screen.SetCell(state.apple.x, state.apple.y, tcell.StyleDefault, ' ')
		screen.SetCell(state.apple.x+1, state.apple.y, tcell.StyleDefault, ' ')
	}

	for _, bodyPart := range state.snake {
		screen.SetCell(bodyPart.x, bodyPart.y, tcell.StyleDefault, ' ')
		screen.SetCell(bodyPart.x+1, bodyPart.y, tcell.StyleDefault, ' ')
	}

	//Clear bottombar staring at 7, sicne we want to leave "Score: "
	for i := 7; i < state.width; i++ {
		screen.SetCell(i, state.height, tcell.StyleDefault, ' ')
	}
}

// draw fills the screen according to state. It draws the apple and the
// snake, followed by pushing the update to the terminal. 
func (state *gameState) draw(screen tcell.Screen) {
	if state.apple != nil {
		screen.SetContent(state.apple.x, state.apple.y, apple[0], apple[1:], tcell.StyleDefault)
	}

	for index, bodyPart := range state.snake {
		if index == len(state.snake) - 1 {	
			screen.SetCell(bodyPart.x, bodyPart.y, greenStyle, fullBlock)
			screen.SetCell(bodyPart.x+1, bodyPart.y, greenStyle, fullBlock)			
		} else {
			screen.SetCell(bodyPart.x, bodyPart.y, tcell.StyleDefault, fullBlock)
			screen.SetCell(bodyPart.x+1, bodyPart.y, tcell.StyleDefault, fullBlock)
		}
	}

	//7, since we want to leave a space
	nextCell := 7
	for _, char := range []rune(strconv.Itoa(state.score)) {
		screen.SetCell(nextCell, state.height, tcell.StyleDefault, char)
		nextCell = nextCell + 1
	}
	

	screen.Show()
}

// addApple sets a new apple for the state. The apple will spawn anywhere,
// except for where any body part of the snake already is.
func (state *gameState) addApple() {
GEN_NEW_APPLE:
	newAppleX, newAppleY := generateRandomLocation(state.width, state.height)
	for _, bodyPart := range state.snake {
		if newAppleX == bodyPart.x && newAppleY == bodyPart.y {
			goto GEN_NEW_APPLE
		}
	}

	state.apple = &loc{newAppleX, newAppleY}
}

func generateRandomLocation(width, height int) (int, int) {
	rand.Seed(time.Now().Unix())

	x := rand.Intn(width)
	y := rand.Intn(height)

	if x%2 != 0 {
		x++
	}
	if x >= width-1 {
		x = x - 2
	}
	return x, y
}
