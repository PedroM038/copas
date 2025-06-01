package controller

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"copas/model"
	"copas/network"
)

// Tipos de mensagens específicas do jogo
const (
	GAME_MSG_START_GAME   = "START_GAME"
	GAME_MSG_PLAYER_READY = "PLAYER_READY"
	GAME_MSG_PLAY_CARD    = "PLAY_CARD"
	GAME_MSG_TRICK_RESULT = "TRICK_RESULT"
	GAME_MSG_ROUND_END    = "ROUND_END"
	GAME_MSG_GAME_OVER    = "GAME_OVER"
	GAME_MSG_GAME_STATE   = "GAME_STATE"
	GAME_MSG_REQUEST_PLAY = "REQUEST_PLAY"
	GAME_MSG_PLAYER_JOIN  = "PLAYER_JOIN"
)

// Estruturas para mensagens do jogo
type GameMessage struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

type PlayerJoinMessage struct {
	PlayerID   int    `json:"player_id"`
	PlayerName string `json:"player_name"`
}

type PlayCardMessage struct {
	PlayerID int        `json:"player_id"`
	Card     model.Card `json:"card"`
}

type TrickResultMessage struct {
	WinnerID   int          `json:"winner_id"`
	WinnerName string       `json:"winner_name"`
	Cards      []model.Card `json:"cards"`
	PlayerIDs  []int        `json:"player_ids"`
	Points     int          `json:"points"`
}

type GameStateMessage struct {
	CurrentPlayer int         `json:"current_player"`
	HeartsBreoken bool        `json:"hearts_broken"`
	IsFirstTrick  bool        `json:"is_first_trick"`
	TrickCount    int         `json:"trick_count"`
	Scores        map[int]int `json:"scores"`
	CurrentTrick  model.Trick `json:"current_trick"`
}

type GameController struct {
	node         *network.Node
	game         *model.Game
	logger       *log.Logger
	playerNames  map[int]string
	playersReady map[int]bool
	isHost       bool
	gameStarted  bool
	scanner      *bufio.Scanner
}

func NewGameController(node *network.Node, logger *log.Logger, isHost bool) *GameController {
	return &GameController{
		node:         node,
		logger:       logger,
		playerNames:  make(map[int]string),
		playersReady: make(map[int]bool),
		isHost:       isHost,
		gameStarted:  false,
		scanner:      bufio.NewScanner(os.Stdin),
	}
}

func (gc *GameController) Start() {
	gc.logger.Printf("🎮 Iniciando controlador do jogo - Host: %v", gc.isHost)

	// Configura nome do jogador
	gc.setupPlayerName()

	if gc.isHost {
		gc.logger.Println("🏠 Aguardando outros jogadores se conectarem...")
		go gc.hostRoutine()
	} else {
		gc.logger.Println("👤 Conectando ao jogo...")
		go gc.clientRoutine()
	}

	// Loop principal do jogo
	gc.gameLoop()
}

func (gc *GameController) setupPlayerName() {
	fmt.Printf("Digite seu nome: ")
	if gc.scanner.Scan() {
		name := strings.TrimSpace(gc.scanner.Text())
		if name == "" {
			name = fmt.Sprintf("Jogador %d", gc.node.ID)
		}
		gc.playerNames[gc.node.ID] = name
		gc.logger.Printf("👤 Nome do jogador definido: %s", name)
	}
}

func (gc *GameController) hostRoutine() {
	// Anuncia que é o host
	gc.announcePlayer()

	// Aguarda outros jogadores
	gc.waitForPlayers()

	// Inicia o jogo
	time.Sleep(2 * time.Second)
	gc.startGame()
}

func (gc *GameController) clientRoutine() {
	// Anuncia entrada no jogo
	gc.announcePlayer()

	// Aguarda início do jogo
	for !gc.gameStarted {
		time.Sleep(500 * time.Millisecond)
	}
}

func (gc *GameController) announcePlayer() {
	gc.waitForToken()

	joinMsg := PlayerJoinMessage{
		PlayerID:   gc.node.ID,
		PlayerName: gc.playerNames[gc.node.ID],
	}

	gameMsg := GameMessage{
		Type:    GAME_MSG_PLAYER_JOIN,
		Content: joinMsg,
	}

	if err := gc.node.SendBroadcast(gameMsg); err != nil {
		gc.logger.Printf("❌ Erro ao anunciar jogador: %v", err)
	} else {
		gc.logger.Printf("📢 Jogador anunciado na rede")
	}

	gc.playersReady[gc.node.ID] = true
	gc.node.PassToken((gc.node.ID + 1) % 4)
}

func (gc *GameController) waitForPlayers() {
	gc.logger.Println("⏳ Aguardando todos os jogadores...")

	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			gc.logger.Println("⚠️ Timeout aguardando jogadores")
			return
		case <-ticker.C:
			if len(gc.playerNames) >= 4 {
				gc.logger.Println("✅ Todos os jogadores conectados!")
				return
			}
			gc.logger.Printf("🔄 Aguardando jogadores (%d/4)", len(gc.playerNames))
		}
	}
}

func (gc *GameController) startGame() {
	gc.waitForToken()

	// Cria lista de nomes na ordem correta
	playerNames := make([]string, 4)
	for i := 0; i < 4; i++ {
		if name, exists := gc.playerNames[i]; exists {
			playerNames[i] = name
		} else {
			playerNames[i] = fmt.Sprintf("Jogador %d", i)
		}
	}

	// Inicia o jogo
	game, err := model.NewGame(playerNames)
	if err != nil {
		gc.logger.Printf("❌ Erro ao criar jogo: %v", err)
		return
	}

	gc.game = game
	gc.gameStarted = true

	// Anuncia início do jogo
	startMsg := GameMessage{
		Type:    GAME_MSG_START_GAME,
		Content: playerNames,
	}

	if err := gc.node.SendBroadcast(startMsg); err != nil {
		gc.logger.Printf("❌ Erro ao anunciar início do jogo: %v", err)
	} else {
		gc.logger.Println("🎮 Jogo iniciado!")
	}

	gc.node.PassToken((gc.node.ID + 1) % 4)
}

func (gc *GameController) gameLoop() {
	for {
		if gc.gameStarted && gc.game != nil {
			if gc.game.GameOver {
				gc.showFinalResults()
				break
			}

			if gc.game.CurrentPlayer == gc.node.ID {
				gc.playTurn()
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (gc *GameController) playTurn() {
	gc.logger.Printf("🎯 Sua vez de jogar!")

	// Ordena a mão
	gc.game.SortHand(gc.node.ID)

	// Mostra estado do jogo
	gc.showGameState()

	// Mostra mão do jogador
	gc.showPlayerHand()

	// Obtém jogadas válidas
	validPlays := gc.game.GetValidPlays(gc.node.ID)
	if len(validPlays) == 0 {
		gc.logger.Println("❌ Nenhuma jogada válida disponível")
		return
	}

	// Solicita escolha do jogador
	card := gc.getPlayerChoice(validPlays)

	// Executa a jogada
	if err := gc.game.PlayCard(gc.node.ID, card); err != nil {
		gc.logger.Printf("❌ Erro ao jogar carta: %v", err)
		return
	}

	gc.logger.Printf("✅ Você jogou: %s", card.String())

	// Anuncia a jogada
	gc.announcePlay(card)

	// Verifica se a vaza foi completada
	if len(gc.game.CurrentTrick.Cards) == 4 {
		gc.announceTrickResult()
	} else {
		gc.announceGameState()
	}
}

func (gc *GameController) getPlayerChoice(validPlays []model.Card) model.Card {
	for {
		fmt.Printf("Escolha uma carta (1-%d): ", len(validPlays))
		if gc.scanner.Scan() {
			choice, err := strconv.Atoi(strings.TrimSpace(gc.scanner.Text()))
			if err == nil && choice >= 1 && choice <= len(validPlays) {
				return validPlays[choice-1]
			}
		}
		fmt.Println("❌ Escolha inválida. Tente novamente.")
	}
}

func (gc *GameController) announcePlay(card model.Card) {
	gc.waitForToken()

	playMsg := PlayCardMessage{
		PlayerID: gc.node.ID,
		Card:     card,
	}

	gameMsg := GameMessage{
		Type:    GAME_MSG_PLAY_CARD,
		Content: playMsg,
	}

	if err := gc.node.SendBroadcast(gameMsg); err != nil {
		gc.logger.Printf("❌ Erro ao anunciar jogada: %v", err)
	}

	gc.node.PassToken((gc.node.ID + 1) % 4)
}

func (gc *GameController) announceTrickResult() {
	gc.waitForToken()

	lastTrick := gc.game.CompletedTricks[len(gc.game.CompletedTricks)-1]
	points := 0
	for _, card := range lastTrick.Cards {
		points += card.GetPoints()
	}

	resultMsg := TrickResultMessage{
		WinnerID:   lastTrick.WinnerID,
		WinnerName: gc.playerNames[lastTrick.WinnerID],
		Cards:      lastTrick.Cards,
		PlayerIDs:  lastTrick.PlayerIDs,
		Points:     points,
	}

	gameMsg := GameMessage{
		Type:    GAME_MSG_TRICK_RESULT,
		Content: resultMsg,
	}

	if err := gc.node.SendBroadcast(gameMsg); err != nil {
		gc.logger.Printf("❌ Erro ao anunciar resultado da vaza: %v", err)
	}

	gc.node.PassToken((gc.node.ID + 1) % 4)
}

func (gc *GameController) announceGameState() {
	gc.waitForToken()

	stateMsg := GameStateMessage{
		CurrentPlayer: gc.game.CurrentPlayer,
		HeartsBreoken: gc.game.HeartsBreoken,
		IsFirstTrick:  gc.game.IsFirstTrick,
		TrickCount:    len(gc.game.CompletedTricks),
		Scores:        gc.game.GetScores(),
		CurrentTrick:  gc.game.CurrentTrick,
	}

	gameMsg := GameMessage{
		Type:    GAME_MSG_GAME_STATE,
		Content: stateMsg,
	}

	if err := gc.node.SendBroadcast(gameMsg); err != nil {
		gc.logger.Printf("❌ Erro ao anunciar estado do jogo: %v", err)
	}

	gc.node.PassToken((gc.node.ID + 1) % 4)
}

func (gc *GameController) ProcessGameMessage(msgData interface{}) {
	data, err := json.Marshal(msgData)
	if err != nil {
		gc.logger.Printf("❌ Erro ao processar mensagem do jogo: %v", err)
		return
	}

	var gameMsg GameMessage
	if err := json.Unmarshal(data, &gameMsg); err != nil {
		gc.logger.Printf("❌ Erro ao decodificar mensagem do jogo: %v", err)
		return
	}

	switch gameMsg.Type {
	case GAME_MSG_PLAYER_JOIN:
		gc.handlePlayerJoin(gameMsg.Content)
	case GAME_MSG_START_GAME:
		gc.handleStartGame(gameMsg.Content)
	case GAME_MSG_PLAY_CARD:
		gc.handlePlayCard(gameMsg.Content)
	case GAME_MSG_TRICK_RESULT:
		gc.handleTrickResult(gameMsg.Content)
	case GAME_MSG_GAME_STATE:
		gc.handleGameState(gameMsg.Content)
	}
}

func (gc *GameController) handlePlayerJoin(content interface{}) {
	data, _ := json.Marshal(content)
	var joinMsg PlayerJoinMessage
	if err := json.Unmarshal(data, &joinMsg); err != nil {
		return
	}

	gc.playerNames[joinMsg.PlayerID] = joinMsg.PlayerName
	gc.playersReady[joinMsg.PlayerID] = true

	gc.logger.Printf("👤 Jogador conectado: %s (ID: %d)", joinMsg.PlayerName, joinMsg.PlayerID)
}

func (gc *GameController) handleStartGame(content interface{}) {
	if gc.gameStarted {
		return
	}

	data, _ := json.Marshal(content)
	var playerNames []string
	if err := json.Unmarshal(data, &playerNames); err != nil {
		return
	}

	game, err := model.NewGame(playerNames)
	if err != nil {
		gc.logger.Printf("❌ Erro ao criar jogo: %v", err)
		return
	}

	gc.game = game
	gc.gameStarted = true

	gc.logger.Println("🎮 Jogo iniciado!")
	gc.showGameInfo()
}

func (gc *GameController) handlePlayCard(content interface{}) {
	if gc.game == nil || gc.game.GameOver {
		return
	}

	data, _ := json.Marshal(content)
	var playMsg PlayCardMessage
	if err := json.Unmarshal(data, &playMsg); err != nil {
		return
	}

	if playMsg.PlayerID == gc.node.ID {
		return // Ignora próprias jogadas
	}

	playerName := gc.playerNames[playMsg.PlayerID]
	gc.logger.Printf("🎯 %s jogou: %s", playerName, playMsg.Card.String())
}

func (gc *GameController) handleTrickResult(content interface{}) {
	data, _ := json.Marshal(content)
	var resultMsg TrickResultMessage
	if err := json.Unmarshal(data, &resultMsg); err != nil {
		return
	}

	gc.logger.Printf("🏆 Vaza ganha por: %s (%d pontos)", resultMsg.WinnerName, resultMsg.Points)
	gc.showCurrentScores()
}

func (gc *GameController) handleGameState(content interface{}) {
	data, _ := json.Marshal(content)
	var stateMsg GameStateMessage
	if err := json.Unmarshal(data, &stateMsg); err != nil {
		return
	}

	currentPlayerName := gc.playerNames[stateMsg.CurrentPlayer]
	gc.logger.Printf("🎮 Vez de: %s | Vazas: %d/13", currentPlayerName, stateMsg.TrickCount)
}

func (gc *GameController) showGameInfo() {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("🎮 JOGO DE COPAS INICIADO")
	fmt.Println(strings.Repeat("=", 50))

	fmt.Println("👥 Jogadores:")
	for i, player := range gc.game.Players {
		marker := ""
		if i == gc.node.ID {
			marker = " (VOCÊ)"
		}
		fmt.Printf("  %d. %s%s\n", i+1, player.Name, marker)
	}
	fmt.Println()
}

func (gc *GameController) showGameState() {
	fmt.Println("\n" + strings.Repeat("-", 40))
	fmt.Printf("🎮 VAZA %d/13\n", len(gc.game.CompletedTricks)+1)

	if gc.game.IsFirstTrick {
		fmt.Println("🚫 Primeira vaza - Não pode jogar copas nem dama de espadas")
	}

	if gc.game.HeartsBreoken {
		fmt.Println("💔 Copas foram quebradas")
	}

	// Mostra cartas já jogadas na vaza atual
	if len(gc.game.CurrentTrick.Cards) > 0 {
		fmt.Printf("🃏 Cartas na mesa: ")
		for i, card := range gc.game.CurrentTrick.Cards {
			playerName := gc.playerNames[gc.game.CurrentTrick.PlayerIDs[i]]
			fmt.Printf("%s(%s) ", card.String(), playerName)
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("-", 40))
}

func (gc *GameController) showPlayerHand() {
	hand, _ := gc.game.GetPlayerHand(gc.node.ID)
	validPlays := gc.game.GetValidPlays(gc.node.ID)

	fmt.Printf("\n🃏 Sua mão (%d cartas):\n", len(hand))

	validMap := make(map[string]bool)
	for _, card := range validPlays {
		validMap[card.String()] = true
	}

	for i, card := range hand {
		marker := ""
		if validMap[card.String()] {
			marker = " ✅"
		} else {
			marker = " ❌"
		}

		fmt.Printf("  %d. %s%s\n", i+1, card.String(), marker)
	}
	fmt.Println()
}

func (gc *GameController) showCurrentScores() {
	if gc.game == nil {
		return
	}

	scores := gc.game.GetScores()
	fmt.Println("\n📊 Pontuações atuais:")
	for i, player := range gc.game.Players {
		marker := ""
		if i == gc.node.ID {
			marker = " (VOCÊ)"
		}
		fmt.Printf("  %s: %d pontos%s\n", player.Name, scores[i], marker)
	}
	fmt.Println()
}

func (gc *GameController) showFinalResults() {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("🏁 FIM DE JOGO!")
	fmt.Println(strings.Repeat("=", 50))

	gc.showCurrentScores()

	winner := gc.game.Players[gc.game.WinnerID]
	fmt.Printf("🏆 VENCEDOR: %s com %d pontos!\n", winner.Name, winner.Score)

	if gc.game.WinnerID == gc.node.ID {
		fmt.Println("🎉 PARABÉNS! VOCÊ VENCEU!")
	}

	fmt.Println(strings.Repeat("=", 50))
}

func (gc *GameController) waitForToken() {
	timeout := time.After(15 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			gc.logger.Println("⚠️ Timeout aguardando bastão")
			return
		case <-ticker.C:
			_, hasToken := gc.node.GetState()
			if hasToken {
				return
			}
		}
	}
}
