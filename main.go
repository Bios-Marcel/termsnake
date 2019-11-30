package main

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/gdamore/tcell"
)

var apple = []rune("🍎")

const fullBlock = '█'

type loc struct {
	x int
	y int
}

type gameState struct {
	snake   []*loc
	apple   *loc
	lock    *sync.Mutex
	heading direction
	width   int
	height  int
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
	snakeX, snakeY := generateRandomLocation(width, height)
	state := &gameState{
		lock:   &sync.Mutex{},
		width:  width,
		height: height,
		snake:  []*loc{{snakeX, snakeY}},
	}
	state.draw(screen)

	screen.Show()

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
	up    direction = 0
	right           = 1
	down            = 2
	left            = 3
)

func (state *gameState) changeDirection(newDirection direction) {
	state.lock.Lock()
	if len(state.snake) == 1 || state.heading == up && newDirection != down ||
		state.heading == left && newDirection != right || state.heading == right && newDirection != left ||
		state.heading == down && newDirection != up {
		state.heading = newDirection
	}

	state.lock.Unlock()
}

func (state *gameState) gameOver(screen tcell.Screen) {
	screen.Fini()
	fmt.Println("You died")
	os.Exit(0)
}

func (state *gameState) updateSnake(screen tcell.Screen) {
	state.lock.Lock()

	state.clearScreen(screen)

	tail := state.snake[0]

	var oldHead *loc
	if len(state.snake) == 0 {
		oldHead = tail
	} else {
		oldHead = state.snake[len(state.snake)-1]
	}

	oldHeadChar, _, _, _ := screen.GetContent(oldHead.x, oldHead.y)
	if oldHeadChar == 0 {
		state.gameOver(screen)
	}

	newHead := &loc{oldHead.x, oldHead.y}
	switch state.heading {
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
		state.addApple(screen)
	} else {
		if state.apple != nil && newHead.x == state.apple.x && newHead.y == state.apple.y {
			//TODO Check whether snake fills out the field
			grow = true

			state.addApple(screen)
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

	state.lock.Unlock()
}

func (state *gameState) clearScreen(screen tcell.Screen) {
	if state.apple != nil {
		screen.SetCell(state.apple.x, state.apple.y, tcell.StyleDefault, ' ')
		screen.SetCell(state.apple.x+1, state.apple.y, tcell.StyleDefault, ' ')
	}

	for _, bodyPart := range state.snake {
		screen.SetCell(bodyPart.x, bodyPart.y, tcell.StyleDefault, ' ')
		screen.SetCell(bodyPart.x+1, bodyPart.y, tcell.StyleDefault, ' ')
	}
}

func (state *gameState) draw(screen tcell.Screen) {
	if state.apple != nil {
		screen.SetContent(state.apple.x, state.apple.y, apple[0], apple[1:], tcell.StyleDefault)
	}

	for _, bodyPart := range state.snake {
		screen.SetCell(bodyPart.x, bodyPart.y, tcell.StyleDefault, fullBlock)
		screen.SetCell(bodyPart.x+1, bodyPart.y, tcell.StyleDefault, fullBlock)
	}

	screen.Show()
}

func (state *gameState) addApple(screen tcell.Screen) {
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
