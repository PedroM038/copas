package model

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"
)

// Naipe representa os naipes das cartas.
type Naipe string

const (
	NaipeCopas   Naipe = "Copas"
	NaipeEspadas Naipe = "Espadas"
	NaipeOuros   Naipe = "Ouros"
	NaipePaus    Naipe = "Paus"
)

// Valor representa o valor de uma carta (2 a 14, onde 11=J, 12=Q, 13=K, 14=A).
type Valor int

const (
	Dois   Valor = 2
	Tres   Valor = 3
	Quatro Valor = 4
	Cinco  Valor = 5
	Seis   Valor = 6
	Sete   Valor = 7
	Oito   Valor = 8
	Nove   Valor = 9
	Dez    Valor = 10
	Valete Valor = 11
	Dama   Valor = 12
	Rei    Valor = 13
	As     Valor = 14
)

// Card representa uma carta do baralho.
type Card struct {
	Naipe Naipe `json:"naipe"`
	Valor Valor `json:"valor"`
}

// String retorna a representação textual da carta.
func (c Card) String() string {
	valorStr := map[Valor]string{
		Valete: "J", Dama: "Q", Rei: "K", As: "A",
	}
	v, ok := valorStr[c.Valor]
	if !ok {
		v = fmt.Sprintf("%d", c.Valor)
	}
	return fmt.Sprintf("%s de %s", v, c.Naipe)
}

// GetPoints retorna os pontos que esta carta vale.
func (c Card) GetPoints() int {
	if c.Naipe == NaipeCopas {
		return 1 // Cada carta de Copas = 1 ponto
	}
	if c.Naipe == NaipeEspadas && c.Valor == Dama {
		return 13 // Dama de Espadas = 13 pontos
	}
	return 0 // Outras cartas = 0 pontos
}

// IsTwoOfClubs verifica se a carta é o 2 de paus.
func (c Card) IsTwoOfClubs() bool {
	return c.Naipe == NaipePaus && c.Valor == Dois
}

// IsQueenOfSpades verifica se a carta é a dama de espadas.
func (c Card) IsQueenOfSpades() bool {
	return c.Naipe == NaipeEspadas && c.Valor == Dama
}

// IsHearts verifica se a carta é de copas.
func (c Card) IsHearts() bool {
	return c.Naipe == NaipeCopas
}

// Player representa um jogador.
type Player struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Hand  []Card `json:"hand"`
	Score int    `json:"score"`
}

// HasCard verifica se o jogador tem uma carta específica.
func (p *Player) HasCard(card Card) bool {
	for _, c := range p.Hand {
		if c.Naipe == card.Naipe && c.Valor == card.Valor {
			return true
		}
	}
	return false
}

// RemoveCard remove uma carta da mão do jogador.
func (p *Player) RemoveCard(card Card) error {
	for i, c := range p.Hand {
		if c.Naipe == card.Naipe && c.Valor == card.Valor {
			p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
			return nil
		}
	}
	return errors.New("carta não encontrada na mão do jogador")
}

// HasSuit verifica se o jogador tem cartas de um naipe específico.
func (p *Player) HasSuit(naipe Naipe) bool {
	for _, c := range p.Hand {
		if c.Naipe == naipe {
			return true
		}
	}
	return false
}

// HasOnlyHearts verifica se o jogador tem apenas cartas de copas.
func (p *Player) HasOnlyHearts() bool {
	for _, c := range p.Hand {
		if c.Naipe != NaipeCopas {
			return false
		}
	}
	return len(p.Hand) > 0
}

// Trick representa uma rodada (vaza).
type Trick struct {
	Cards       []Card `json:"cards"`        // Cartas jogadas na ordem
	PlayerIDs   []int  `json:"player_ids"`   // IDs dos jogadores na ordem que jogaram
	WinnerID    int    `json:"winner_id"`    // ID do jogador que ganhou a vaza
	LeadingSuit Naipe  `json:"leading_suit"` // Naipe que iniciou a vaza
}

// Game representa o estado do jogo.
type Game struct {
	Players         []Player `json:"players"`
	CurrentTrick    Trick    `json:"current_trick"`
	CompletedTricks []Trick  `json:"completed_tricks"`
	CurrentPlayer   int      `json:"current_player"` // Índice do jogador atual
	HeartsBreoken   bool     `json:"hearts_broken"`  // Se copas já foram quebradas
	IsFirstTrick    bool     `json:"is_first_trick"` // Se é a primeira vaza da rodada
	GameOver        bool     `json:"game_over"`
	WinnerID        int      `json:"winner_id"`
}

// NewGame cria um novo jogo com 4 jogadores.
func NewGame(playerNames []string) (*Game, error) {
	if len(playerNames) != 4 {
		return nil, errors.New("o jogo precisa ter exatamente 4 jogadores")
	}

	players := make([]Player, 4)
	for i, name := range playerNames {
		players[i] = Player{
			ID:    i,
			Name:  name,
			Hand:  make([]Card, 0, 13),
			Score: 0,
		}
	}

	game := &Game{
		Players:         players,
		CurrentTrick:    Trick{Cards: make([]Card, 0, 4), PlayerIDs: make([]int, 0, 4)},
		CompletedTricks: make([]Trick, 0, 13),
		HeartsBreoken:   false,
		IsFirstTrick:    true,
		GameOver:        false,
	}

	game.dealCards()
	game.findStartingPlayer()

	return game, nil
}

// NewDeck retorna um baralho tradicional de 52 cartas.
func NewDeck() []Card {
	naipes := []Naipe{NaipeCopas, NaipeEspadas, NaipeOuros, NaipePaus}
	valores := []Valor{Dois, Tres, Quatro, Cinco, Seis, Sete, Oito, Nove, Dez, Valete, Dama, Rei, As}
	deck := make([]Card, 0, 52)
	for _, n := range naipes {
		for _, v := range valores {
			deck = append(deck, Card{Naipe: n, Valor: v})
		}
	}
	return deck
}

// dealCards embaralha e distribui as cartas para os jogadores.
func (g *Game) dealCards() {
	deck := NewDeck()

	// Embaralha o deck
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})

	// Distribui 13 cartas para cada jogador
	for i := 0; i < 52; i++ {
		playerIndex := i % 4
		g.Players[playerIndex].Hand = append(g.Players[playerIndex].Hand, deck[i])
	}
}

// findStartingPlayer encontra quem tem o 2 de paus para começar.
func (g *Game) findStartingPlayer() {
	twoOfClubs := Card{Naipe: NaipePaus, Valor: Dois}
	for i, player := range g.Players {
		if player.HasCard(twoOfClubs) {
			g.CurrentPlayer = i
			return
		}
	}
}

// IsValidPlay verifica se uma jogada é válida.
func (g *Game) IsValidPlay(playerID int, card Card) error {
	if g.GameOver {
		return errors.New("o jogo já terminou")
	}

	if playerID != g.CurrentPlayer {
		return errors.New("não é a vez deste jogador")
	}

	player := &g.Players[playerID]
	if !player.HasCard(card) {
		return errors.New("jogador não tem esta carta")
	}

	// Se é a primeira carta da vaza (primeira jogada da rodada)
	if len(g.CurrentTrick.Cards) == 0 {
		// Na primeira vaza do jogo, deve começar com 2 de paus
		if g.IsFirstTrick && !card.IsTwoOfClubs() {
			return errors.New("primeira jogada deve ser o 2 de paus")
		}

		// Copas não podem ser jogadas para iniciar vaza até serem quebradas
		if card.IsHearts() && !g.HeartsBreoken && !player.HasOnlyHearts() {
			return errors.New("copas não podem ser jogadas para iniciar vaza até serem quebradas")
		}

		return nil
	}

	// Se não é a primeira carta da vaza
	leadingSuit := g.CurrentTrick.LeadingSuit

	// Deve seguir o naipe se tiver
	if player.HasSuit(leadingSuit) {
		if card.Naipe != leadingSuit {
			return errors.New("deve seguir o naipe se tiver")
		}
	}

	// Na primeira vaza, não pode jogar copas nem dama de espadas
	if g.IsFirstTrick {
		if card.IsHearts() || card.IsQueenOfSpades() {
			return errors.New("na primeira vaza não pode jogar copas nem dama de espadas")
		}
	}

	return nil
}

// PlayCard executa uma jogada.
func (g *Game) PlayCard(playerID int, card Card) error {
	if err := g.IsValidPlay(playerID, card); err != nil {
		return err
	}

	player := &g.Players[playerID]
	if err := player.RemoveCard(card); err != nil {
		return err
	}

	// Se é a primeira carta da vaza, define o naipe líder
	if len(g.CurrentTrick.Cards) == 0 {
		g.CurrentTrick.LeadingSuit = card.Naipe
	}

	g.CurrentTrick.Cards = append(g.CurrentTrick.Cards, card)
	g.CurrentTrick.PlayerIDs = append(g.CurrentTrick.PlayerIDs, playerID)

	// Verifica se copas foram quebradas
	if card.IsHearts() {
		g.HeartsBreoken = true
	}

	// Se a vaza está completa (4 cartas)
	if len(g.CurrentTrick.Cards) == 4 {
		g.completeTrick()
	} else {
		// Próximo jogador
		g.CurrentPlayer = (g.CurrentPlayer + 1) % 4
	}

	return nil
}

func (g *Game) completeTrick() {
	// Encontra quem ganhou a vaza (carta mais alta do naipe líder)
	winnerIndex := 0
	highestValue := g.CurrentTrick.Cards[0].Valor
	leadingSuit := g.CurrentTrick.LeadingSuit

	for i := 1; i < len(g.CurrentTrick.Cards); i++ {
		card := g.CurrentTrick.Cards[i]
		if card.Naipe == leadingSuit && card.Valor > highestValue {
			highestValue = card.Valor
			winnerIndex = i
		}
	}

	winnerID := g.CurrentTrick.PlayerIDs[winnerIndex] // <-- Salva o vencedor antes de resetar

	g.CurrentTrick.WinnerID = winnerID
	g.CompletedTricks = append(g.CompletedTricks, g.CurrentTrick)

	// Calcula pontos da vaza
	points := 0
	for _, card := range g.CurrentTrick.Cards {
		points += card.GetPoints()
	}
	g.Players[winnerID].Score += points

	// Prepara próxima vaza
	g.CurrentTrick = Trick{Cards: make([]Card, 0, 4), PlayerIDs: make([]int, 0, 4)}
	g.CurrentPlayer = winnerID // <-- Usa o vencedor salvo
	g.IsFirstTrick = false

	// Verifica se a rodada terminou (13 vazas)
	if len(g.CompletedTricks) == 13 {
		g.endRound()
	}
}

// endRound finaliza uma rodada e verifica shoot the moon.
func (g *Game) endRound() {
	// Verifica shoot the moon
	for i := range g.Players {
		roundPoints := g.calculateRoundPoints(i)
		if roundPoints == 26 { // Pegou todas as copas + dama de espadas
			// Shoot the moon: todos os outros ganham 26 pontos
			for j := range g.Players {
				if j != i {
					g.Players[j].Score += 26
				}
			}
			// Remove os 26 pontos do jogador que fez shoot the moon
			g.Players[i].Score -= 26
			break
		}
	}

	// Verifica se alguém atingiu 100 pontos
	for i := range g.Players {
		if g.Players[i].Score >= 100 {
			g.GameOver = true
			g.findWinner()
			return
		}
	}

	// Se o jogo não terminou, prepara nova rodada
	g.startNewRound()
}

// calculateRoundPoints calcula quantos pontos um jogador fez nesta rodada.
func (g *Game) calculateRoundPoints(playerID int) int {
	points := 0
	for _, trick := range g.CompletedTricks {
		if trick.WinnerID == playerID {
			for _, card := range trick.Cards {
				points += card.GetPoints()
			}
		}
	}
	return points
}

func (g *Game) startNewRound() {
	// Limpa as vazas
	g.CompletedTricks = make([]Trick, 0, 13)
	g.HeartsBreoken = false
	g.IsFirstTrick = true

	// Limpa as mãos dos jogadores antes de redistribuir
	for i := range g.Players {
		g.Players[i].Hand = make([]Card, 0, 13)
	}

	// Redistribui as cartas
	g.dealCards()

	// Define o jogador que tem 2 de paus
	g.findStartingPlayer()
}

func (g *Game) findWinner() {
	minScore := g.Players[0].Score
	g.WinnerID = 0

	for i := 1; i < len(g.Players); i++ {
		if g.Players[i].Score < minScore {
			minScore = g.Players[i].Score
			g.WinnerID = i
		}
	}
}

// GetGameState retorna o estado atual do jogo.
func (g *Game) GetGameState() *Game {
	return g
}

// GetPlayerHand retorna a mão de um jogador específico.
func (g *Game) GetPlayerHand(playerID int) ([]Card, error) {
	if playerID < 0 || playerID >= len(g.Players) {
		return nil, errors.New("ID de jogador inválido")
	}
	return g.Players[playerID].Hand, nil
}

// GetScores retorna as pontuações de todos os jogadores.
func (g *Game) GetScores() map[int]int {
	scores := make(map[int]int)
	for _, player := range g.Players {
		scores[player.ID] = player.Score
	}
	return scores
}

// GetValidPlays retorna as jogadas válidas para um jogador.
func (g *Game) GetValidPlays(playerID int) []Card {
	if playerID != g.CurrentPlayer || g.GameOver {
		return []Card{}
	}

	validCards := []Card{}
	player := &g.Players[playerID]

	for _, card := range player.Hand {
		if g.IsValidPlay(playerID, card) == nil {
			validCards = append(validCards, card)
		}
	}

	return validCards
}

// SortHand ordena a mão de um jogador por naipe e valor.
func (g *Game) SortHand(playerID int) error {
	if playerID < 0 || playerID >= len(g.Players) {
		return errors.New("ID de jogador inválido")
	}

	sort.Slice(g.Players[playerID].Hand, func(i, j int) bool {
		hand := g.Players[playerID].Hand
		if hand[i].Naipe != hand[j].Naipe {
			// Ordena por naipe: Paus, Ouros, Espadas, Copas
			naipeOrder := map[Naipe]int{
				NaipePaus: 0, NaipeOuros: 1, NaipeEspadas: 2, NaipeCopas: 3,
			}
			return naipeOrder[hand[i].Naipe] < naipeOrder[hand[j].Naipe]
		}
		return hand[i].Valor < hand[j].Valor
	})

	return nil
}
