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
	0: "192.168.1.10", // Substitua pelos IPs reais das máquinas
	1: "192.168.1.11",
	2: "192.168.1.12",
	3: "192.168.1.13",
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
		fmt.Println("Onde node_id é 0, 1, 2 ou 3")
		os.Exit(1)
	}

	nodeID, err := strconv.Atoi(os.Args[1])
	if err != nil || nodeID < 0 || nodeID > 3 {
		fmt.Printf("ERRO: ID do nó deve ser 0, 1, 2 ou 3. Recebido: %s\n", os.Args[1])
		os.Exit(1)
	}

	// Configurar logger específico para o nó
	logger := log.New(os.Stdout, fmt.Sprintf("[Nó %d] ", nodeID), log.LstdFlags|log.Lmicroseconds)

	logger.Printf("=== INICIANDO TESTE DE REDE - NÓ %d ===", nodeID)
	logger.Printf("Endereços configurados:")
	for id, addr := range nodeAddresses {
		logger.Printf("  Nó %d: %s:%d", id, addr, basePorts[id])
	}

	// Criar nó com configuração de rede
	port := basePorts[nodeID]
	nextNodeID := (nodeID + 1) % 4
	nextPort := basePorts[nextNodeID]

	node := network.NewNode(nodeID, port, nextPort, logger)

	// Configurar IPs dos nós
	node.SetNodeIPs(nodeAddresses)

	// Inicializar conexão
	logger.Printf("Inicializando conexão na porta %d...", port)
	if err := node.InitConnection(); err != nil {
		logger.Fatalf("ERRO: Falha ao inicializar conexão: %v", err)
	}

	// Canal para coordenação
	var wg sync.WaitGroup

	// Iniciar gorrotinas do nó
	wg.Add(2)
	go func() {
		defer wg.Done()
		node.Listen()
	}()

	go func() {
		defer wg.Done()
		node.ProcessMessages()
	}()

	// Aguardar um pouco para todas as conexões se estabilizarem
	time.Sleep(2 * time.Second)

	// Se for o nó 0 (host), aguardar todos os nós e iniciar testes
	if nodeID == 0 {
		logger.Printf("=== NÓ HOST INICIANDO COORDENAÇÃO ===")
		go hostCoordination(node, logger)
	} else {
		logger.Printf("=== NÓ CLIENTE AGUARDANDO COORDENAÇÃO DO HOST ===")
		go clientBehavior(node, logger)
	}

	// Configurar captura de sinais para encerramento gracioso
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Aguardar sinal de encerramento
	<-sigChan
	logger.Printf("=== ENCERRANDO NÓ %d ===", nodeID)

	// Fechar nó
	if err := node.Close(); err != nil {
		logger.Printf("ERRO ao fechar nó: %v", err)
	}

	logger.Printf("=== NÓ %d ENCERRADO ===", nodeID)
}

// Coordenação do host (nó 0)
func hostCoordination(node *network.Node, logger *log.Logger) {
	logger.Printf("HOST: Aguardando 10 segundos para todos os nós se conectarem...")
	time.Sleep(10 * time.Second)

	logger.Printf("HOST: Iniciando testes de rede...")

	// Teste 1: Verificar se o bastão está circulando
	logger.Printf("=== TESTE 1: CIRCULAÇÃO DO BASTÃO ===")
	testTokenCirculation(node, logger)

	time.Sleep(5 * time.Second)

	// Teste 2: Envio de mensagens dirigidas
	logger.Printf("=== TESTE 2: MENSAGENS DIRIGIDAS ===")
	testDirectMessages(node, logger)

	time.Sleep(5 * time.Second)

	// Teste 3: Mensagens de broadcast
	logger.Printf("=== TESTE 3: MENSAGENS DE BROADCAST ===")
	testBroadcastMessages(node, logger)

	time.Sleep(5 * time.Second)

	// Teste 4: Estatísticas da rede
	logger.Printf("=== TESTE 4: ESTATÍSTICAS DA REDE ===")
	testNetworkStats(node, logger)

	time.Sleep(5 * time.Second)

	// Teste 5: Simulação contínua
	logger.Printf("=== TESTE 5: SIMULAÇÃO CONTÍNUA ===")
	logger.Printf("HOST: Iniciando simulação contínua por 30 segundos...")

	go node.Simulate()
	time.Sleep(30 * time.Second)

	logger.Printf("=== TODOS OS TESTES CONCLUÍDOS ===")

	// Enviar sinal de encerramento para todos os nós
	if err := node.SendBroadcast("ENCERRAR_TESTES"); err != nil {
		logger.Printf("ERRO ao enviar sinal de encerramento: %v", err)
	}
}

// Comportamento dos clientes (nós 1, 2, 3)
func clientBehavior(node *network.Node, logger *log.Logger) {
	// Clientes respondem a comandos do host e participam dos testes
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Enviar heartbeat periodicamente
			state, hasToken := node.GetState()
			logger.Printf("CLIENTE: Estado=%s, Bastão=%v", state, hasToken)

			// Se tiver o bastão, enviar uma mensagem de teste ocasionalmente
			if hasToken {
				target := (node.ID + 2) % 4 // Enviar para nó dois à frente
				if err := node.SendMessage(target,
					fmt.Sprintf("Teste do nó %d", node.ID),
					network.MSG_DATA); err != nil {
					logger.Printf("ERRO ao enviar mensagem de teste: %v", err)
				}
			}
		}
	}
}

// Testes específicos

func testTokenCirculation(node *network.Node, logger *log.Logger) {
	logger.Printf("Testando circulação do bastão...")

	_, hasToken := node.GetState()
	if hasToken {
		logger.Printf("HOST tem o bastão, passando para nó 1...")
		if err := node.PassToken(1); err != nil {
			logger.Printf("ERRO ao passar bastão: %v", err)
		}
	}

	// Aguardar o bastão voltar
	timeout := time.After(15 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			logger.Printf("TIMEOUT: Bastão não retornou em 15 segundos")
			return
		case <-ticker.C:
			_, hasToken := node.GetState()
			if hasToken {
				logger.Printf("SUCESSO: Bastão retornou ao host!")
				return
			}
		}
	}
}

func testDirectMessages(node *network.Node, logger *log.Logger) {
	logger.Printf("Testando mensagens dirigidas...")

	// Aguardar ter o bastão
	for {
		_, hasToken := node.GetState()
		if hasToken {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Enviar mensagem para cada nó
	for targetID := 1; targetID < 4; targetID++ {
		message := fmt.Sprintf("Mensagem de teste do host para nó %d", targetID)
		if err := node.SendMessage(targetID, message, network.MSG_DATA); err != nil {
			logger.Printf("ERRO ao enviar mensagem para nó %d: %v", targetID, err)
		} else {
			logger.Printf("Mensagem enviada para nó %d", targetID)
		}
		time.Sleep(1 * time.Second)
	}
}

func testBroadcastMessages(node *network.Node, logger *log.Logger) {
	logger.Printf("Testando mensagens de broadcast...")

	// Aguardar ter o bastão
	for {
		_, hasToken := node.GetState()
		if hasToken {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	message := "BROADCAST: Mensagem para todos os nós!"
	if err := node.SendBroadcast(message); err != nil {
		logger.Printf("ERRO ao enviar broadcast: %v", err)
	} else {
		logger.Printf("Broadcast enviado com sucesso")
	}
}

func testNetworkStats(node *network.Node, logger *log.Logger) {
	logger.Printf("Coletando estatísticas da rede...")

	stats := node.GetStats()
	logger.Printf("ESTATÍSTICAS DO NÓ:")
	logger.Printf("  Mensagens Enviadas: %d", stats.MessagesSent)
	logger.Printf("  Mensagens Recebidas: %d", stats.MessagesReceived)
	logger.Printf("  Passagens de Bastão: %d", stats.TokenPasses)
	logger.Printf("  Erros: %d", stats.Errors)
	logger.Printf("  Última Atividade: %s", stats.LastActivity.Format("15:04:05"))
}
