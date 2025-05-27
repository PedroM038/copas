package network

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Network representa a configuração de rede de um jogador no anel.
type Network struct {
	PlayerID  int           // ID do jogador
	LocalAddr *net.UDPAddr  // Endereço local deste jogador
	NextAddr  *net.UDPAddr  // Endereço do próximo jogador no anel
	Conn      *net.UDPConn  // Conexão UDP
	ReceiveCh chan Message  // Canal para mensagens recebidas
	SendCh    chan Message  // Canal para mensagens a serem enviadas
	QuitCh    chan struct{} // Canal para encerrar goroutines
	HasToken  bool          // Indica se este jogador possui o token
	TokenCh   chan Token    // Canal específico para recebimento de token
}

// NewNetwork inicializa a rede para o jogador.
func NewNetwork(playerID int, localAddr, nextAddr string) (*Network, error) {
	lAddr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		return nil, fmt.Errorf("erro ao resolver endereço local: %w", err)
	}
	nAddr, err := net.ResolveUDPAddr("udp", nextAddr)
	if err != nil {
		return nil, fmt.Errorf("erro ao resolver endereço do próximo: %w", err)
	}
	conn, err := net.ListenUDP("udp", lAddr)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir conexão UDP: %w", err)
	}

	netw := &Network{
		PlayerID:  playerID,
		LocalAddr: lAddr,
		NextAddr:  nAddr,
		Conn:      conn,
		ReceiveCh: make(chan Message, 8),
		SendCh:    make(chan Message, 8),
		QuitCh:    make(chan struct{}),
		HasToken:  playerID == 0, // Jogador 0 inicia com o token
		TokenCh:   make(chan Token, 1),
	}

	go netw.listen()
	go netw.sender()
	go netw.messageRouter()

	return netw, nil
}

// listen escuta por mensagens UDP e as envia para ReceiveCh.
func (n *Network) listen() {
	buf := make([]byte, 4096)
	for {
		select {
		case <-n.QuitCh:
			return
		default:
			n.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			nBytes, _, err := n.Conn.ReadFromUDP(buf)
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					continue
				}
				continue
			}
			msg, err := DecodeMessage(buf[:nBytes])
			if err == nil {
				n.ReceiveCh <- msg
			}
		}
	}
}

// sender envia mensagens do canal SendCh para o próximo jogador.
func (n *Network) sender() {
	for {
		select {
		case <-n.QuitCh:
			return
		case msg := <-n.SendCh:
			data, err := EncodeMessage(msg)
			if err != nil {
				continue
			}
			n.Conn.WriteToUDP(data, n.NextAddr)
		}
	}
}

// messageRouter roteia mensagens recebidas
func (n *Network) messageRouter() {
	for {
		select {
		case <-n.QuitCh:
			return
		case msg := <-n.ReceiveCh:
			n.handleMessage(msg)
		}
	}
}

func (n *Network) handleMessage(msg Message) {
	// Se a mensagem já deu a volta completa (voltou para o remetente), descartar
	if msg.From == n.PlayerID {
		return
	}

	// Se é uma mensagem de token, processar especialmente
	if msg.Type == MessageToken {
		n.handleTokenMessage(msg)
		return
	}

	// Se a mensagem é para este jogador, processar localmente
	if msg.To == n.PlayerID || msg.To == -1 { // -1 para broadcast
		n.ReceiveCh <- msg // ✅ Entrega ao handler do jogo
	}

	// Se não é para este jogador ou é broadcast, retransmitir
	if msg.To != n.PlayerID {
		n.Send(msg)
	}
}

// handleTokenMessage processa mensagens de token
func (n *Network) handleTokenMessage(msg Message) {
	token, err := ParseTokenPayload(msg.Payload)
	if err != nil {
		return
	}

	// Se o token é para este jogador
	if token.HolderID == n.PlayerID {
		n.HasToken = true
		n.TokenCh <- token
	} else {
		// Retransmitir o token
		n.Send(msg)
	}
}

// Send envia uma mensagem para o próximo jogador.
func (n *Network) Send(msg Message) {
	n.SendCh <- msg
}

// Receive retorna o canal de mensagens recebidas.
func (n *Network) Receive() <-chan Message {
	return n.ReceiveCh
}

// PassToken passa o token para o próximo jogador
func (n *Network) PassToken() error {
	if !n.HasToken {
		return fmt.Errorf("jogador %d não possui o token", n.PlayerID)
	}

	nextPlayerID := (n.PlayerID + 1) % 4 // Assumindo 4 jogadores
	msg, err := NewTokenMessage(n.PlayerID, nextPlayerID, nextPlayerID)
	if err != nil {
		return err
	}

	n.Send(msg)
	n.HasToken = false
	return nil
}

// WaitForToken aguarda receber o token
func (n *Network) WaitForToken() Token {
	return <-n.TokenCh
}

// HasTokenNow verifica se o jogador possui o token atualmente
func (n *Network) HasTokenNow() bool {
	return n.HasToken
}

// BroadcastMessage envia uma mensagem para todos os jogadores (broadcast)
func (n *Network) BroadcastMessage(msgType MessageType, payload json.RawMessage) {
	msg := Message{
		Type:    msgType,
		From:    n.PlayerID,
		To:      -1, // -1 indica broadcast
		Payload: payload,
	}
	n.Send(msg)
}

// Close encerra a conexão de rede.
func (n *Network) Close() {
	close(n.QuitCh)
	n.Conn.Close()
}
