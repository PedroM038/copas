package network

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Estados do nó na rede
const (
	NODE_STATE_DISCONNECTED = "DISCONNECTED"
	NODE_STATE_CONNECTING   = "CONNECTING"
	NODE_STATE_CONNECTED    = "CONNECTED"
	NODE_STATE_ERROR        = "ERROR"
)

// Configurações da rede
const (
	MAX_HOPS           = 8
	HEARTBEAT_INTERVAL = 5 * time.Second
	TOKEN_TIMEOUT      = 10 * time.Second
	MESSAGE_TIMEOUT    = 3 * time.Second
	BUFFER_SIZE        = 4096
	RING_SIZE          = 4
)

// Estatísticas do nó
type NodeStatistics struct {
	MessagesSent      uint64
	MessagesReceived  uint64
	MessagesForwarded uint64
	MessagesDropped   uint64
	ErrorsCount       uint64
	TokenReceived     uint64
	TokenPassed       uint64
	Uptime            time.Time
	LastActivity      time.Time
}

// Configuração do nó
type NodeConfig struct {
	ID           int
	ListenAddr   string
	NextNodeAddr string
	RingSize     int
	UseColors    bool
}

// Nó da rede em anel
type RingNode struct {
	config       *NodeConfig
	state        string
	conn         *net.UDPConn
	nextNodeAddr *net.UDPAddr
	logger       *NetworkLogger
	tokenCtrl    *TokenController
	stats        *NodeStatistics

	// Canais de comunicação
	incomingChan chan *Message
	outgoingChan chan *Message
	controlChan  chan string

	// Controle de execução
	running bool
	mutex   sync.RWMutex
	wg      sync.WaitGroup

	// Controle de mensagens
	messageQueue map[string]*Message
	queueMutex   sync.Mutex

	// Handlers de mensagem
	messageHandlers map[string]func(*Message)
}

// Cria um novo nó da rede
func NewRingNode(config *NodeConfig) (*RingNode, error) {
	// Resolve endereço do próximo nó
	nextAddr, err := net.ResolveUDPAddr("udp", config.NextNodeAddr)
	if err != nil {
		return nil, fmt.Errorf("erro ao resolver endereço do próximo nó: %v", err)
	}

	// Resolve endereço local
	listenAddr, err := net.ResolveUDPAddr("udp", config.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("erro ao resolver endereço local: %v", err)
	}

	// Cria conexão UDP
	conn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar conexão UDP: %v", err)
	}

	node := &RingNode{
		config:          config,
		state:           NODE_STATE_DISCONNECTED,
		conn:            conn,
		nextNodeAddr:    nextAddr,
		logger:          NewNetworkLogger(config.ID, config.UseColors),
		tokenCtrl:       NewTokenController(config.ID),
		stats:           &NodeStatistics{Uptime: time.Now()},
		incomingChan:    make(chan *Message, 100),
		outgoingChan:    make(chan *Message, 100),
		controlChan:     make(chan string, 10),
		running:         false,
		messageQueue:    make(map[string]*Message),
		messageHandlers: make(map[string]func(*Message)),
	}

	// Configura handlers padrão
	node.setupDefaultHandlers()

	return node, nil
}

// Configura handlers padrão de mensagem
func (rn *RingNode) setupDefaultHandlers() {
	rn.messageHandlers[MSG_TOKEN] = rn.handleTokenMessage
	rn.messageHandlers[MSG_HEARTBEAT] = rn.handleHeartbeatMessage
	rn.messageHandlers[MSG_ERROR] = rn.handleErrorMessage
	rn.messageHandlers[MSG_ACK] = rn.handleAckMessage
}

// Inicia o nó
func (rn *RingNode) Start() error {
	rn.mutex.Lock()
	defer rn.mutex.Unlock()

	if rn.running {
		return fmt.Errorf("nó já está em execução")
	}

	rn.running = true
	rn.state = NODE_STATE_CONNECTING

	rn.logger.LogOperationStart("Start Node")
	rn.logger.LogConnection(rn.config.ListenAddr, "LISTENING")

	// Inicia goroutines
	rn.wg.Add(4)
	go rn.receiveLoop()
	go rn.sendLoop()
	go rn.processLoop()
	go rn.heartbeatLoop()

	rn.state = NODE_STATE_CONNECTED
	rn.logger.LogNetworkState(rn.state, "Nó iniciado com sucesso")

	// Se for o nó 0, inicia com o token
	if rn.config.ID == 0 {
		go rn.initializeToken()
	}

	return nil
}

// Para o nó
func (rn *RingNode) Stop() {
	rn.mutex.Lock()
	defer rn.mutex.Unlock()

	if !rn.running {
		return
	}

	rn.logger.LogOperationStart("Stop Node")
	rn.running = false
	rn.state = NODE_STATE_DISCONNECTED

	// Sinaliza parada
	close(rn.controlChan)

	// Fecha conexão
	rn.conn.Close()

	// Espera goroutines terminarem
	rn.wg.Wait()

	// Fecha logger
	rn.logger.Close()

	rn.logger.LogOperationEnd("Stop Node", true)
}

// Loop de recebimento de mensagens
func (rn *RingNode) receiveLoop() {
	defer rn.wg.Done()

	buffer := make([]byte, BUFFER_SIZE)

	for rn.running {
		rn.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, addr, err := rn.conn.ReadFromUDP(buffer)

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if rn.running {
				rn.logger.LogError("ReadFromUDP", err)
				rn.stats.ErrorsCount++
			}
			continue
		}

		// Decodifica mensagem
		msg, err := DecodeMessage(buffer[:n])
		if err != nil {
			rn.logger.LogError("DecodeMessage", err)
			rn.stats.ErrorsCount++
			continue
		}

		// Valida mensagem
		if !msg.IsValid() {
			rn.logger.LogWarning("receiveLoop", "Mensagem inválida recebida")
			rn.stats.MessagesDropped++
			continue
		}

		rn.logger.LogMessageReceived(msg)
		rn.stats.MessagesReceived++
		rn.stats.LastActivity = time.Now()

		// Envia para processamento
		select {
		case rn.incomingChan <- msg:
		default:
			rn.logger.LogWarning("receiveLoop", "Canal de entrada cheio, descartando mensagem")
			rn.stats.MessagesDropped++
		}

		_ = addr // Para evitar warning de variável não utilizada
	}
}

// Loop de envio de mensagens
func (rn *RingNode) sendLoop() {
	defer rn.wg.Done()

	for rn.running {
		select {
		case msg := <-rn.outgoingChan:
			err := rn.sendMessage(msg)
			if err != nil {
				rn.logger.LogError("sendMessage", err)
				rn.stats.ErrorsCount++
			} else {
				rn.logger.LogMessageSent(msg)
				rn.stats.MessagesSent++
			}

		case <-rn.controlChan:
			return
		}
	}
}

// Loop de processamento de mensagens
func (rn *RingNode) processLoop() {
	defer rn.wg.Done()

	for rn.running {
		select {
		case msg := <-rn.incomingChan:
			rn.processMessage(msg)

		case <-rn.controlChan:
			return
		}
	}
}

// Loop de heartbeat
func (rn *RingNode) heartbeatLoop() {
	defer rn.wg.Done()

	ticker := time.NewTicker(HEARTBEAT_INTERVAL)
	defer ticker.Stop()

	for rn.running {
		select {
		case <-ticker.C:
			rn.sendHeartbeat()

		case <-rn.controlChan:
			return
		}
	}
}

// Envia mensagem via UDP
func (rn *RingNode) sendMessage(msg *Message) error {
	data, err := msg.Encode()
	if err != nil {
		return fmt.Errorf("erro ao codificar mensagem: %v", err)
	}

	_, err = rn.conn.WriteToUDP(data, rn.nextNodeAddr)
	if err != nil {
		return fmt.Errorf("erro ao enviar mensagem: %v", err)
	}

	return nil
}

// Processa mensagem recebida
func (rn *RingNode) processMessage(msg *Message) {
	// Incrementa hops
	msg.IncrementHops()

	// Verifica se deve descartar por excesso de hops
	if msg.ShouldDiscard(MAX_HOPS) {
		rn.logger.LogWarning("processMessage", "Mensagem descartada por excesso de hops")
		rn.stats.MessagesDropped++
		return
	}

	// Verifica se a mensagem é para este nó
	if msg.IsForNode(rn.config.ID) {
		rn.handleMessage(msg)
	} else {
		rn.forwardMessage(msg)
	}
}

// Manipula mensagem destinada a este nó
func (rn *RingNode) handleMessage(msg *Message) {
	rn.logger.LogMessageProcessed(msg, "HANDLING")

	// Chama handler específico
	if handler, exists := rn.messageHandlers[msg.Type]; exists {
		handler(msg)
	} else {
		rn.logger.LogWarning("handleMessage", fmt.Sprintf("Handler não encontrado para tipo: %s", msg.Type))
	}
}

// Encaminha mensagem para o próximo nó
func (rn *RingNode) forwardMessage(msg *Message) {
	rn.logger.LogMessageForwarded(msg)
	rn.stats.MessagesForwarded++

	select {
	case rn.outgoingChan <- msg:
	default:
		rn.logger.LogWarning("forwardMessage", "Canal de saída cheio")
		rn.stats.MessagesDropped++
	}
}

// Handlers específicos de mensagem

func (rn *RingNode) handleTokenMessage(msg *Message) {
	token, err := GetTokenFromMessage(msg)
	if err != nil {
		rn.logger.LogError("handleTokenMessage", err)
		return
	}

	err = rn.tokenCtrl.ReceiveToken(token)
	if err != nil {
		rn.logger.LogError("ReceiveToken", err)
		return
	}

	rn.logger.LogTokenReceived(token)
	rn.stats.TokenReceived++

	// Processa mensagens em espera
	rn.processWaitingMessages()

	// Passa o token após um tempo
	go rn.scheduleTokenPass()
}

func (rn *RingNode) handleHeartbeatMessage(msg *Message) {
	rn.logger.Debug("Heartbeat recebido de nó %d", msg.From)

	// Responde com ACK se solicitado
	if msg.To == rn.config.ID {
		ack := NewAckMessage(rn.config.ID, msg.From, msg.MessageID, "success")
		rn.SendMessage(ack)
	}
}

func (rn *RingNode) handleErrorMessage(msg *Message) {
	errorContent, err := msg.GetErrorContent()
	if err != nil {
		rn.logger.LogError("handleErrorMessage", err)
		return
	}

	rn.logger.Error("Erro recebido: %s - %s", errorContent.Description, errorContent.Details)
}

func (rn *RingNode) handleAckMessage(msg *Message) {
	ackContent, err := msg.GetAckContent()
	if err != nil {
		rn.logger.LogError("handleAckMessage", err)
		return
	}

	rn.logger.Debug("ACK recebido para mensagem %s: %s", ackContent.OriginalMessageID, ackContent.Status)
}

// Métodos públicos para envio de mensagens

func (rn *RingNode) SendMessage(msg *Message) error {
	if !rn.running {
		return fmt.Errorf("nó não está em execução")
	}

	msg.From = rn.config.ID

	// Se não tem token e mensagem não é de alta prioridade, adiciona à fila
	if !rn.tokenCtrl.HasToken() && msg.Priority != PRIORITY_HIGH {
		rn.tokenCtrl.AddToWaitingQueue(msg)
		rn.tokenCtrl.RequestToken()
		return nil
	}

	select {
	case rn.outgoingChan <- msg:
		return nil
	default:
		return fmt.Errorf("canal de saída cheio")
	}
}

func (rn *RingNode) SendGameMessage(to int, action string, data interface{}) error {
	msg := NewGameMessage(rn.config.ID, to, action, data)
	return rn.SendMessage(msg)
}

func (rn *RingNode) BroadcastMessage(content interface{}) error {
	msg := NewBroadcastMessage(rn.config.ID, content)
	return rn.SendMessage(msg)
}

// Métodos auxiliares

func (rn *RingNode) initializeToken() {
	time.Sleep(2 * time.Second) // Aguarda estabilização
	token := NewToken(rn.config.ID)
	rn.tokenCtrl.ReceiveToken(token)
	rn.logger.LogTokenReceived(token)
	rn.stats.TokenReceived++
}

func (rn *RingNode) scheduleTokenPass() {
	time.Sleep(1 * time.Second) // Tempo para processar mensagens

	nextNode := (rn.config.ID + 1) % rn.config.RingSize
	tokenMsg, err := rn.tokenCtrl.PassToken(nextNode)
	if err != nil {
		rn.logger.LogError("PassToken", err)
		return
	}

	rn.logger.LogTokenPassed(rn.tokenCtrl.token, nextNode)
	rn.stats.TokenPassed++

	rn.SendMessage(tokenMsg)
}

func (rn *RingNode) processWaitingMessages() {
	messages := rn.tokenCtrl.GetWaitingMessages()
	for _, msg := range messages {
		rn.SendMessage(msg)
	}
}

func (rn *RingNode) sendHeartbeat() {
	msg := NewMessage(MSG_HEARTBEAT, rn.config.ID, -1, "ping", PRIORITY_LOW)
	rn.SendMessage(msg)
}

// Getters para informações do nó

func (rn *RingNode) GetState() string {
	rn.mutex.RLock()
	defer rn.mutex.RUnlock()
	return rn.state
}

func (rn *RingNode) GetStatistics() *NodeStatistics {
	rn.mutex.RLock()
	defer rn.mutex.RUnlock()

	// Cria cópia das estatísticas
	statsCopy := *rn.stats
	return &statsCopy
}

func (rn *RingNode) GetTokenInfo() map[string]interface{} {
	return rn.tokenCtrl.GetTokenInfo()
}

func (rn *RingNode) HasToken() bool {
	return rn.tokenCtrl.HasToken()
}

// Registra handler customizado para tipo de mensagem
func (rn *RingNode) RegisterMessageHandler(msgType string, handler func(*Message)) {
	rn.messageHandlers[msgType] = handler
}

// Obtém estatísticas detalhadas
func (rn *RingNode) GetDetailedStats() map[string]interface{} {
	stats := rn.GetStatistics()
	tokenInfo := rn.GetTokenInfo()

	return map[string]interface{}{
		"node_id":            rn.config.ID,
		"state":              rn.GetState(),
		"uptime":             time.Since(stats.Uptime).Seconds(),
		"messages_sent":      stats.MessagesSent,
		"messages_received":  stats.MessagesReceived,
		"messages_forwarded": stats.MessagesForwarded,
		"messages_dropped":   stats.MessagesDropped,
		"errors_count":       stats.ErrorsCount,
		"token_received":     stats.TokenReceived,
		"token_passed":       stats.TokenPassed,
		"last_activity":      stats.LastActivity.Unix(),
		"token_info":         tokenInfo,
		"queue_size":         len(rn.incomingChan),
		"outgoing_size":      len(rn.outgoingChan),
	}
}
