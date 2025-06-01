package controller

import (
	"copas/model"
	"copas/network"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Estados do jogo distribuído
const (
	GAME_STATE_WAITING  = "WAITING"  // Aguardando jogadores
	GAME_STATE_STARTING = "STARTING" // Iniciando jogo
	GAME_STATE_PLAYING  = "PLAYING"  // Jogo em andamento
	GAME_STATE_FINISHED = "FINISHED" // Jogo finalizado
	GAME_STATE_ERROR    = "ERROR"    // Erro no jogo
)

// Tipos de ações do jogo
const (
	ACTION_JOIN_GAME   = "JOIN_GAME"
	ACTION_START_GAME  = "START_GAME"
	ACTION_PLAY_CARD   = "PLAY_CARD"
	ACTION_GAME_STATE  = "GAME_STATE"
	ACTION_PLAYER_JOIN = "PLAYER_JOIN"
	ACTION_GAME_UPDATE = "GAME_UPDATE"
	ACTION_GAME_END    = "GAME_END"
	ACTION_ERROR       = "ERROR"
)

// Estruturas para comunicação
type JoinGameData struct {
	PlayerID   int    `json:"player_id"`
	PlayerName string `json:"player_name"`
}

type PlayCardData struct {
	PlayerID int        `json:"player_id"`
	Card     model.Card `json:"card"`
}

type GameStateData struct {
	Game    *model.Game    `json:"game"`
	Players map[int]string `json:"players"` // ID -> Nome
	State   string         `json:"state"`
}

type GameUpdateData struct {
	LastAction    string      `json:"last_action"`
	CurrentPlayer int         `json:"current_player"`
	LastCard      *model.Card `json:"last_card,omitempty"`
	LastPlayerID  int         `json:"last_player_id"`
	Scores        map[int]int `json:"scores"`
}

// Controller principal do jogo
type GameController struct {
	nodeID     int
	playerName string
	isHost     bool
	node       *network.RingNode
	game       *model.Game
	gameState  string
	players    map[int]string // ID -> Nome

	// Controle de estado
	mutex       sync.RWMutex
	waitingChan chan bool

	// Callbacks para UI
	onGameUpdate func(*GameStateData)
	onGameEnd    func(winnerID int, winnerName string)
	onError      func(error)
	onPlayerJoin func(playerID int, playerName string)
}

// Cria um novo controller
func NewGameController(nodeID int, playerName string, isHost bool) *GameController {
	return &GameController{
		nodeID:      nodeID,
		playerName:  playerName,
		isHost:      isHost,
		gameState:   GAME_STATE_WAITING,
		players:     make(map[int]string),
		waitingChan: make(chan bool, 1),
	}
}

// Inicia o controller
func (gc *GameController) Start(listenAddr, nextNodeAddr string) error {
	// Configura o nó da rede
	config := &network.NodeConfig{
		ID:           gc.nodeID,
		ListenAddr:   listenAddr,
		NextNodeAddr: nextNodeAddr,
		RingSize:     4,
		UseColors:    true,
	}

	var err error
	gc.node, err = network.NewRingNode(config)
	if err != nil {
		return fmt.Errorf("erro ao criar nó da rede: %v", err)
	}

	// Registra handlers de mensagem do jogo
	gc.node.RegisterMessageHandler(network.MSG_GAME, gc.handleGameMessage)

	// Inicia o nó
	err = gc.node.Start()
	if err != nil {
		return fmt.Errorf("erro ao iniciar nó: %v", err)
	}

	fmt.Printf("🌐 Nó %d iniciado\n", gc.nodeID)
	fmt.Printf("👤 Jogador: %s\n", gc.playerName)

	// Adiciona o próprio jogador
	gc.players[gc.nodeID] = gc.playerName

	// Se é o host, aguarda outros jogadores
	if gc.isHost {
		fmt.Println("🎮 Aguardando outros jogadores se conectarem...")
		go gc.hostWaitForPlayers()
	} else {
		// Se não é host, envia pedido para entrar no jogo
		go gc.requestJoinGame()
	}

	return nil
}

// Para o controller
func (gc *GameController) Stop() {
	if gc.node != nil {
		gc.node.Stop()
	}
}

// Host aguarda outros jogadores
func (gc *GameController) hostWaitForPlayers() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			gc.mutex.RLock()
			playerCount := len(gc.players)
			gc.mutex.RUnlock()

			fmt.Printf("👥 Jogadores conectados: %d/4\n", playerCount)

			if playerCount == 4 {
				fmt.Println("🎯 Todos os jogadores conectados! Iniciando jogo...")
				gc.startGame()
				return
			}

		case <-gc.waitingChan:
			return
		}
	}
}

// Solicita entrada no jogo
func (gc *GameController) requestJoinGame() {
	time.Sleep(2 * time.Second) // Aguarda rede estabilizar

	joinData := JoinGameData{
		PlayerID:   gc.nodeID,
		PlayerName: gc.playerName,
	}

	// Envia para o host (nó 0)
	err := gc.node.SendGameMessage(0, ACTION_JOIN_GAME, joinData)
	if err != nil {
		fmt.Printf("❌ Erro ao solicitar entrada no jogo: %v\n", err)
	} else {
		fmt.Println("📤 Solicitação de entrada enviada ao host")
	}
}

// Inicia o jogo
func (gc *GameController) startGame() {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	if len(gc.players) != 4 {
		fmt.Printf("❌ Não é possível iniciar: apenas %d jogadores\n", len(gc.players))
		return
	}

	// Cria lista de nomes na ordem dos IDs
	playerNames := make([]string, 4)
	for i := 0; i < 4; i++ {
		if name, exists := gc.players[i]; exists {
			playerNames[i] = name
		} else {
			fmt.Printf("❌ Jogador %d não encontrado\n", i)
			return
		}
	}

	// Cria o jogo
	var err error
	gc.game, err = model.NewGame(playerNames)
	if err != nil {
		fmt.Printf("❌ Erro ao criar jogo: %v\n", err)
		return
	}

	gc.gameState = GAME_STATE_PLAYING
	fmt.Println("🎮 Jogo iniciado!")

	// Envia estado inicial para todos
	gc.broadcastGameState()
	gc.showGameStatus()
}

// Manipula mensagens do jogo
func (gc *GameController) handleGameMessage(msg *network.Message) {
	gameContent, err := msg.GetGameContent()
	if err != nil {
		fmt.Printf("❌ Erro ao processar mensagem do jogo: %v\n", err)
		return
	}

	switch gameContent.Action {
	case ACTION_JOIN_GAME:
		gc.handleJoinGame(msg.From, gameContent.Data)
	case ACTION_PLAY_CARD:
		gc.handlePlayCard(gameContent.Data)
	case ACTION_GAME_STATE:
		gc.handleGameState(gameContent.Data)
	case ACTION_GAME_UPDATE:
		gc.handleGameUpdate(gameContent.Data)
	case ACTION_GAME_END:
		gc.handleGameEnd(gameContent.Data)
	default:
		fmt.Printf("⚠️  Ação desconhecida: %s\n", gameContent.Action)
	}
}

// Manipula pedido de entrada no jogo
func (gc *GameController) handleJoinGame(playerID int, data interface{}) {
	if !gc.isHost {
		return // Apenas o host processa
	}

	joinData := &JoinGameData{}
	if err := gc.unmarshalData(data, joinData); err != nil {
		fmt.Printf("❌ Erro ao processar entrada: %v\n", err)
		return
	}

	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	if len(gc.players) >= 4 {
		fmt.Printf("⚠️  Jogo lotado, rejeitando jogador %s\n", joinData.PlayerName)
		return
	}

	gc.players[joinData.PlayerID] = joinData.PlayerName
	fmt.Printf("👤 Jogador %s (ID: %d) entrou no jogo\n", joinData.PlayerName, joinData.PlayerID)

	if gc.onPlayerJoin != nil {
		gc.onPlayerJoin(joinData.PlayerID, joinData.PlayerName)
	}
}

// Joga uma carta
func (gc *GameController) PlayCard(card model.Card) error {
	gc.mutex.RLock()
	if gc.game == nil || gc.gameState != GAME_STATE_PLAYING {
		gc.mutex.RUnlock()
		return fmt.Errorf("jogo não está em andamento")
	}

	if gc.game.CurrentPlayer != gc.nodeID {
		gc.mutex.RUnlock()
		return fmt.Errorf("não é sua vez")
	}
	gc.mutex.RUnlock()

	// Valida a jogada localmente
	if err := gc.game.IsValidPlay(gc.nodeID, card); err != nil {
		return fmt.Errorf("jogada inválida: %v", err)
	}

	// Envia a jogada para todos
	playData := PlayCardData{
		PlayerID: gc.nodeID,
		Card:     card,
	}

	return gc.node.BroadcastMessage(map[string]interface{}{
		"action": ACTION_PLAY_CARD,
		"data":   playData,
	})
}

// Manipula jogada de carta
func (gc *GameController) handlePlayCard(data interface{}) {
	playData := &PlayCardData{}
	if err := gc.unmarshalData(data, playData); err != nil {
		fmt.Printf("❌ Erro ao processar jogada: %v\n", err)
		return
	}

	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	if gc.game == nil {
		return
	}

	// Executa a jogada
	err := gc.game.PlayCard(playData.PlayerID, playData.Card)
	if err != nil {
		fmt.Printf("❌ Erro na jogada de %s: %v\n", gc.players[playData.PlayerID], err)
		return
	}

	playerName := gc.players[playData.PlayerID]
	fmt.Printf("🃏 %s jogou: %s\n", playerName, playData.Card.String())

	// Verifica se o jogo terminou
	if gc.game.GameOver {
		gc.gameState = GAME_STATE_FINISHED
		winnerName := gc.players[gc.game.WinnerID]
		fmt.Printf("🏆 Jogo terminado! Vencedor: %s\n", winnerName)

		if gc.onGameEnd != nil {
			gc.onGameEnd(gc.game.WinnerID, winnerName)
		}
		return
	}

	// Envia atualização do jogo
	if gc.isHost {
		gc.broadcastGameUpdate(playData.PlayerID, &playData.Card)
	}

	gc.showGameStatus()
}

// Envia estado do jogo
func (gc *GameController) broadcastGameState() {
	if !gc.isHost || gc.game == nil {
		return
	}

	stateData := GameStateData{
		Game:    gc.game,
		Players: gc.players,
		State:   gc.gameState,
	}

	gc.node.BroadcastMessage(map[string]interface{}{
		"action": ACTION_GAME_STATE,
		"data":   stateData,
	})
}

// Envia atualização do jogo
func (gc *GameController) broadcastGameUpdate(lastPlayerID int, lastCard *model.Card) {
	if !gc.isHost || gc.game == nil {
		return
	}

	updateData := GameUpdateData{
		LastAction:    ACTION_PLAY_CARD,
		CurrentPlayer: gc.game.CurrentPlayer,
		LastCard:      lastCard,
		LastPlayerID:  lastPlayerID,
		Scores:        gc.game.GetScores(),
	}

	gc.node.BroadcastMessage(map[string]interface{}{
		"action": ACTION_GAME_UPDATE,
		"data":   updateData,
	})
}

// Manipula estado do jogo recebido
func (gc *GameController) handleGameState(data interface{}) {
	stateData := &GameStateData{}
	if err := gc.unmarshalData(data, stateData); err != nil {
		fmt.Printf("❌ Erro ao processar estado: %v\n", err)
		return
	}

	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	gc.game = stateData.Game
	gc.players = stateData.Players
	gc.gameState = stateData.State

	fmt.Println("🔄 Estado do jogo atualizado")
	gc.showGameStatus()

	if gc.onGameUpdate != nil {
		gc.onGameUpdate(stateData)
	}
}

// Manipula atualização do jogo
func (gc *GameController) handleGameUpdate(data interface{}) {
	updateData := &GameUpdateData{}
	if err := gc.unmarshalData(data, updateData); err != nil {
		fmt.Printf("❌ Erro ao processar atualização: %v\n", err)
		return
	}

	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	if gc.game != nil {
		gc.game.CurrentPlayer = updateData.CurrentPlayer
		// Atualiza scores
		for playerID, score := range updateData.Scores {
			if playerID < len(gc.game.Players) {
				gc.game.Players[playerID].Score = score
			}
		}
	}

	gc.showGameStatus()

	if gc.onGameUpdate != nil {
		stateData := &GameStateData{
			Game:    gc.game,
			Players: gc.players,
			State:   gc.gameState,
		}
		gc.onGameUpdate(stateData)
	}
}

// Manipula fim do jogo
func (gc *GameController) handleGameEnd(data interface{}) {
	fmt.Println("🏁 Jogo finalizado!")
	gc.mutex.Lock()
	gc.gameState = GAME_STATE_FINISHED
	gc.mutex.Unlock()
}

// Mostra status do jogo
func (gc *GameController) showGameStatus() {
	if gc.game == nil {
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Printf("🎮 JOGO COPAS - Rodada %d\n", len(gc.game.CompletedTricks)+1)
	fmt.Println(strings.Repeat("=", 50))

	// Mostra pontuações
	fmt.Println("📊 PONTUAÇÕES:")
	for i, player := range gc.game.Players {
		indicator := "  "
		if i == gc.game.CurrentPlayer {
			indicator = "👉"
		}
		fmt.Printf("%s %s: %d pontos\n", indicator, player.Name, player.Score)
	}

	// Mostra vaza atual
	if len(gc.game.CurrentTrick.Cards) > 0 {
		fmt.Println("\n🃏 VAZA ATUAL:")
		for i, card := range gc.game.CurrentTrick.Cards {
			playerID := gc.game.CurrentTrick.PlayerIDs[i]
			fmt.Printf("  %s: %s\n", gc.players[playerID], card.String())
		}
	}

	// Mostra sua mão
	if !gc.game.GameOver {
		hand, err := gc.game.GetPlayerHand(gc.nodeID)
		if err == nil {
			fmt.Printf("\n🤚 SUA MÃO (%s):\n", gc.playerName)
			gc.game.SortHand(gc.nodeID)
			for i, card := range hand {
				fmt.Printf("  %d. %s\n", i+1, card.String())
			}

			if gc.game.CurrentPlayer == gc.nodeID {
				validPlays := gc.game.GetValidPlays(gc.nodeID)
				fmt.Printf("\n✅ JOGADAS VÁLIDAS: %d cartas\n", len(validPlays))
				fmt.Println("Digite o número da carta para jogar:")
			} else {
				currentPlayerName := gc.players[gc.game.CurrentPlayer]
				fmt.Printf("\n⏳ Aguardando jogada de %s...\n", currentPlayerName)
			}
		}
	}

	fmt.Println(strings.Repeat("=", 50))
}

// Utilitários
func (gc *GameController) unmarshalData(data interface{}, target interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, target)
}

// Getters para estado
func (gc *GameController) GetGame() *model.Game {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()
	return gc.game
}

func (gc *GameController) GetGameState() string {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()
	return gc.gameState
}

func (gc *GameController) GetPlayers() map[int]string {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()
	playersCopy := make(map[int]string)
	for k, v := range gc.players {
		playersCopy[k] = v
	}
	return playersCopy
}

func (gc *GameController) IsMyTurn() bool {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()
	return gc.game != nil && gc.game.CurrentPlayer == gc.nodeID && !gc.game.GameOver
}

func (gc *GameController) GetNodeID() int {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()
	return gc.nodeID
}

// Callbacks
func (gc *GameController) SetOnGameUpdate(callback func(*GameStateData)) {
	gc.onGameUpdate = callback
}

func (gc *GameController) SetOnGameEnd(callback func(winnerID int, winnerName string)) {
	gc.onGameEnd = callback
}

func (gc *GameController) SetOnError(callback func(error)) {
	gc.onError = callback
}

func (gc *GameController) SetOnPlayerJoin(callback func(playerID int, playerName string)) {
	gc.onPlayerJoin = callback
}
