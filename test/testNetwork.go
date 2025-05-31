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
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run main.go <node_id>")
		os.Exit(1)
	}

	nodeID, err := strconv.Atoi(os.Args[1])
	if err != nil || nodeID < 0 || nodeID > 3 {
		fmt.Printf("ID inválido: %s\n", os.Args[1])
		os.Exit(1)
	}

	logger := log.New(os.Stdout, fmt.Sprintf("[Node %d] ", nodeID), log.LstdFlags|log.Lmicroseconds)

	logger.Printf("==== INICIANDO NÓ %d ====", nodeID)

	port := basePorts[nodeID]
	nextID := (nodeID + 1) % 4
	nextPort := basePorts[nextID]

	node := network.NewNode(nodeID, port, nextPort, logger)
	node.SetNodeIPs(nodeAddresses)

	if err := node.InitConnection(); err != nil {
		logger.Fatalf("Erro ao inicializar conexão: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		node.Listen()
	}()
	go func() {
		defer wg.Done()
		node.ProcessMessages()
	}()

	// Aguardar estabilidade mínima da rede
	time.Sleep(1 * time.Second)

	// Iniciar comportamento específico
	if nodeID == 0 {
		logger.Println("== NÓ HOST INICIANDO TESTES ==")
		go hostRoutine(node, logger)
	} else {
		logger.Println("== NÓ CLIENTE AGUARDANDO ==")
		go clientRoutine(node, logger)
	}

	// Capturar sinal de encerramento
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	logger.Println("Encerrando...")
	node.Close()
	wg.Wait()
	logger.Println("Nó finalizado.")
}

// Coordenação do host (nó 0)
func hostRoutine(node *network.Node, logger *log.Logger) {
	logger.Println("Host aguardando conexões...")

	time.Sleep(5 * time.Second) // Menor tempo para estabilizar

	// Início dos testes
	logger.Println("== Teste 1: Circulação do Bastão ==")
	if !testTokenCirculation(node, logger) {
		logger.Println("⚠️  Falha no Teste 1: Bastão não circulou corretamente.")
	}

	time.Sleep(1 * time.Second)

	logger.Println("== Teste 2: Mensagens Diretas ==")
	testDirectMessages(node, logger)

	time.Sleep(1 * time.Second)

	logger.Println("== Teste 3: Broadcast ==")
	testBroadcast(node, logger)

	time.Sleep(1 * time.Second)

	logger.Println("== Teste 4: Estatísticas ==")
	testStats(node, logger)

	time.Sleep(1 * time.Second)

	// Encerramento
	logger.Println("Encerrando testes, notificando os nós...")
	node.SendBroadcast("ENCERRAR_TESTES")
}

// Comportamento dos clientes (nós 1, 2, 3)
func clientRoutine(node *network.Node, logger *log.Logger) {
	heartbeat := time.NewTicker(5 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-heartbeat.C:
			state, hasToken := node.GetState()
			logger.Printf("Status: %s | Bastão: %v", state, hasToken)

			if hasToken {
				target := (node.ID + 2) % 4
				msg := fmt.Sprintf("Ping do nó %d para nó %d", node.ID, target)
				err := node.SendMessage(target, msg, network.MSG_DATA)
				if err != nil {
					logger.Printf("Erro ao enviar ping: %v", err)
				} else {
					logger.Printf("Ping enviado para nó %d", target)
				}
				node.PassToken((node.ID + 1) % 4)
			}
		}
	}
}

// Testes específicos

func testTokenCirculation(node *network.Node, logger *log.Logger) bool {
	_, hasToken := node.GetState()
	if hasToken {
		logger.Println("Host tem o bastão inicial. Passando para nó 1...")
		node.PassToken(1)
	} else {
		logger.Println("Host NÃO tem o bastão no início.")
	}

	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			logger.Println("⛔ TIMEOUT: Bastão não retornou ao host.")
			return false
		case <-ticker.C:
			_, hasToken := node.GetState()
			if hasToken {
				logger.Println("✅ Bastão retornou ao host com sucesso!")
				return true
			}
		}
	}
}

func testDirectMessages(node *network.Node, logger *log.Logger) {
	waitForToken(node, logger)

	for id := 1; id <= 3; id++ {
		msg := fmt.Sprintf("MSG direta do host para nó %d", id)
		if err := node.SendMessage(id, msg, network.MSG_DATA); err != nil {
			logger.Printf("Erro para nó %d: %v", id, err)
		} else {
			logger.Printf("Mensagem enviada para nó %d", id)
		}
		time.Sleep(500 * time.Millisecond)
	}
	node.PassToken(1)
}

func testBroadcast(node *network.Node, logger *log.Logger) {
	waitForToken(node, logger)

	msg := "BROADCAST do Host para todos os nós"
	if err := node.SendBroadcast(msg); err != nil {
		logger.Printf("Erro no broadcast: %v", err)
	} else {
		logger.Println("Broadcast enviado com sucesso.")
	}
	node.PassToken(1)
}

func testStats(node *network.Node, logger *log.Logger) {
	stats := node.GetStats()
	logger.Printf("== Estatísticas ==")
	logger.Printf("Enviadas: %d | Recebidas: %d | Bastões Passados: %d | Erros: %d",
		stats.MessagesSent, stats.MessagesReceived, stats.TokenPasses, stats.Errors)
}

func waitForToken(node *network.Node, logger *log.Logger) {
	logger.Println("Aguardando bastão...")
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			logger.Println("⛔ TIMEOUT esperando pelo bastão.")
			return
		case <-ticker.C:
			_, hasToken := node.GetState()
			if hasToken {
				logger.Println("✅ Bastão adquirido.")
				return
			}
		}
	}
}
