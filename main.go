package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"copas/controller"
	"copas/network"
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
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run main.go <node_id>")
		fmt.Println("  node_id: 0, 1, 2 ou 3")
		fmt.Println("  Nó 0 é o host que inicia o jogo")
		os.Exit(1)
	}

	nodeID, err := strconv.Atoi(os.Args[1])
	if err != nil || nodeID < 0 || nodeID > 3 {
		fmt.Printf("❌ ID de nó inválido: %s\n", os.Args[1])
		fmt.Println("   Use: 0, 1, 2 ou 3")
		os.Exit(1)
	}

	// Configura logger
	logger := log.New(os.Stdout, fmt.Sprintf("[Nó %d] ", nodeID), log.LstdFlags|log.Lmicroseconds)

	// Banner de início
	printBanner(nodeID, logger)

	// Configuração da rede
	port := basePorts[nodeID]
	nextID := (nodeID + 1) % 4
	nextPort := basePorts[nextID]

	// Cria o nó de rede
	node := network.NewNode(nodeID, port, nextPort, logger)
	node.SetNodeIPs(nodeAddresses)

	// Inicializa conexão
	if err := node.InitConnection(); err != nil {
		logger.Fatalf("❌ Erro ao inicializar conexão: %v", err)
	}

	// Configura controlador do jogo
	isHost := nodeID == 0
	gameController := controller.NewGameController(node, logger, isHost)

	// Inicia goroutines da rede
	var wg sync.WaitGroup
	wg.Add(3) // Listen, ProcessMessages, ProcessGameMessages

	// Goroutine para escutar mensagens da rede
	go func() {
		defer wg.Done()
		node.Listen()
	}()

	// Goroutine para processar mensagens da rede
	go func() {
		defer wg.Done()
		for {
			select {
			case <-time.After(100 * time.Millisecond):
				// Verifica se deve encerrar
				if shouldExit(node) {
					return
				}
			}
		}
	}()

	// Goroutine para processar mensagens específicas do jogo
	go func() {
		defer wg.Done()
		processGameMessages(node, gameController, logger)
	}()

	// Aguarda estabilização da rede
	logger.Println("⏳ Aguardando estabilização da rede...")
	time.Sleep(2 * time.Second)

	// Inicia o controlador do jogo
	go gameController.Start()

	// Captura sinais de encerramento
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Println("✅ Sistema iniciado. Pressione Ctrl+C para encerrar.")

	// Aguarda sinal de encerramento
	<-sigChan

	// Encerramento gracioso
	logger.Println("🔄 Encerrando sistema...")

	// Notifica outros nós sobre o encerramento
	if _, hasToken := node.GetState(); hasToken {
		node.SendBroadcast("PLAYER_DISCONNECTED")
		node.PassToken((nodeID + 1) % 4)
	}

	// Fecha conexões
	node.Close()

	// Aguarda finalização das goroutines com timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Println("✅ Sistema encerrado com sucesso.")
	case <-time.After(5 * time.Second):
		logger.Println("⚠️ Timeout no encerramento, forçando saída.")
	}
}

func printBanner(nodeID int, logger *log.Logger) {
	fmt.Println("\n" + "==================================================================")
	fmt.Println("🎮           JOGO DE COPAS - REDE EM ANEL")
	fmt.Println("===================================================================")
	fmt.Printf("🖥️  Nó: %d\n", nodeID)
	if nodeID == 0 {
		fmt.Println("👑 Tipo: HOST (inicia o jogo)")
	} else {
		fmt.Println("👤 Tipo: CLIENTE")
	}
	fmt.Printf("🌐 Porta: %d\n", basePorts[nodeID])
	fmt.Printf("📡 Próximo nó: %d (porta %d)\n", (nodeID+1)%4, basePorts[(nodeID+1)%4])
	fmt.Println("====================================================================")
	fmt.Println()

	logger.Printf("🚀 Iniciando Nó %d...", nodeID)
}

func processGameMessages(node *network.Node, gameController *controller.GameController, logger *log.Logger) {
	for {
		select {
		case msg := <-node.Messages:
			// Processa apenas mensagens do tipo GAME
			if msg.Type == network.MSG_GAME || msg.Type == network.MSG_BROADCAST {
				gameController.ProcessGameMessage(msg.Content)
			}
		case <-time.After(100 * time.Millisecond):
			// Timeout para não bloquear indefinidamente
			continue
		}
	}
}

func shouldExit(node *network.Node) bool {
	// Implementa lógica para determinar se deve encerrar
	// Por exemplo, se perdeu conectividade por muito tempo
	return false
}
