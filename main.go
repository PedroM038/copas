package main

import (
	"bufio"
	"copas/controller"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Configuração da rede - IPs das máquinas
var nodeAddresses = map[int]string{
	0: "10.254.223.41", // Substitua pelos IPs reais das máquinas
	1: "10.254.223.43",
	2: "10.254.223.44",
	3: "10.254.223.46",
}

// Portas base para cada nó
var basePorts = map[int]int{
	0: 8000,
	1: 8001,
	2: 8002,
	3: 8003,
}

func main() {
	// Verifica argumentos
	if len(os.Args) < 3 {
		fmt.Println("Uso: go run main.go <node_id> <nome_jogador>")
		fmt.Println("  node_id: 0, 1, 2 ou 3")
		fmt.Println("  nome_jogador: Nome do jogador (sem espaços)")
		fmt.Println("  Nó 0 é o host que inicia o jogo")
		fmt.Println("")
		fmt.Println("Exemplos:")
		fmt.Println("  go run main.go 0 Pedro")
		fmt.Println("  go run main.go 1 Maria")
		fmt.Println("  go run main.go 2 João")
		fmt.Println("  go run main.go 3 Ana")
		os.Exit(1)
	}

	nodeID, err := strconv.Atoi(os.Args[1])
	if err != nil || nodeID < 0 || nodeID > 3 {
		fmt.Printf("❌ ID de nó inválido: %s\n", os.Args[1])
		fmt.Println("   Use: 0, 1, 2 ou 3")
		os.Exit(1)
	}

	// Obtém o nome do jogador do argumento
	playerName := strings.TrimSpace(os.Args[2])
	if playerName == "" {
		fmt.Println("❌ Nome do jogador não pode estar vazio")
		os.Exit(1)
	}

	// Validação simples do nome (apenas letras, números e alguns caracteres especiais)
	if len(playerName) > 20 {
		fmt.Println("❌ Nome do jogador muito longo (máximo 20 caracteres)")
		os.Exit(1)
	}

	// Verifica se é o host (nó 0)
	isHost := nodeID == 0

	// Constrói endereços da rede
	listenAddr := fmt.Sprintf("%s:%d", nodeAddresses[nodeID], basePorts[nodeID])
	nextNodeID := (nodeID + 1) % 4
	nextNodeAddr := fmt.Sprintf("%s:%d", nodeAddresses[nextNodeID], basePorts[nextNodeID])

	fmt.Println("🎮 =================================")
	fmt.Println("🎯 JOGO COPAS - REDE EM ANEL")
	fmt.Println("🎮 =================================")
	fmt.Printf("🔗 Nó ID: %d\n", nodeID)
	fmt.Printf("👤 Jogador: %s\n", playerName)
	fmt.Printf("🌐 Endereço local: %s\n", listenAddr)
	fmt.Printf("📡 Próximo nó: %s\n", nextNodeAddr)
	if isHost {
		fmt.Println("👑 ESTE É O NÓ HOST")
	}
	fmt.Println("🎮 =================================")

	// Cria o controller do jogo
	gameController := controller.NewGameController(nodeID, playerName, isHost)

	// Define callbacks
	gameController.SetOnPlayerJoin(func(playerID int, playerName string) {
		fmt.Printf("🎉 %s entrou no jogo!\n", playerName)
	})

	gameController.SetOnGameEnd(func(winnerID int, winnerName string) {
		fmt.Printf("\n🏆 JOGO FINALIZADO!\n")
		fmt.Printf("🎊 Vencedor: %s (ID: %d)\n", winnerName, winnerID)
		fmt.Println("Pressione Ctrl+C para sair")
	})

	gameController.SetOnError(func(err error) {
		fmt.Printf("❌ Erro no jogo: %v\n", err)
	})

	// Inicia o controller
	err = gameController.Start(listenAddr, nextNodeAddr)
	if err != nil {
		fmt.Printf("❌ Erro ao iniciar controller: %v\n", err)
		os.Exit(1)
	}

	// Configura tratamento de sinais para encerramento limpo
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Inicia loop principal do jogo
	go gameLoop(gameController)

	// Aguarda sinal de encerramento
	<-sigChan
	fmt.Println("\n🛑 Encerrando jogo...")
	gameController.Stop()
	fmt.Println("👋 Até logo!")
}

// Loop principal do jogo - interface de linha de comando
func gameLoop(gc *controller.GameController) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		// Verifica se é a vez do jogador
		if !gc.IsMyTurn() {
			// Se não é a vez, aguarda um pouco e verifica novamente
			time.Sleep(1 * time.Second)
			continue
		}

		game := gc.GetGame()
		if game == nil || game.GameOver {
			time.Sleep(1 * time.Second)
			continue
		}

		// Obtém jogadas válidas
		validPlays := game.GetValidPlays(gc.GetNodeID())
		if len(validPlays) == 0 {
			fmt.Println("❌ Nenhuma jogada válida disponível")
			time.Sleep(1 * time.Second)
			continue
		}

		// Aguarda input do usuário
		fmt.Print("👉 Digite o número da carta (1-13): ")
		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())

			// Verifica comandos especiais
			switch input {
			case "quit", "exit", "q":
				fmt.Println("🛑 Saindo do jogo...")
				os.Exit(0)
			case "help", "h":
				showHelp()
				continue
			case "status", "s":
				showDetailedStatus(gc)
				continue
			}

			// Tenta converter para número
			cardIndex, err := strconv.Atoi(input)
			if err != nil {
				fmt.Printf("❌ Entrada inválida: %s\n", input)
				continue
			}

			// Verifica se o índice é válido
			hand, err := game.GetPlayerHand(gc.GetNodeID())
			if err != nil {
				fmt.Printf("❌ Erro ao obter mão: %v\n", err)
				continue
			}

			if cardIndex < 1 || cardIndex > len(hand) {
				fmt.Printf("❌ Número inválido. Use 1-%d\n", len(hand))
				continue
			}

			// Obtém a carta selecionada
			selectedCard := hand[cardIndex-1]

			// Verifica se a carta é válida
			isValid := false
			for _, validCard := range validPlays {
				if validCard.Naipe == selectedCard.Naipe && validCard.Valor == selectedCard.Valor {
					isValid = true
					break
				}
			}

			if !isValid {
				fmt.Printf("❌ Carta inválida: %s\n", selectedCard.String())
				fmt.Println("💡 Jogadas válidas disponíveis:")
				for i, card := range validPlays {
					fmt.Printf("   %d. %s\n", i+1, card.String())
				}
				continue
			}

			// Joga a carta
			err = gc.PlayCard(selectedCard)
			if err != nil {
				fmt.Printf("❌ Erro ao jogar carta: %v\n", err)
				continue
			}

			fmt.Printf("✅ Você jogou: %s\n", selectedCard.String())
		}
	}
}

// Mostra ajuda
func showHelp() {
	fmt.Println("\n📚 COMANDOS DISPONÍVEIS:")
	fmt.Println("  1-13    : Jogar carta (número correspondente na sua mão)")
	fmt.Println("  help, h : Mostrar esta ajuda")
	fmt.Println("  status, s : Mostrar status detalhado")
	fmt.Println("  quit, q : Sair do jogo")
	fmt.Println("\n🎯 REGRAS RÁPIDAS:")
	fmt.Println("  • Evite cartas de Copas (♥) e Dama de Espadas (♠Q)")
	fmt.Println("  • Copas (♥): 1 ponto cada")
	fmt.Println("  • Dama de Espadas (♠Q): 13 pontos")
	fmt.Println("  • Menor pontuação vence")
	fmt.Println("  • Deve seguir o naipe se tiver")
	fmt.Println()
}

// Mostra status detalhado
func showDetailedStatus(gc *controller.GameController) {
	fmt.Println("\n📊 STATUS DETALHADO:")
	fmt.Printf("🎮 Estado do jogo: %s\n", gc.GetGameState())

	players := gc.GetPlayers()
	fmt.Println("👥 Jogadores:")
	for id, name := range players {
		fmt.Printf("   %d: %s\n", id, name)
	}

	game := gc.GetGame()
	if game != nil {
		fmt.Printf("🔄 Rodada atual: %d/13\n", len(game.CompletedTricks)+1)
		fmt.Printf("💖 Copas quebradas: %v\n", game.HeartsBreoken)
		fmt.Printf("🎯 Jogador atual: %s\n", players[game.CurrentPlayer])

		if game.GameOver {
			fmt.Printf("🏆 Vencedor: %s\n", players[game.WinnerID])
		}
	}
	fmt.Println()
}
