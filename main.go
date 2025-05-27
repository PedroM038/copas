package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"copas/network"
)

// Configurações da rede para 4 jogadores
var networkConfig = map[int]map[string]string{
	0: {"local": "localhost:8000", "next": "localhost:8001"},
	1: {"local": "localhost:8001", "next": "localhost:8002"},
	2: {"local": "localhost:8002", "next": "localhost:8003"},
	3: {"local": "localhost:8003", "next": "localhost:8000"},
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Uso: go run main.go <player_id>")
		fmt.Println("player_id deve ser 0, 1, 2 ou 3")
		os.Exit(1)
	}

	playerID, err := strconv.Atoi(os.Args[1])
	if err != nil || playerID < 0 || playerID > 3 {
		fmt.Println("player_id deve ser um número entre 0 e 3")
		os.Exit(1)
	}

	// Inicializar rede
	config := networkConfig[playerID]
	net, err := network.NewNetwork(playerID, config["local"], config["next"])
	if err != nil {
		log.Fatalf("Erro ao inicializar rede: %v", err)
	}
	defer net.Close()

	fmt.Printf("=== Jogador %d iniciado ===\n", playerID)
	fmt.Printf("Local: %s, Próximo: %s\n", config["local"], config["next"])

	if playerID == 0 {
		fmt.Println("🎯 Este jogador possui o token inicial!")
		fmt.Println("⏳ Aguardando outros jogadores se conectarem...")
	} else {
		fmt.Println("⏳ Aguardando rede estabilizar...")
	}

	// Aguardar um pouco para todos os jogadores iniciarem
	time.Sleep(2 * time.Second)

	// Escutar mensagens primeiro
	go listenForMessages(net, playerID)

	// Aguardar mais um pouco antes de iniciar simulação
	time.Sleep(2 * time.Second)

	// Iniciar simulação
	go simulatePlayer(net, playerID)

	// Manter o programa rodando
	select {}
}

func simulatePlayer(net *network.Network, playerID int) {
	fmt.Printf("🚀 Iniciando simulação para jogador %d\n", playerID)

	// Aguardar estabilização da rede
	time.Sleep(3 * time.Second)

	// Enviar mensagem de descoberta (broadcast)
	discoveryMsg, _ := network.NewDiscoveryMessage(
		playerID, -1, playerID,
		fmt.Sprintf("Jogador_%d", playerID),
		true,
	)
	net.Send(discoveryMsg)
	fmt.Printf("📡 Enviou mensagem de descoberta\n")

	// Loop principal do jogador
	actionCount := 0
	for {
		// Se tem o token, fazer uma ação
		if net.HasTokenNow() {
			fmt.Printf("🎯 Tenho o token! Executando ação %d\n", actionCount+1)

			switch actionCount {
			case 0:
				// Primeira ação: enviar mensagem de estado
				sendStateMessage(net, playerID)
			case 1:
				// Segunda ação: enviar mensagem de troca de cartas
				sendPassMessage(net, playerID)
			case 2:
				// Terceira ação: enviar mensagem de jogada
				sendPlayMessage(net, playerID)
			default:
				// Ações subsequentes: apenas passar o token
				fmt.Printf("⏭️ Passando token para o próximo jogador\n")
			}

			actionCount++

			// Aguardar um pouco antes de passar o token
			time.Sleep(2 * time.Second) // Aumentar delay

			// Passar o token
			err := net.PassToken()
			if err != nil {
				fmt.Printf("❌ Erro ao passar token: %v\n", err)
			} else {
				fmt.Printf("✅ Token passado para jogador %d\n", (playerID+1)%4)
			}
		}

		// Aguardar antes da próxima verificação
		time.Sleep(500 * time.Millisecond)
	}
}

// ...existing code... (resto das funções permanecem iguais)

func sendStateMessage(net *network.Network, playerID int) {
	// Simular estado do jogo
	gameState := map[string]interface{}{
		"round":         1,
		"current_turn":  playerID,
		"cards_played":  []string{},
		"scores":        []int{0, 0, 0, 0},
		"hearts_broken": false,
	}

	gameStateBytes, _ := json.Marshal(gameState)
	msg, _ := network.NewStateMessage(
		playerID, -1, // broadcast
		json.RawMessage(gameStateBytes),
	)

	net.Send(msg)
	fmt.Printf("🎮 Enviou estado do jogo\n")
}

func sendPassMessage(net *network.Network, playerID int) {
	// Simular troca de cartas
	cards := []map[string]interface{}{
		{"suit": "hearts", "rank": "A"},
		{"suit": "spades", "rank": "Q"},
		{"suit": "hearts", "rank": "K"},
	}

	var cardMessages []json.RawMessage
	for _, card := range cards {
		cardBytes, _ := json.Marshal(card)
		cardMessages = append(cardMessages, json.RawMessage(cardBytes))
	}

	targetPlayer := (playerID + 1) % 4
	msg, _ := network.NewPassMessage(
		playerID, targetPlayer, playerID, targetPlayer, cardMessages,
	)

	net.Send(msg)
	fmt.Printf("🃏 Enviou troca de cartas para jogador %d\n", targetPlayer)
}

func sendPlayMessage(net *network.Network, playerID int) {
	// Simular jogada de carta
	card := map[string]interface{}{
		"suit": "clubs",
		"rank": "2",
	}

	cardBytes, _ := json.Marshal(card)
	msg, _ := network.NewPlayMessage(
		playerID, -1, // broadcast
		playerID, json.RawMessage(cardBytes),
	)

	net.Send(msg)
	fmt.Printf("🎯 Enviou jogada de carta\n")
}

func listenForMessages(net *network.Network, playerID int) {
	for {
		select {
		case msg := <-net.Receive():
			handleReceivedMessage(msg, playerID)
		case token := <-net.TokenCh:
			fmt.Printf("🎯 Recebi o token! (HolderID: %d)\n", token.HolderID)
		}
	}
}

func handleReceivedMessage(msg network.Message, playerID int) {
	// Ignorar mensagens próprias que deram a volta
	if msg.From == playerID {
		return
	}

	switch msg.Type {
	case network.MessageDiscovery:
		var payload network.DiscoveryMessagePayload
		json.Unmarshal(msg.Payload, &payload)
		fmt.Printf("📡 Descoberta: %s (ID:%d, Ready:%v) de jogador %d\n",
			payload.PlayerName, payload.PlayerID, payload.IsReady, msg.From)

	case network.MessageState:
		var payload network.StateMessagePayload
		json.Unmarshal(msg.Payload, &payload)
		fmt.Printf("🎮 Estado do jogo recebido de jogador %d\n", msg.From)

	case network.MessagePass:
		var payload network.PassMessagePayload
		json.Unmarshal(msg.Payload, &payload)
		if msg.To == playerID {
			fmt.Printf("🃏 Recebi %d cartas do jogador %d\n",
				len(payload.Cards), msg.From)
		} else {
			fmt.Printf("🃏 Troca de cartas: jogador %d -> jogador %d (%d cartas)\n",
				msg.From, msg.To, len(payload.Cards))
		}

	case network.MessagePlay:
		var payload network.PlayMessagePayload
		json.Unmarshal(msg.Payload, &payload)
		fmt.Printf("🎯 Jogada do jogador %d\n", msg.From)

	case network.MessageToken:
		fmt.Printf("⚡ Token em trânsito (não é para mim)\n")
	}
}
