package network

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Tipos de mensagem no protocolo
const (
	MSG_TOKEN     = "TOKEN"     // Bastão
	MSG_DATA      = "DATA"      // Mensagem de dados
	MSG_GAME      = "GAME"      // Mensagem específica do jogo
	MSG_HEARTBEAT = "HEARTBEAT" // Para manter a rede ativa
	MSG_BROADCAST = "BROADCAST" // Mensagem de broadcast
	MSG_ERROR     = "ERROR"     // Mensagem de erro
	MSG_ACK       = "ACK"       // Confirmação de recebimento
)

// Estados do nó
const (
	NODE_IDLE    = "IDLE"
	NODE_ACTIVE  = "ACTIVE"
	NODE_SENDING = "SENDING"
	NODE_WAITING = "WAITING"
	NODE_ERROR   = "ERROR"
)

// Códigos de erro
const (
	ERR_NO_TOKEN        = "NO_TOKEN"
	ERR_INVALID_MESSAGE = "INVALID_MESSAGE"
	ERR_NETWORK_ERROR   = "NETWORK_ERROR"
	ERR_TIMEOUT         = "TIMEOUT"
	ERR_INVALID_NODE    = "INVALID_NODE"
)

// Estrutura da mensagem
type Message struct {
	Type      string      `json:"type"`
	From      int         `json:"from"`
	To        int         `json:"to"`
	Content   interface{} `json:"content"`
	Hops      int         `json:"hops"`
	MessageID string      `json:"message_id"`
	Timestamp int64       `json:"timestamp"`
	Priority  int         `json:"priority"` // 0=baixa, 1=normal, 2=alta
}

// Estrutura de erro personalizada
type NetworkError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	NodeID  int    `json:"node_id"`
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("Node %d [%s]: %s", e.NodeID, e.Code, e.Message)
}

// Estatísticas do nó
type NodeStats struct {
	MessagesSent     int64
	MessagesReceived int64
	TokenPasses      int64
	Errors           int64
	LastActivity     time.Time
}

// Mapeamento global de IPs dos nós (será configurado na inicialização)
var nodeIPMap = map[int]string{
	0: "localhost", // Padrão - será atualizado pela main
	1: "localhost",
	2: "localhost",
	3: "localhost",
}

// Estrutura do nó da rede
type Node struct {
	ID         int
	Port       int
	NextPort   int
	Conn       *net.UDPConn
	HasToken   bool
	State      string
	Messages   chan Message
	Quit       chan bool
	Stats      *NodeStats
	Logger     *log.Logger
	mu         sync.RWMutex
	messageLog map[string]bool // Para evitar loops infinitos
	maxHops    int
	NodeIPs    map[int]string // Mapeamento de IPs dos nós
}

// Função para obter IP do nó
func getNodeIP(nodeID int) string {
	if ip, exists := nodeIPMap[nodeID]; exists {
		return ip
	}
	return "localhost" // Fallback
}

// Função para configurar o mapeamento de IPs
func SetNodeIPs(ips map[int]string) {
	for id, ip := range ips {
		nodeIPMap[id] = ip
	}
}

// Criar um novo nó
func NewNode(id, port, nextPort int, logger *log.Logger) *Node {
	if logger == nil {
		logger = log.New(log.Writer(), fmt.Sprintf("[Node %d] ", id), log.LstdFlags|log.Lshortfile)
	}

	return &Node{
		ID:         id,
		Port:       port,
		NextPort:   nextPort,
		HasToken:   id == 0, // Nó 0 começa com o bastão
		State:      NODE_IDLE,
		Messages:   make(chan Message, 100),
		Quit:       make(chan bool),
		Stats:      &NodeStats{LastActivity: time.Now()},
		Logger:     logger,
		messageLog: make(map[string]bool),
		maxHops:    4,
		NodeIPs:    make(map[int]string),
	}
}

// Método para obter IP de um nó específico
func (n *Node) GetNodeIP(nodeID int) string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if ip, exists := n.NodeIPs[nodeID]; exists {
		return ip
	}

	// Fallback para o mapeamento global
	return getNodeIP(nodeID)
}

// Método para configurar IPs no nó
func (n *Node) SetNodeIPs(ips map[int]string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.NodeIPs = make(map[int]string)
	for id, ip := range ips {
		n.NodeIPs[id] = ip
	}

	// Também atualiza o mapeamento global
	SetNodeIPs(ips)

	n.Logger.Printf("INFO: IPs dos nós configurados: %v", n.NodeIPs)
}

// Inicializar conexão UDP
func (n *Node) InitConnection() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", n.Port))
	if err != nil {
		netErr := &NetworkError{
			Code:    ERR_NETWORK_ERROR,
			Message: fmt.Sprintf("Falha ao resolver endereço UDP: %v", err),
			NodeID:  n.ID,
		}
		n.Logger.Printf("ERRO: %s", netErr.Error())
		return netErr
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		netErr := &NetworkError{
			Code:    ERR_NETWORK_ERROR,
			Message: fmt.Sprintf("Falha ao iniciar listener UDP: %v", err),
			NodeID:  n.ID,
		}
		n.Logger.Printf("ERRO: %s", netErr.Error())
		return netErr
	}

	n.Conn = conn
	n.State = NODE_ACTIVE
	n.Logger.Printf("INFO: Nó iniciado na porta %d, próximo nó na porta %d", n.Port, n.NextPort)

	if n.HasToken {
		n.Logger.Printf("INFO: Nó iniciado com o bastão")
	}

	return nil
}

// Enviar mensagem para o próximo nó
func (n *Node) SendToNext(msg Message) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	msg.Timestamp = time.Now().UnixNano()

	data, err := json.Marshal(msg)
	if err != nil {
		netErr := &NetworkError{
			Code:    ERR_INVALID_MESSAGE,
			Message: fmt.Sprintf("Falha ao serializar mensagem: %v", err),
			NodeID:  n.ID,
		}
		n.Stats.Errors++
		n.Logger.Printf("ERRO: %s", netErr.Error())
		return netErr
	}

	// Por esta (você precisará passar o mapa de endereços):
	nextNodeID := (n.ID + 1) % 4
	nextNodeIP := getNodeIP(nextNodeID)
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", nextNodeIP, n.NextPort))
	if err != nil {
		netErr := &NetworkError{
			Code:    ERR_NETWORK_ERROR,
			Message: fmt.Sprintf("Falha ao resolver endereço do próximo nó: %v", err),
			NodeID:  n.ID,
		}
		n.Stats.Errors++
		n.Logger.Printf("ERRO: %s", netErr.Error())
		return netErr
	}

	_, err = n.Conn.WriteToUDP(data, addr)
	if err != nil {
		netErr := &NetworkError{
			Code:    ERR_NETWORK_ERROR,
			Message: fmt.Sprintf("Falha ao enviar dados UDP: %v", err),
			NodeID:  n.ID,
		}
		n.Stats.Errors++
		n.Logger.Printf("ERRO: %s", netErr.Error())
		return netErr
	}

	n.Stats.MessagesSent++
	n.Stats.LastActivity = time.Now()
	n.Logger.Printf("DEBUG: Mensagem enviada para porta %d - Tipo: %s, ID: %s",
		n.NextPort, msg.Type, msg.MessageID)

	return nil
}

// Escutar mensagens
func (n *Node) Listen() {
	buffer := make([]byte, 2048) // Aumentado para mensagens maiores
	n.Logger.Printf("INFO: Iniciando escuta de mensagens")

	for {
		select {
		case <-n.Quit:
			n.Logger.Printf("INFO: Parando escuta de mensagens")
			return
		default:
			n.Conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			length, remoteAddr, err := n.Conn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // Timeout esperado
				}
				n.mu.Lock()
				n.Stats.Errors++
				n.mu.Unlock()
				n.Logger.Printf("ERRO: Falha ao ler UDP: %v", err)
				continue
			}

			var msg Message
			if err := json.Unmarshal(buffer[:length], &msg); err != nil {
				n.mu.Lock()
				n.Stats.Errors++
				n.mu.Unlock()
				n.Logger.Printf("ERRO: Falha ao decodificar mensagem de %s: %v", remoteAddr, err)
				continue
			}

			n.mu.Lock()
			n.Stats.MessagesReceived++
			n.Stats.LastActivity = time.Now()
			n.mu.Unlock()

			n.Logger.Printf("DEBUG: Mensagem recebida - Tipo: %s, De: %d, Para: %d, Hops: %d, ID: %s",
				msg.Type, msg.From, msg.To, msg.Hops, msg.MessageID)

			select {
			case n.Messages <- msg:
			default:
				n.Logger.Printf("AVISO: Canal de mensagens cheio, descartando mensagem %s", msg.MessageID)
			}
		}
	}
}

// Processar mensagens recebidas
func (n *Node) ProcessMessages() {
	n.Logger.Printf("INFO: Iniciando processamento de mensagens")

	for {
		select {
		case <-n.Quit:
			n.Logger.Printf("INFO: Parando processamento de mensagens")
			return
		case msg := <-n.Messages:
			if err := n.handleMessage(msg); err != nil {
				n.Logger.Printf("ERRO: Falha ao processar mensagem %s: %v", msg.MessageID, err)
			}
		}
	}
}

// Lidar com mensagens recebidas
func (n *Node) handleMessage(msg Message) error {
	// Verificar se já processamos esta mensagem (evitar loops)
	n.mu.Lock()
	if n.messageLog[msg.MessageID] {
		n.mu.Unlock()
		n.Logger.Printf("DEBUG: Mensagem %s já processada, ignorando", msg.MessageID)
		return nil
	}
	n.messageLog[msg.MessageID] = true
	n.mu.Unlock()

	// Incrementa o número de saltos
	msg.Hops++

	// Verificar se excedeu o máximo de saltos
	if msg.Hops > n.maxHops {
		n.Logger.Printf("AVISO: Mensagem %s excedeu máximo de saltos (%d), descartando",
			msg.MessageID, n.maxHops)
		return nil
	}

	switch msg.Type {
	case MSG_TOKEN:
		return n.handleTokenMessage(msg)
	case MSG_DATA, MSG_GAME:
		return n.handleDataMessage(msg)
	case MSG_BROADCAST:
		return n.handleBroadcastMessage(msg)
	case MSG_HEARTBEAT:
		return n.handleHeartbeatMessage(msg)
	case MSG_ERROR:
		return n.handleErrorMessage(msg)
	default:
		return &NetworkError{
			Code:    ERR_INVALID_MESSAGE,
			Message: fmt.Sprintf("Tipo de mensagem desconhecido: %s", msg.Type),
			NodeID:  n.ID,
		}
	}
}

// Lidar com mensagem de bastão
func (n *Node) handleTokenMessage(msg Message) error {
	if msg.To == n.ID || msg.To == -1 {
		n.mu.Lock()
		n.HasToken = true
		n.State = NODE_ACTIVE
		n.Stats.TokenPasses++
		n.mu.Unlock()

		n.Logger.Printf("INFO: Bastão recebido de nó %d", msg.From)

		// Se não é o destinatário original e ainda não completou a volta, continua passando
		if msg.To != n.ID && msg.Hops < n.maxHops {
			return n.forwardMessage(msg)
		}
	} else {
		return n.forwardMessage(msg)
	}
	return nil
}

// Lidar com mensagem de dados
func (n *Node) handleDataMessage(msg Message) error {
	if msg.To == n.ID || msg.To == -1 {
		n.Logger.Printf("INFO: Mensagem recebida de nó %d: %v", msg.From, msg.Content)
	}

	// Continua passando se não é do próprio nó ou se acabou de ser enviada
	if msg.From != n.ID {
		return n.forwardMessage(msg)
	}

	return nil
}

// Lidar com mensagem de broadcast
func (n *Node) handleBroadcastMessage(msg Message) error {
	n.Logger.Printf("INFO: Broadcast recebido de nó %d: %v", msg.From, msg.Content)

	if msg.From != n.ID {
		return n.forwardMessage(msg)
	}

	return nil
}

// Lidar com mensagem de heartbeat
func (n *Node) handleHeartbeatMessage(msg Message) error {
	n.Logger.Printf("DEBUG: Heartbeat recebido de nó %d", msg.From)

	if msg.From != n.ID {
		return n.forwardMessage(msg)
	}

	return nil
}

// Lidar com mensagem de erro
func (n *Node) handleErrorMessage(msg Message) error {
	if errorData, ok := msg.Content.(map[string]interface{}); ok {
		n.Logger.Printf("ERRO: Erro recebido de nó %d: %v", msg.From, errorData)
	}

	if msg.From != n.ID {
		return n.forwardMessage(msg)
	}

	return nil
}

// Encaminhar mensagem para o próximo nó
func (n *Node) forwardMessage(msg Message) error {
	if err := n.SendToNext(msg); err != nil {
		return fmt.Errorf("falha ao encaminhar mensagem %s: %w", msg.MessageID, err)
	}

	n.Logger.Printf("DEBUG: Mensagem %s encaminhada", msg.MessageID)
	return nil
}

// Enviar uma mensagem (apenas se tiver o bastão)
func (n *Node) SendMessage(to int, content interface{}, msgType string) error {
	n.mu.RLock()
	hasToken := n.HasToken
	n.mu.RUnlock()

	if !hasToken {
		return &NetworkError{
			Code:    ERR_NO_TOKEN,
			Message: "Não é possível enviar mensagem sem o bastão",
			NodeID:  n.ID,
		}
	}

	if to < -1 || to >= 4 {
		return &NetworkError{
			Code:    ERR_INVALID_NODE,
			Message: fmt.Sprintf("ID de nó inválido: %d", to),
			NodeID:  n.ID,
		}
	}

	msg := Message{
		Type:      msgType,
		From:      n.ID,
		To:        to,
		Content:   content,
		Hops:      0,
		MessageID: fmt.Sprintf("%d_%d", n.ID, time.Now().UnixNano()),
		Priority:  1, // Normal por padrão
	}

	if err := n.SendToNext(msg); err != nil {
		return fmt.Errorf("falha ao enviar mensagem: %w", err)
	}

	n.Logger.Printf("INFO: Mensagem enviada para nó %d - Tipo: %s", to, msgType)
	return nil
}

// Enviar broadcast
func (n *Node) SendBroadcast(content interface{}) error {
	return n.SendMessage(-1, content, MSG_BROADCAST)
}

// Passar o bastão para o próximo nó
func (n *Node) PassToken(to int) error {
	n.mu.Lock()
	hasToken := n.HasToken
	n.mu.Unlock()

	if !hasToken {
		return &NetworkError{
			Code:    ERR_NO_TOKEN,
			Message: "Não é possível passar bastão que não possui",
			NodeID:  n.ID,
		}
	}

	if to < 0 || to >= 4 {
		return &NetworkError{
			Code:    ERR_INVALID_NODE,
			Message: fmt.Sprintf("ID de nó inválido para passar bastão: %d", to),
			NodeID:  n.ID,
		}
	}

	n.mu.Lock()
	n.HasToken = false
	n.State = NODE_WAITING
	n.mu.Unlock()

	msg := Message{
		Type:      MSG_TOKEN,
		From:      n.ID,
		To:        to,
		Hops:      0,
		MessageID: fmt.Sprintf("token_%d_%d", n.ID, time.Now().UnixNano()),
		Priority:  2, // Alta prioridade para bastão
	}

	if err := n.SendToNext(msg); err != nil {
		// Recupera o bastão se houve erro
		n.mu.Lock()
		n.HasToken = true
		n.State = NODE_ACTIVE
		n.mu.Unlock()
		return fmt.Errorf("falha ao passar bastão: %w", err)
	}

	n.Logger.Printf("INFO: Bastão passado para nó %d", to)
	return nil
}

// Obter estatísticas do nó
func (n *Node) GetStats() NodeStats {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return *n.Stats
}

// Obter estado atual do nó
func (n *Node) GetState() (string, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.State, n.HasToken
}

// Limpar log de mensagens antigas
func (n *Node) CleanMessageLog() {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Limpa mensagens mais antigas que 1 minuto
	n.messageLog = make(map[string]bool)
	n.Logger.Printf("DEBUG: Log de mensagens limpo")
}

// Simular atividade do nó (para testes)
func (n *Node) Simulate() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	cleanTicker := time.NewTicker(1 * time.Minute)
	defer cleanTicker.Stop()

	n.Logger.Printf("INFO: Iniciando simulação")

	for {
		select {
		case <-n.Quit:
			n.Logger.Printf("INFO: Parando simulação")
			return
		case <-ticker.C:
			n.mu.RLock()
			hasToken := n.HasToken
			n.mu.RUnlock()

			if hasToken {
				// Simula envio de uma mensagem
				target := (n.ID + 1) % 4
				if err := n.SendMessage(target, fmt.Sprintf("Olá do nó %d", n.ID), MSG_DATA); err != nil {
					n.Logger.Printf("ERRO: Falha ao enviar mensagem de simulação: %v", err)
				}

				// Passa o bastão para o próximo nó
				time.Sleep(500 * time.Millisecond)
				nextNode := (n.ID + 1) % 4
				if err := n.PassToken(nextNode); err != nil {
					n.Logger.Printf("ERRO: Falha ao passar bastão na simulação: %v", err)
				}
			}
		case <-cleanTicker.C:
			n.CleanMessageLog()
		}
	}
}

// Fechar conexões
func (n *Node) Close() error {
	n.Logger.Printf("INFO: Fechando nó")

	close(n.Quit)

	if n.Conn != nil {
		if err := n.Conn.Close(); err != nil {
			return fmt.Errorf("erro ao fechar conexão: %w", err)
		}
	}

	n.Logger.Printf("INFO: Nó fechado com sucesso")
	return nil
}
